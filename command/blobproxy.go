package command

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/logsquaredn/blobproxy"
	"github.com/logsquaredn/blobproxy/bucketfs"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func New() *cobra.Command {
	var (
		verbosity int
		port      int64
		cmd       = &cobra.Command{
			Use:           "blobproxy {s3|azblob|gs}://bucket [/prefix] [--s3-endpoint=http://minio:9000/] [--s3-force-path-style] [--s3-disable-ssl]",
			Args:          cobra.RangeArgs(1, 2),
			Version:       blobproxy.GetSemver(),
			SilenceErrors: true,
			SilenceUsage:  true,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				cmd.SetContext(blobproxy.WithLogger(cmd.Context(), blobproxy.NewLogger().V(2-verbosity)))
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx    = cmd.Context()
					log    = blobproxy.LoggerFrom(ctx)
					prefix = "/"
				)

				if len(args) > 1 {
					prefix = path.Clean(args[1])
					if prefix == "." {
						prefix = "/"
					}
				}

				addr, err := url.Parse(args[0])
				if err != nil {
					return err
				}

				bucket, err := blob.OpenBucket(ctx, addr.String())
				if err != nil {
					return err
				}
				defer bucket.Close()

				if accessible, err := bucket.IsAccessible(ctx); !accessible || err != nil {
					return fmt.Errorf("inaccessible bucket %s", addr.String())
				}

				l, err := net.Listen("tcp", fmt.Sprint(":", port))
				if err != nil {
					return err
				}

				log.Info("serving " + addr.String() + " at " + l.Addr().String() + prefix)

				//nolint:gosec
				return http.Serve(l, http.StripPrefix(prefix, bucketfs.NewFileServer(bucketfs.NewFS(bucket).WithContext(ctx))))
			},
		}
	)

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} " + runtime.Version() + "\n")
	cmd.PersistentFlags().CountVarP(&verbosity, "verbose", "V", "verbose")
	cmd.Flags().Int64VarP(&port, "port", "p", mustParsePort(), "port")

	return cmd
}

func mustParsePort() int64 {
	p, err := strconv.Atoi(os.Getenv("PORT"))
	if p != 0 && err != nil {
		return int64(p)
	}

	return 8080
}
