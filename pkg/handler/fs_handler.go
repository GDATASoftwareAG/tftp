package handler

import (
	"io"
	"io/fs"

	"github.com/gdatasoftwareag/tftp/internal/logging"
	"github.com/gdatasoftwareag/tftp/pkg/tftp"
)

func NewFSHandler(fileSystem fs.FS, logger logging.Logger) tftp.Handler {
	return &fsHandler{
		logger:     logger,
		fileSystem: fileSystem,
	}
}

type fsHandler struct {
	logger     logging.Logger
	fileSystem fs.FS
}

func (f fsHandler) Matches(_ string) bool {
	return true
}

func (f fsHandler) Reader(filePath string) (io.ReadCloser, int64, error) {
	file, err := f.fileSystem.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	var stats fs.FileInfo
	if stats, err = file.Stat(); err != nil {
		_ = file.Close()
		return nil, 0, err
	}

	return file, stats.Size(), nil
}
