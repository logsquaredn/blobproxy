package bucketfs

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/logsquaredn/blobproxy"
)

func NewFileServer(fs ContextFS) http.Handler {
	log := blobproxy.LoggerFrom(fs.Context())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rLog := log.WithValues("request", uuid.NewString())
		rLog.Info(r.Method + " /" + r.URL.Path)
		r = r.WithContext(blobproxy.WithLogger(r.Context(), rLog))
		http.FileServer(http.FS(fs.WithContext(r.Context()))).ServeHTTP(w, r)
	})
}
