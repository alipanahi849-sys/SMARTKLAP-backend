package mediainit

import "clap/pkg/media/optimize"

// Optimizer returns the shared upload optimizer (ffmpeg when installed).
func Optimizer() optimize.Optimizer {
	return optimize.New()
}
