package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalProvider stores files on disk for development / when R2 is not configured.
type LocalProvider struct {
	root      string
	publicURL string // e.g. "/uploads" — prepended to keys in GenerateSignedURL
}

// NewLocalProvider creates a disk-backed storage provider under root.
// publicURL is the HTTP prefix used to fetch files (served by the API).
func NewLocalProvider(root, publicURL string) (*LocalProvider, error) {
	if root == "" {
		root = "./uploads"
	}
	if publicURL == "" {
		publicURL = "/uploads"
	}
	publicURL = strings.TrimRight(publicURL, "/")

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return &LocalProvider{root: root, publicURL: publicURL}, nil
}

func (l *LocalProvider) absPath(key string) (string, error) {
	clean := filepath.Clean("/" + key)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", fmt.Errorf("invalid storage key")
	}
	full := filepath.Join(l.root, clean)
	// Prevent path traversal outside root.
	rel, err := filepath.Rel(l.root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	return full, nil
}

func (l *LocalProvider) Upload(_ context.Context, key string, reader io.Reader, _ string, _ int64) error {
	full, err := l.absPath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	f, err := os.OpenFile(full, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}
	return nil
}

func (l *LocalProvider) Delete(_ context.Context, key string) error {
	full, err := l.absPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local file: %w", err)
	}
	return nil
}

func (l *LocalProvider) GenerateSignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	clean := strings.TrimPrefix(filepath.Clean("/"+key), "/")
	return l.publicURL + "/" + clean, nil
}

func (l *LocalProvider) Exists(_ context.Context, key string) (bool, error) {
	full, err := l.absPath(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(full)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *LocalProvider) Get(_ context.Context, key string) (io.ReadCloser, error) {
	full, err := l.absPath(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	return f, nil
}

// Root returns the on-disk root directory (for static file serving).
func (l *LocalProvider) Root() string {
	return l.root
}
