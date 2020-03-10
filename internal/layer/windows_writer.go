package layer

import (
	"archive/tar"
	"io"
)

type WindowsWriter struct {
	tarWriter *tar.Writer
}

func NewWindowsWriter(dataWriter io.WriteCloser) *WindowsWriter {
	return &WindowsWriter{tar.NewWriter(dataWriter)}
}

func (w WindowsWriter) Write(content []byte) (int, error) {
	return -1, nil
}

func (w WindowsWriter) WriteHeader(header *tar.Header) error {
	return nil
}

func (w WindowsWriter) Close() error {
	return nil
}
