package unit

import (
	"context"
	"net/http"
	"testing"

	"clap/internal/modules/auth/models"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/errors"
)

type stubGoogleVerifier struct {
	identity *authsvc.GoogleIdentity
	err      error
}

func (v stubGoogleVerifier) VerifyIDToken(context.Context, string) (*authsvc.GoogleIdentity, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.identity, nil
}

func newGoogleTestService(verifier authsvc.GoogleTokenVerifier) (authsvc.AuthService, *stubUserRepo) {
	userRepo := newStubUserRepo()
	svc := authsvc.NewAuthServiceWithGoogle(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		verifier,
	)
	return svc, userRepo
}

func TestGoogleLogin_CreatesUserAndIssuesTokens(t *testing.T) {
	svc, userRepo := newGoogleTestService(stubGoogleVerifier{
		identity: &authsvc.GoogleIdentity{
			Subject: "google-sub-1",
			Email:   "Fan@Example.com",
			Name:    "Alex Fan",
			Given:   "Alex",
			Family:  "Fan",
		},
	})

	user, tokens, err := svc.LoginWithGoogle(context.Background(), "fake-id-token", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("LoginWithGoogle failed: %v", err)
	}
	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected a full token pair")
	}
	if user.Email != "fan@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.FirstName != "Alex" || user.LastName != "Fan" {
		t.Fatalf("unexpected name %q %q", user.FirstName, user.LastName)
	}
	if user.GoogleID == nil || *user.GoogleID != "google-sub-1" {
		t.Fatal("expected google_id to be stored")
	}
	if !user.IsVerified {
		t.Fatal("expected google user to be verified")
	}

	stored, err := userRepo.FindByGoogleID(context.Background(), "google-sub-1")
	if err != nil {
		t.Fatalf("FindByGoogleID: %v", err)
	}
	if stored.ID != user.ID {
		t.Fatal("stored user mismatch")
	}
}

func TestGoogleLogin_LinksExistingOTPAccount(t *testing.T) {
	otpSvc, userRepo, sender := newOTPTestService()
	ctx := context.Background()
	if _, err := otpSvc.Register(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, _, err := otpSvc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "", ""); err != nil {
		t.Fatalf("verify: %v", err)
	}

	googleSvc := authsvc.NewAuthServiceWithGoogle(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		stubGoogleVerifier{
			identity: &authsvc.GoogleIdentity{
				Subject: "google-sub-1",
				Email:   "fan@example.com",
				Given:   "Alex",
				Family:  "Fan",
			},
		},
	)

	user, tokens, err := googleSvc.LoginWithGoogle(ctx, "fake-id-token", "", "")
	if err != nil {
		t.Fatalf("LoginWithGoogle failed: %v", err)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatal("expected tokens")
	}
	if user.GoogleID == nil || *user.GoogleID != "google-sub-1" {
		t.Fatal("expected existing account to be linked")
	}

	count := 0
	for range userRepo.byID {
		count++
	}
	if count != 1 {
		t.Fatalf("expected a single linked user, got %d", count)
	}
}

func TestGoogleLogin_ReusesGoogleID(t *testing.T) {
	svc, _ := newGoogleTestService(stubGoogleVerifier{
		identity: &authsvc.GoogleIdentity{
			Subject: "google-sub-1",
			Email:   "fan@example.com",
			Given:   "Alex",
		},
	})
	ctx := context.Background()

	first, _, err := svc.LoginWithGoogle(ctx, "token-1", "", "")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, _, err := svc.LoginWithGoogle(ctx, "token-2", "", "")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if first.ID != second.ID {
		t.Fatal("expected the same user for the same google subject")
	}
}

func TestGoogleLogin_InvalidTokenUnauthorized(t *testing.T) {
	svc, _ := newGoogleTestService(stubGoogleVerifier{
		err: errors.NewUnauthorized("Invalid Google token", nil),
	})

	_, _, err := svc.LoginWithGoogle(context.Background(), "bad-token", "", "")
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestGoogleLogin_Unconfigured(t *testing.T) {
	svc, _ := newGoogleTestService(nil)

	_, _, err := svc.LoginWithGoogle(context.Background(), "token", "", "")
	if err == nil {
		t.Fatal("expected unconfigured error")
	}
	if status := appErrorStatus(t, err); status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", status)
	}
}

func TestGoogleLogin_InactiveUser(t *testing.T) {
	svc, userRepo := newGoogleTestService(stubGoogleVerifier{
		identity: &authsvc.GoogleIdentity{
			Subject: "google-sub-1",
			Email:   "fan@example.com",
			Given:   "Alex",
		},
	})
	gid := "google-sub-1"
	_ = userRepo.Create(context.Background(), &models.User{
		Email:      "fan@example.com",
		GoogleID:   &gid,
		FirstName:  "Alex",
		IsActive:   false,
		IsVerified: true,
	})

	_, _, err := svc.LoginWithGoogle(context.Background(), "token", "", "")
	if err == nil {
		t.Fatal("expected inactive error")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}
