package command

import (
	"context"
	"io"
	stdlog "log"
	"log/slog"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/logsquaredn/blobproxy"
	"github.com/logsquaredn/blobproxy/internal/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func New() *cobra.Command {
	var (
		addr     string
		certFile string
		keyFile  string
		logCfg   = new(log.Config)
		cmd      = &cobra.Command{
			Use:           "blobproxy [--addr|-a 127.0.0.1:80] [--tls-key tls.key --tls-crt tls.crt] {s3|azblob|gs}://bucket [/prefix]",
			Args:          cobra.RangeArgs(1, 2),
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := log.SloggerInto(cmd.Context(), slog.New(slog.NewTextHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
					Level: logCfg,
				})))
				prefix := "/"
				log.SetLogger(log.SloggerFrom(ctx))

				if len(args) > 1 {
					prefix = path.Clean(args[1])
					if prefix == "." {
						prefix = "/"
					}
				}

				lis, err := net.Listen("tcp", addr)
				if err != nil {
					return err
				}

				handler, err := blobproxy.New(ctx, args[0])
				if err != nil {
					return err
				}
				defer handler.Close()

				srv := &http.Server{
					ReadHeaderTimeout: time.Second * 5,
					BaseContext: func(_ net.Listener) context.Context {
						return ctx
					},
					ErrorLog: stdlog.New(io.Discard, "", 0),
					Handler:  http.StripPrefix(prefix, handler),
				}

				eg, ctx := errgroup.WithContext(cmd.Context())

				eg.Go(func() error {
					<-ctx.Done()
					if err = srv.Shutdown(context.WithoutCancel(ctx)); err != nil {
						return err
					}
					return ctx.Err()
				})

				eg.Go(func() error {
					log.Info("listening...", "addr", lis.Addr().String())

					if certFile != "" {
						return srv.ServeTLS(lis, certFile, keyFile)
					}

					return srv.Serve(lis)
				})

				return eg.Wait()
			},
		}
	)

	logCfg.AddFlags(cmd.Flags())

	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }}")
	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "Listen address")

	cmd.Flags().StringVar(&certFile, "tls-crt", "", "TLS certificate file")
	cmd.Flags().StringVar(&keyFile, "tls-key", "", "TLS private key file")
	cmd.MarkFlagFilename("tls-crt")
	cmd.MarkFlagFilename("tls-key")
	cmd.MarkFlagsRequiredTogether("tls-crt", "tls-key")

	return cmd
}
