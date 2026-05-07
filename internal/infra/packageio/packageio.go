// Package packageio reads and writes .avm.zip packages: zip layout,
// manifest parsing, checksum verification, and path safety.
package packageio

import (
	"errors"
	"io"

	"github.com/xz1220/agent-vm/internal/app/model"
)

// IO is the contract Application layer uses for package files.
type IO interface {
	Read(path string) (*model.PackageManifest, ReadHandle, error)
	Write(manifest *model.PackageManifest, payload io.Reader, dst string) error
	Verify(path string) error
}

// ReadHandle exposes lazy access to package contents after Read returns
// the manifest. Caller must Close.
type ReadHandle interface {
	Open(name string) (io.ReadCloser, error)
	Files() []string
	Close() error
}

// Default is the in-process default implementation.
type Default struct{}

func New() *Default { return &Default{} }

func (Default) Read(path string) (*model.PackageManifest, ReadHandle, error) {
	return nil, nil, errors.New("packageio: Read not yet implemented")
}

func (Default) Write(manifest *model.PackageManifest, payload io.Reader, dst string) error {
	return errors.New("packageio: Write not yet implemented")
}

func (Default) Verify(path string) error {
	return errors.New("packageio: Verify not yet implemented")
}
