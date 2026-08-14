package service

import (
	"testing"

	"clap/internal/modules/notification/models"
)

func TestNormalizeFCMToken(t *testing.T) {
	t.Parallel()

	if _, err := normalizeFCMToken("   "); err == nil {
		t.Fatal("expected error for empty token")
	}
	if _, err := normalizeFCMToken("short"); err == nil {
		t.Fatal("expected error for short token")
	}

	token := "abcdefghijklmnopqrstuvwxyz012345"
	got, err := normalizeFCMToken("  " + token + "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != token {
		t.Fatalf("got %q, want %q", got, token)
	}
}

func TestNormalizePlatform(t *testing.T) {
	t.Parallel()

	got, err := normalizePlatform("iOS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != models.PlatformIOS {
		t.Fatalf("got %q, want ios", got)
	}

	got, err = normalizePlatform("Android")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != models.PlatformAndroid {
		t.Fatalf("got %q, want android", got)
	}

	if _, err := normalizePlatform("web"); err == nil {
		t.Fatal("expected error for unsupported platform")
	}
}
