package bucketfs

import (
	"io/fs"

	"gocloud.dev/blob"
)

type File struct {
	Path string
	*blob.Reader
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f, nil
}
