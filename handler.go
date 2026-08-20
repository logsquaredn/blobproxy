package blobproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/logsquaredn/blobproxy/internal/httputil"
	"github.com/logsquaredn/blobproxy/internal/log"
	"gocloud.dev/blob"
)

const (
	useSignedURLsParamKey = "use_signed_urls"
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
	Bucket        *blob.Bucket
	UseSignedURLs bool
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
		signedURL, err := h.Bucket.SignedURL(r.Context(), key, &blob.SignedURLOptions{
			ContentType: r.Header.Get("Content-Type"),
		})
		if err != nil {
			http.Error(w, err.Error(), httputil.HTTPStatusCode(err))
			log.Error(err.Error())
			return
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
