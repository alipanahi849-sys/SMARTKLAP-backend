package config

import (
	"os"
	"testing"
)

func TestParsePostgresURL(t *testing.T) {
	got, err := parsePostgresURL("postgres://clap:s3cret@dpg-xyz:5432/clap")
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "dpg-xyz" || got.Port != "5432" || got.User != "clap" || got.Password != "s3cret" || got.DBName != "clap" {
		t.Fatalf("parsed fields: %+v", got)
	}
	if got.SSLMode != "require" {
		t.Fatalf("sslmode = %q, want require", got.SSLMode)
	}
	if got.URL == "" {
		t.Fatal("expected normalized URL")
	}
}

func TestLoadFromEnvRenderStyle(t *testing.T) {
	t.Setenv("JWT_SECRET", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	t.Setenv("JWT_REFRESH_SECRET", "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy")
	t.Setenv("PORT", "10000")
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://clap:s3cret@dpg-xyz:5432/clap")
	t.Setenv("REDIS_URL", "redis://red-xyz:6379")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("SMTP_HOST", "")

	if err := LoadFromEnv(); err != nil {
		t.Fatal(err)
	}
	if AppConfig.Server.Port != "10000" {
		t.Fatalf("port = %q, want 10000 (Render PORT)", AppConfig.Server.Port)
	}
	if AppConfig.Database.Host != "dpg-xyz" || AppConfig.Database.Password != "s3cret" {
		t.Fatalf("database = %+v", AppConfig.Database)
	}
	if AppConfig.Redis.Host != "red-xyz" || AppConfig.Redis.Port != "6379" {
		t.Fatalf("redis = %+v", AppConfig.Redis)
	}
	if dsn := GetDSN(); dsn == "" || dsn[:8] != "postgres" {
		t.Fatalf("GetDSN() = %q", dsn)
	}
}

func TestLoadFromEnvRequiresPasswordWithoutURL(t *testing.T) {
	t.Setenv("JWT_SECRET", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	t.Setenv("JWT_REFRESH_SECRET", "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("PORT")
	t.Setenv("DB_PASSWORD", "")

	if err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when database password is missing")
	}
}
