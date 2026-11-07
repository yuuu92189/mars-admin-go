package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mars-admin-go/internal/modules/storage/domain"
)

type Backend struct {
	root      string
	publicURL string
}

func New(root, publicURL string) (*Backend, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage root: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, fmt.Errorf("create local storage root: %w", err)
	}
	return &Backend{root: absolute, publicURL: strings.TrimRight(publicURL, "/")}, nil
}

func (b *Backend) Name() string { return "local" }

func (b *Backend) Put(ctx context.Context, key string, body io.Reader, size int64, _ string) (domain.ObjectInfo, error) {
	path, err := b.path(key)
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return domain.ObjectInfo{}, fmt.Errorf("create object directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	written, err := io.Copy(tmp, body)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	if size > 0 && written != size {
		return domain.ObjectInfo{}, errors.New("stored object size differs from declared size")
	}
	if err := ctx.Err(); err != nil {
		return domain.ObjectInfo{}, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return domain.ObjectInfo{}, fmt.Errorf("commit local object: %w", err)
	}
	return domain.ObjectInfo{Size: written}, nil
}

func (b *Backend) Get(_ context.Context, key string) (io.ReadCloser, domain.ObjectInfo, error) {
	path, err := b.path(key)
	if err != nil {
		return nil, domain.ObjectInfo{}, err
	}
	file, err := os.Open(path) // #nosec G304 -- path was canonicalized and constrained to root.
	if err != nil {
		return nil, domain.ObjectInfo{}, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, domain.ObjectInfo{}, err
	}
	return file, domain.ObjectInfo{Size: info.Size()}, nil
}

func (b *Backend) Delete(_ context.Context, key string) error {
	path, err := b.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (b *Backend) Exists(_ context.Context, key string) (bool, error) {
	path, err := b.path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (b *Backend) URL(key string) string {
	return b.publicURL + "/" + strings.TrimLeft(filepath.ToSlash(key), "/")
}

func (b *Backend) path(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("invalid storage key")
	}
	full := filepath.Join(b.root, clean)
	relative, err := filepath.Rel(b.root, full)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("storage key escapes root")
	}
	return full, nil
}
