package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/logsquaredn/blobproxy/command"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if err := command.New().ExecuteContext(ctx); err != nil {
		stop()
		os.Stdout.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	stop()
	os.Exit(0)
}
