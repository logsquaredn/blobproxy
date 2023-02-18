package bucketfs

import (
	"context"
	"io/fs"

	"github.com/logsquaredn/blobproxy"
	"gocloud.dev/blob"
)

type ContextFS interface {
	fs.FS
	Context() context.Context
	WithContext(context.Context) ContextFS
}

func NewFS(bucket *blob.Bucket) ContextFS {
	return &FS{context.Background(), bucket}
}

type FS struct {
	ctx context.Context
	*blob.Bucket
}

func (f *FS) Open(name string) (fs.File, error) {
	_ = blobproxy.LoggerFrom(f.ctx)

	if name == "." {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	reader, err := f.NewReader(f.ctx, name, nil)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	return &File{name, reader}, nil
}

func (f *FS) WithContext(ctx context.Context) ContextFS {
	f.ctx = ctx
	return f
}

func (f *FS) Context() context.Context {
	return f.ctx
}
