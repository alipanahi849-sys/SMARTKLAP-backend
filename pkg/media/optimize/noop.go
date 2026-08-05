package optimize

import (
	"context"
	"io"
	"strings"
)

// Noop stores uploads unchanged when ffmpeg is unavailable.
type Noop struct{}

func (Noop) OptimizeImage(_ context.Context, src io.Reader, inputExt string, _ ImageProfile) (*PreparedMedia, error) {
	ext := normalizeExt(inputExt)
	return stageOriginal(src, ext, mimeForExt(ext))
}

func (Noop) OptimizeVideo(_ context.Context, src io.Reader, inputExt string) (*PreparedMedia, error) {
	ext := normalizeExt(inputExt)
	return stageOriginal(src, ext, mimeForExt(ext))
}

func (Noop) VideoThumbnail(context.Context, string) (*PreparedMedia, error) {
	return nil, nil
}

func mimeForExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	default:
		return "video/mp4"
	}
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
