package unit

import (
	"os"
	"testing"

	"clap/internal/shared/config"
)

// TestMain initialises a minimal in-memory config before any test in this
// package runs.  This prevents nil-pointer panics in code that reads
// config.AppConfig (e.g. JWT validation utilities).
func TestMain(m *testing.M) {
	config.AppConfig = &config.Config{
		Environment: "test",
		JWT: config.JWT{
			Secret:        "test-secret-key-for-unit-tests-only",
			AccessExpiry:  3600,
			RefreshExpiry: 86400,
			Issuer:        "clap-test",
			RefreshSecret: "test-refresh-secret-key",
		},
		Server: config.Server{
			Port: "8080",
		},
	}

	os.Exit(m.Run())
}
