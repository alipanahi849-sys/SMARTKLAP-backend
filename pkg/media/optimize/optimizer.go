package optimize

import (
	"context"
	"io"
)

// ImageProfile tunes output dimensions for different upload surfaces.
type ImageProfile string

const (
	ImageProfileFeed    ImageProfile = "feed"
	ImageProfileAvatar  ImageProfile = "avatar"
	ImageProfileProduct ImageProfile = "product"
)

// PreparedMedia is a temp-file-backed upload payload. Call Cleanup when done.
type PreparedMedia struct {
	Path        string
	Size        int64
	ContentType string
	Extension   string
	cleanup     func()
}

func (p *PreparedMedia) Open() (io.ReadCloser, error) {
	if p == nil {
		return nil, nil
	}
	return openFile(p.Path)
}

func (p *PreparedMedia) Cleanup() {
	if p != nil && p.cleanup != nil {
		p.cleanup()
	}
}

// Optimizer losslessly shrinks uploaded images and videos when possible.
type Optimizer interface {
	OptimizeImage(ctx context.Context, src io.Reader, inputExt string, profile ImageProfile) (*PreparedMedia, error)
	OptimizeVideo(ctx context.Context, src io.Reader, inputExt string) (*PreparedMedia, error)
	// VideoThumbnail extracts a poster frame from a local video file path.
	// Returns nil, nil when thumbnails are not supported.
	VideoThumbnail(ctx context.Context, videoPath string) (*PreparedMedia, error)
}
