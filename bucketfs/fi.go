package bucketfs

import "io/fs"

func (f *File) Name() string {
	return f.Path
}

func (f *File) IsDir() bool {
	return false
}

func (f *File) Mode() fs.FileMode {
	return fs.ModeNamedPipe
}

func (f *File) Sys() any {
	return f.Reader
}
