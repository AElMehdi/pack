package layer

import (
	"archive/tar"
	"io"
)

type LinuxWriter struct {
	tarWriter *tar.Writer
}

func NewLinuxWriter(dataWriter io.WriteCloser) *LinuxWriter {
	return &LinuxWriter{tar.NewWriter(dataWriter)}
}

func (w LinuxWriter) Write(content []byte) (int, error) {
	return w.tarWriter.Write(content)
}

func (w LinuxWriter) WriteHeader(header *tar.Header) error {
	return w.tarWriter.WriteHeader(header)
}

func (w LinuxWriter) Close() error {
	return w.tarWriter.Close()
}
