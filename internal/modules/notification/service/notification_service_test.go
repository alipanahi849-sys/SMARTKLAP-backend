package service

import (
	"context"
	"testing"
	"time"

	"clap/internal/modules/notification/models"

	"github.com/google/uuid"
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

func TestChantCountdownCopy(t *testing.T) {
	t.Parallel()

	title, body := chantCountdownCopy("  Seven Nation Army  ", 2)
	if title != "Seven Nation Army" {
		t.Fatalf("title = %q", title)
	}
	if body != "Chant countdown has started. It begins in 2 minutes." {
		t.Fatalf("body = %q", body)
	}

	title, body = chantCountdownCopy("", 1)
	if title != "SMARTKLAP" {
		t.Fatalf("fallback title = %q", title)
	}
	if body != "Chant countdown has started. It begins in 1 minute." {
		t.Fatalf("singular body = %q", body)
	}
}

func TestCountdownMinutes(t *testing.T) {
	t.Parallel()

	got := countdownMinutes(time.Now().Add(90*time.Second), 2*time.Minute)
	if got != 2 {
		t.Fatalf("90s remaining: got %d, want 2", got)
	}
	got = countdownMinutes(time.Now().Add(-time.Second), 2*time.Minute)
	if got != 2 {
		t.Fatalf("already started: got %d, want lead minutes", got)
	}
}

type stubDeviceRepo struct {
	tokens []string
}

func (stubDeviceRepo) Upsert(context.Context, *models.PushDevice) (*models.PushDevice, error) {
	return nil, nil
}
func (stubDeviceRepo) DeleteByUserAndToken(context.Context, uuid.UUID, string) error { return nil }
func (r stubDeviceRepo) ListTokens(context.Context) ([]string, error)                { return r.tokens, nil }
func (stubDeviceRepo) DeleteByTokens(context.Context, []string) error                { return nil }

type recordingSender struct {
	tokens []string
	msg    PushMessage
}

func (s *recordingSender) Send(_ context.Context, tokens []string, msg PushMessage) ([]string, error) {
	s.tokens = append([]string(nil), tokens...)
	s.msg = msg
	return nil, nil
}

func TestNotifyChantCountdown(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := NewNotificationService(stubDeviceRepo{tokens: []string{"abcdefghijklmnopqrstuvwxyz012345"}}, sender)
	startsAt := time.Now().UTC().Add(2 * time.Minute)
	if err := svc.NotifyChantCountdown(context.Background(), "chant-1", "match-1", "Chelsea Anthem", startsAt, 2*time.Minute); err != nil {
		t.Fatalf("NotifyChantCountdown: %v", err)
	}
	if sender.msg.Title != "Chelsea Anthem" {
		t.Fatalf("title = %q", sender.msg.Title)
	}
	if sender.msg.Data["type"] != chantCountdownType {
		t.Fatalf("type = %q", sender.msg.Data["type"])
	}
	if sender.msg.Data["path"] != chantCountdownPath {
		t.Fatalf("path = %q", sender.msg.Data["path"])
	}
	if sender.msg.Data["chant_id"] != "chant-1" {
		t.Fatalf("chant_id = %q", sender.msg.Data["chant_id"])
	}
}

func TestParseServiceAccountRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, _, err := parseServiceAccount([]byte(`{"project_id":"x"}`)); err == nil {
		t.Fatal("expected error for incomplete credentials")
	}
	if _, _, err := parseServiceAccount([]byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestNotifyChantCountdownNoopWithoutSender(t *testing.T) {
	t.Parallel()

	svc := NewNotificationService(stubDeviceRepo{tokens: []string{"abcdefghijklmnopqrstuvwxyz012345"}}, nil)
	if err := svc.NotifyChantCountdown(context.Background(), "c", "m", "Song", time.Now(), time.Minute); err != nil {
		t.Fatalf("expected noop, got %v", err)
	}
}
