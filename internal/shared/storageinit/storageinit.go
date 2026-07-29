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

	if cfg.Storage.Provider == "r2" {
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
		return provider
	}

	return nil
}
