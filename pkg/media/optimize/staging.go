package optimize

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func stageOriginal(src io.Reader, ext, contentType string) (*PreparedMedia, error) {
	dir, err := os.MkdirTemp("", "clap-upload-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	path := filepath.Join(dir, "original"+ext)
	f, err := os.Create(path)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	size, err := io.Copy(f, src)
	closeErr := f.Close()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("buffer upload: %w", err)
	}
	if closeErr != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("close temp file: %w", closeErr)
	}

	return &PreparedMedia{
		Path:        path,
		Size:        size,
		ContentType: contentType,
		Extension:   ext,
		cleanup:     func() { _ = os.RemoveAll(dir) },
	}, nil
}

func openFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func pickSmaller(originalPath string, originalSize int64, originalExt, originalMIME string, optimizedPath, optimizedExt, optimizedMIME string) (*PreparedMedia, error) {
	optInfo, err := os.Stat(optimizedPath)
	if err != nil {
		return nil, err
	}
	if optInfo.Size() >= originalSize {
		_ = os.Remove(optimizedPath)
		return &PreparedMedia{
			Path:        originalPath,
			Size:        originalSize,
			ContentType: originalMIME,
			Extension:   originalExt,
		}, nil
	}

	_ = os.Remove(originalPath)
	return &PreparedMedia{
		Path:        optimizedPath,
		Size:        optInfo.Size(),
		ContentType: optimizedMIME,
		Extension:   optimizedExt,
	}, nil
}
