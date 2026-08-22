package blobproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	xslices "github.com/frantjc/x/slices"
	"github.com/google/uuid"
	"github.com/logsquaredn/blobproxy/internal/httputil"
	"github.com/logsquaredn/blobproxy/internal/log"
	"gocloud.dev/blob"
)

const (
	useSignedURLsParamKey   = "use_signed_urls"
	signedURLExpiryParamKey = "signed_url_expiry"
	allowMethodParamKey     = "allow_method"
)

var (
	DefaultAllowedMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
	}
	SupportedMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
	}
	SupportedUseSignedURLMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
	}
)

type HandleCloser interface {
	http.Handler
	io.Closer
}

func New(ctx context.Context, addr string) (HandleCloser, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	h := &Handler{
		AllowedMethods: DefaultAllowedMethods,
	}

	if useSignedURLsParam := q.Get(useSignedURLsParamKey); useSignedURLsParam != "" {
		q.Del(useSignedURLsParamKey)
		if h.UseSignedURLs, err = strconv.ParseBool(useSignedURLsParam); err != nil {
			return nil, err
		}
	}

	if allowedMethods := q[allowMethodParamKey]; len(allowedMethods) > 0 {
		q.Del(allowMethodParamKey)
		h.AllowedMethods = xslices.Map(allowedMethods, func(allowedMethod string, _ int) string {
			return strings.ToUpper(allowedMethod)
		})
		supportedMethods := SupportedMethods
		if h.UseSignedURLs {
			supportedMethods = SupportedUseSignedURLMethods
		}
		for _, allowedMethod := range h.AllowedMethods {
			if !slices.Contains(supportedMethods, allowedMethod) {
				return nil, fmt.Errorf("unsupported HTTP method %s with %s=%t", allowedMethod, useSignedURLsParamKey, h.UseSignedURLs)
			}
		}
	}

	if rawSignedURLExpiry := q.Get(signedURLExpiryParamKey); rawSignedURLExpiry != "" {
		q.Del(signedURLExpiryParamKey)
		if h.SignedURLExpiry, err = time.ParseDuration(rawSignedURLExpiry); err != nil {
			return nil, err
		}
		if h.SignedURLExpiry < 0 {
			return nil, fmt.Errorf("%s must not be negative", signedURLExpiryParamKey)
		}
	}

	u.RawQuery = q.Encode()

	if h.Bucket, err = blob.OpenBucket(ctx, u.String()); err != nil {
		return nil, err
	}

	if accessible, err := h.Bucket.IsAccessible(ctx); err != nil {
		return nil, err
	} else if !accessible {
		return nil, fmt.Errorf("inaccessible bucket")
	}

	return h, nil
}

type Handler struct {
	AllowedMethods  []string
	Bucket          *blob.Bucket
	UseSignedURLs   bool
	SignedURLExpiry time.Duration
}

var (
	_ http.Handler = new(Handler)
)

// ServeHTTP implements [http.Handler].
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := log.SloggerFrom(ctx).With("request", uuid.NewString())
	log.Info(r.Method + " " + r.URL.Path)

	if !slices.Contains(h.AllowedMethods, r.Method) {
		rawErr := fmt.Sprintf("%d method not allowed", http.StatusMethodNotAllowed)
		w.Header().Set("Allow", strings.Join(h.AllowedMethods, ", "))
		http.Error(w, rawErr, http.StatusMethodNotAllowed)
		log.Error(rawErr)
		return
	}

	// Handled locally rather than sent to the bucket.
	// This is useful for CORS preflight requests.
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/")

	if h.UseSignedURLs {
		if r.Method == http.MethodHead {
			attr, err := h.Bucket.Attributes(ctx, key)
			if err != nil {
				http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
				log.Error(err.Error())
				return
			}

			setBlobHeaders(w, attr)
			return
		}

		expiry := h.SignedURLExpiry
		if expiry == 0 {
			expiry = blob.DefaultSignedURLExpiry
		}
		expires := time.Now().Add(expiry)

		signedURL, err := h.Bucket.SignedURL(r.Context(), key, &blob.SignedURLOptions{
			Expiry:      expiry,
			ContentType: r.Header.Get("Content-Type"),
			Method:      r.Method,
		})
		if err != nil {
			http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
			log.Error(err.Error())
			return
		}

		w.Header().Set("Expires", expires.UTC().Format(http.TimeFormat))
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", expiry/time.Second))

		http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
		return
	}

	attr, err := h.Bucket.Attributes(ctx, key)
	if err != nil {
		http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
		log.Error(err.Error())
		return
	}

	setBlobHeaders(w, attr)

	// HEAD should return the same headers as GET but no response body.
	if r.Method == http.MethodHead {
		return
	}

	rc, err := h.Bucket.NewReader(ctx, key, &blob.ReaderOptions{})
	if err != nil {
		http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
		log.Error(err.Error())
		return
	}
	defer rc.Close()

	_, _ = io.Copy(w, rc)
}

func setBlobHeaders(w http.ResponseWriter, attr *blob.Attributes) {
	for k, v := range map[string]string{
		"Cache-Control":       attr.CacheControl,
		"Content-Disposition": attr.ContentDisposition,
		"Content-Encoding":    attr.ContentEncoding,
		"Content-Language":    attr.ContentLanguage,
		"Content-Type":        attr.ContentType,
		"Content-Length":      fmt.Sprint(attr.Size),
	} {
		if v != "" {
			w.Header().Set(k, v)
		}
	}
}

func (h *Handler) Close() error {
	if h.Bucket != nil {
		return h.Bucket.Close()
	}
	return nil
}
