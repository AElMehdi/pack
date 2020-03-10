package layer

import (
	"archive/tar"
	"io"

	"github.com/buildpacks/imgutil"
)

//go:generate mockgen -package testmocks -destination testmocks/mock_layer_factory.go github.com/buildpacks/pack/internal/layers Factory
type Factory interface {
	NewWriter(fileWriter io.WriteCloser) Writer
}

//go:generate mockgen -package testmocks -destination testmocks/mock_layer_writer.go github.com/buildpacks/pack/internal/layers Writer
type Writer interface {
	Write(content []byte) (int, error)
	WriteHeader(header *tar.Header) error
	Close() error
}

type factory struct {
	os string
}

func NewFactory(image imgutil.Image) (Factory, error) {
	os, err := image.OS()
	if err != nil {
		return nil, err
	}
	return &factory{os}, nil
}

func (f *factory) NewWriter(fileWriter io.WriteCloser) Writer {
	if f.os == "windows" {
		return NewWindowsWriter(fileWriter)
	}
	return NewLinuxWriter(fileWriter)
}
