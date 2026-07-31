// Package storageinit builds the configured StorageProvider so every module
// (media, profile avatars, videos) shares one construction path.
package storageinit

import (
	"clap/internal/shared/config"
	"clap/internal/shared/logger"
	"clap/pkg/storage"
)

// Provider returns the storage backend configured via config.Storage, or nil
// when no provider is configured (uploads will fail with a clear error).
func Provider() storage.StorageProvider {
	cfg := config.AppConfig
	if cfg == nil {
		return nil
	}

	switch cfg.Storage.Provider {
	case "r2":
		if cfg.Storage.R2AccountID == "" || cfg.Storage.R2AccessKeyID == "" ||
			cfg.Storage.R2SecretAccessKey == "" || cfg.Storage.R2Bucket == "" {
			logger.Warn().Msg("R2 storage selected but credentials incomplete; falling back to local disk")
			return newLocal(cfg)
		}
		r2Config := &storage.R2Config{
			AccountID:       cfg.Storage.R2AccountID,
			AccessKeyID:     cfg.Storage.R2AccessKeyID,
			SecretAccessKey: cfg.Storage.R2SecretAccessKey,
			Bucket:          cfg.Storage.R2Bucket,
		}
		provider, err := storage.NewR2Provider(r2Config)
		if err != nil {
			logger.Error().Err(err).Msg("failed to initialize R2 storage provider")
			return nil
		}
		logger.Info().Str("bucket", cfg.Storage.R2Bucket).Msg("R2 storage provider ready")
		return provider

	case "local", "":
		return newLocal(cfg)

	default:
		logger.Error().Str("provider", cfg.Storage.Provider).Msg("unknown storage provider")
		return nil
	}
}

func newLocal(cfg *config.Config) storage.StorageProvider {
	provider, err := storage.NewLocalProvider(cfg.Storage.LocalRoot, cfg.Storage.LocalPublicURL)
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize local storage provider")
		return nil
	}
	logger.Info().
		Str("root", cfg.Storage.LocalRoot).
		Str("public_url", cfg.Storage.LocalPublicURL).
		Msg("local storage provider ready")
	return provider
}
