package bucketfs

import (
	"net/http"
	"path"

	"github.com/google/uuid"
	"github.com/logsquaredn/blobproxy"
)

func NewFileServer(fs ContextFS) http.Handler {
	log := blobproxy.LoggerFrom(fs.Context())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rLog := log.WithValues("request", uuid.NewString())
		r = r.WithContext(blobproxy.WithLogger(r.Context(), rLog))
		http.FileServer(http.FS(fs.WithContext(r.Context()))).ServeHTTP(w, r)
		rLog.Info("request for file", "status", r.Response.StatusCode, "key", path.Clean(r.URL.Path))
	})
}
