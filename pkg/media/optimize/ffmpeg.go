package optimize

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ffmpegBin = "ffmpeg"

// FFmpeg uses ffmpeg for near-lossless re-encoding of images and videos.
type FFmpeg struct {
	timeout time.Duration
}

func NewFFmpeg() *FFmpeg {
	return &FFmpeg{timeout: 10 * time.Minute}
}

func (f *FFmpeg) Available() bool {
	_, err := exec.LookPath(ffmpegBin)
	return err == nil
}

func (f *FFmpeg) OptimizeImage(ctx context.Context, src io.Reader, inputExt string, profile ImageProfile) (*PreparedMedia, error) {
	ext := normalizeExt(inputExt)
	dir, err := os.MkdirTemp("", "clap-img-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(dir) }

	inputPath := filepath.Join(dir, "input"+ext)
	if err = writeReaderToFile(inputPath, src); err != nil {
		cleanup()
		return nil, err
	}

	inInfo, err := os.Stat(inputPath)
	if err != nil {
		cleanup()
		return nil, err
	}

	outputPath := filepath.Join(dir, "output.webp")
	maxEdge := maxEdgeForProfile(profile)
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease", maxEdge, maxEdge),
		"-c:v", "libwebp",
		"-quality", "92",
		"-preset", "picture",
		outputPath,
	}
	if err = f.run(ctx, args...); err != nil {
		return originalPrepared(inputPath, inInfo.Size(), ext, cleanup), nil
	}

	chosen, err := pickSmaller(inputPath, inInfo.Size(), ext, mimeForExt(ext), outputPath, ".webp", "image/webp")
	if err != nil {
		cleanup()
		return nil, err
	}
	chosen.cleanup = cleanup
	return chosen, nil
}

func (f *FFmpeg) OptimizeVideo(ctx context.Context, src io.Reader, inputExt string) (*PreparedMedia, error) {
	ext := normalizeExt(inputExt)
	dir, err := os.MkdirTemp("", "clap-vid-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	cleanup := func() { _ = os.RemoveAll(dir) }

	inputPath := filepath.Join(dir, "input"+ext)
	if err = writeReaderToFile(inputPath, src); err != nil {
		cleanup()
		return nil, err
	}

	inInfo, err := os.Stat(inputPath)
	if err != nil {
		cleanup()
		return nil, err
	}

	outputPath := filepath.Join(dir, "output.mp4")
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-vf", "scale='min(1920,iw)':'-2':force_original_aspect_ratio=decrease",
		"-c:a", "aac",
		"-b:a", "128k",
		outputPath,
	}
	if err = f.run(ctx, args...); err != nil {
		return originalPrepared(inputPath, inInfo.Size(), ext, cleanup), nil
	}

	chosen, err := pickSmaller(inputPath, inInfo.Size(), ext, mimeForExt(ext), outputPath, ".mp4", "video/mp4")
	if err != nil {
		cleanup()
		return nil, err
	}
	chosen.cleanup = cleanup
	return chosen, nil
}

func (f *FFmpeg) VideoThumbnail(ctx context.Context, videoPath string) (*PreparedMedia, error) {
	dir, err := os.MkdirTemp("", "clap-vid-thumb-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	outputPath := filepath.Join(dir, "thumb.webp")
	tryExtract := func(seek string) error {
		args := []string{
			"-hide_banner", "-loglevel", "error", "-y",
			"-ss", seek,
			"-i", videoPath,
			"-frames:v", "1",
			"-vf", "scale='min(640,iw)':'min(640,ih)':force_original_aspect_ratio=decrease",
			"-c:v", "libwebp",
			"-quality", "85",
			outputPath,
		}
		return f.run(ctx, args...)
	}

	if err = tryExtract("1"); err != nil {
		if err = tryExtract("0"); err != nil {
			cleanup()
			return nil, err
		}
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		cleanup()
		return nil, err
	}

	return &PreparedMedia{
		Path:        outputPath,
		Size:        info.Size(),
		ContentType: "image/webp",
		Extension:   ".webp",
		cleanup:     cleanup,
	}, nil
}

func originalPrepared(path string, size int64, ext string, cleanup func()) *PreparedMedia {
	return &PreparedMedia{
		Path:        path,
		Size:        size,
		ContentType: mimeForExt(ext),
		Extension:   ext,
		cleanup:     cleanup,
	}
}

func (f *FFmpeg) run(ctx context.Context, args ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, ffmpegBin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg: %s", msg)
	}
	return nil
}

func maxEdgeForProfile(profile ImageProfile) int {
	switch profile {
	case ImageProfileAvatar:
		return 1024
	case ImageProfileProduct:
		return 1600
	default:
		return 1920
	}
}

func writeReaderToFile(path string, src io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	if _, err = io.Copy(f, src); err != nil {
		_ = f.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	return nil
}
