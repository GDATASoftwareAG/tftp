package secfs

import (
	"io/fs"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
)

type fileSystem struct {
	rootPath string
	fs       fs.FS
}

func New(rootPath string) fs.FS {
	return &fileSystem{
		rootPath: rootPath,
		fs:       os.DirFS(rootPath),
	}
}

func (f fileSystem) Open(name string) (fs.File, error) {
	path, err := securejoin.SecureJoin(f.rootPath, name)
	if err != nil {
		return nil, &os.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}
	if path, err = filepath.Rel(f.rootPath, path); err != nil {
		return nil, &os.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}

	return f.fs.Open(path)
}
