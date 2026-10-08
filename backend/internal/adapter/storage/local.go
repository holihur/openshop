// Package storage provides object-storage adapters. All of them satisfy
// port.ObjectStorage, so the application never binds to a vendor SDK.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// Local writes objects to a directory. It is intended for development only:
// horizontally scaled replicas do not share local disk, so production must use
// the S3 driver (or any other port implementation backed by shared storage).
type Local struct {
	dir       string
	publicURL string
}

func NewLocal(cfg config.StorageConfig) (*Local, error) {
	if err := os.MkdirAll(cfg.LocalDir, 0o750); err != nil {
		return nil, fmt.Errorf("storage local mkdir: %w", err)
	}
	return &Local{dir: cfg.LocalDir, publicURL: strings.TrimRight(cfg.PublicURL, "/")}, nil
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) (string, error) {
	// l.path rejects any key that escapes the storage root.
	path, err := l.path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", err
	}
	f, err := os.Create(path) // #nosec G304 -- path is validated by l.path
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return l.URL(context.Background(), key, 0)
}

func (l *Local) Delete(_ context.Context, key string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *Local) URL(_ context.Context, key string, _ time.Duration) (string, error) {
	return l.publicURL + "/" + strings.TrimLeft(key, "/"), nil
}

// path resolves a key inside the storage root, rejecting traversal attempts.
func (l *Local) path(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	p := filepath.Join(l.dir, clean)
	if !strings.HasPrefix(p, filepath.Clean(l.dir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return p, nil
}

var _ port.ObjectStorage = (*Local)(nil)
