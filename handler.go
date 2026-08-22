package blobproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/logsquaredn/blobproxy/internal/httputil"
	"github.com/logsquaredn/blobproxy/internal/log"
	"gocloud.dev/blob"
)

const (
	useSignedURLsParamKey   = "use_signed_urls"
	signedURLExpiryParamKey = "signed_url_expiry"
)

func New(ctx context.Context, addr string) (http.Handler, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	h := &Handler{}

	if useSignedURLsParam := q.Get(useSignedURLsParamKey); useSignedURLsParam != "" {
		q.Del(useSignedURLsParamKey)
		u.RawQuery = q.Encode()
		if h.UseSignedURLs, err = strconv.ParseBool(useSignedURLsParam); err != nil {
			return nil, err
		}
	}

	if signedURLExpiryParam := q.Get(signedURLExpiryParamKey); signedURLExpiryParam != "" {
		q.Del(signedURLExpiryParamKey)
		u.RawQuery = q.Encode()
		if h.SignedURLExpiry, err = time.ParseDuration(signedURLExpiryParam); err != nil {
			return nil, err
		} else if h.SignedURLExpiry <= 0 {
			return nil, fmt.Errorf("%s must be positive", signedURLExpiryParamKey)
		}
	}

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
	key := strings.TrimPrefix(r.URL.Path, "/")

	if h.UseSignedURLs {
		expiry := h.SignedURLExpiry
		if expiry <= 0 {
			expiry = blob.DefaultSignedURLExpiry
		}

		signedURL, err := h.Bucket.SignedURL(ctx, key, &blob.SignedURLOptions{
			Expiry:      expiry,
			ContentType: r.Header.Get("Content-Type"),
		})
		if err != nil {
			http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
			log.Error(err.Error())
			return
		}

		// A redirect is not cacheable unless it says so, and an uncacheable one
		// makes the object uncacheable too: every request is answered with a
		// freshly signed URL, and the client's cache is keyed on that URL, so it
		// can never reuse bytes it already holds. Letting clients reuse the
		// redirect gives them a stable key to cache the object under.
		//
		// The redirect must go stale before the URL it points at does, or a
		// client would confidently follow a cached redirect to an expired
		// signature. The remaining tenth of the lifetime is headroom for the
		// request itself.
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int((expiry-expiry/10).Seconds())))
		}

		http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
		return
	}

	attr, err := h.Bucket.Attributes(ctx, key)
	if err != nil {
		http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
		log.Error(err.Error())
		return
	}

	w.Header().Set("Cache-Control", attr.CacheControl)
	w.Header().Set("Content-Disposition", attr.ContentDisposition)
	w.Header().Set("Content-Encoding", attr.ContentEncoding)
	w.Header().Set("Content-Language", attr.ContentLanguage)
	w.Header().Set("Content-Type", attr.ContentType)
	w.Header().Set("Content-Length", fmt.Sprint(attr.Size))

	rc, err := h.Bucket.NewReader(ctx, key, &blob.ReaderOptions{})
	if err != nil {
		http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
		log.Error(err.Error())
		return
	}
	defer rc.Close()

	_, _ = io.Copy(w, rc)
}

func (h *Handler) Close() error {
	if h.Bucket != nil {
		return h.Bucket.Close()
	}
	return nil
}
