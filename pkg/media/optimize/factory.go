package optimize

import "clap/internal/shared/logger"

// New returns FFmpeg when available, otherwise a passthrough optimizer.
func New() Optimizer {
	ff := NewFFmpeg()
	if ff.Available() {
		logger.Info().Msg("ffmpeg media optimizer ready")
		return ff
	}
	logger.Warn().Msg("ffmpeg not found; media uploads will be stored without compression")
	return Noop{}
}
