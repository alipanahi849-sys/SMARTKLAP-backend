package unit

import (
	"context"
	"net/http"
	"testing"

	"clap/internal/modules/auth/models"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/errors"
)

type stubAppleVerifier struct {
	identity *authsvc.AppleIdentity
	err      error
}

func (v stubAppleVerifier) VerifyIDToken(context.Context, string) (*authsvc.AppleIdentity, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.identity, nil
}

func newAppleTestService(verifier authsvc.AppleTokenVerifier) (authsvc.AuthService, *stubUserRepo) {
	userRepo := newStubUserRepo()
	svc := authsvc.NewAuthServiceWithApple(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		verifier,
	)
	return svc, userRepo
}

func TestAppleLogin_CreatesUserAndIssuesTokens(t *testing.T) {
	svc, userRepo := newAppleTestService(stubAppleVerifier{
		identity: &authsvc.AppleIdentity{
			Subject: "apple-sub-1",
			Email:   "Fan@Example.com",
		},
	})

	user, tokens, err := svc.LoginWithApple(context.Background(), "fake-id-token", "Alex", "Fan", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("LoginWithApple failed: %v", err)
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
	if user.AppleID == nil || *user.AppleID != "apple-sub-1" {
		t.Fatal("expected apple_id to be stored")
	}
	if !user.IsVerified {
		t.Fatal("expected apple user to be verified")
	}

	stored, err := userRepo.FindByAppleID(context.Background(), "apple-sub-1")
	if err != nil {
		t.Fatalf("FindByAppleID: %v", err)
	}
	if stored.ID != user.ID {
		t.Fatal("stored user mismatch")
	}
}

func TestAppleLogin_LinksExistingOTPAccount(t *testing.T) {
	otpSvc, userRepo, sender := newOTPTestService()
	ctx := context.Background()
	if _, err := otpSvc.Register(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, _, err := otpSvc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "", ""); err != nil {
		t.Fatalf("verify: %v", err)
	}

	appleSvc := authsvc.NewAuthServiceWithApple(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		stubAppleVerifier{
			identity: &authsvc.AppleIdentity{
				Subject: "apple-sub-1",
				Email:   "fan@example.com",
			},
		},
	)

	user, tokens, err := appleSvc.LoginWithApple(ctx, "fake-id-token", "Alex", "Fan", "", "")
	if err != nil {
		t.Fatalf("LoginWithApple failed: %v", err)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatal("expected tokens")
	}
	if user.AppleID == nil || *user.AppleID != "apple-sub-1" {
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

func TestAppleLogin_ReturningUserOmitsEmail(t *testing.T) {
	userRepo := newStubUserRepo()
	ctx := context.Background()

	firstSvc := authsvc.NewAuthServiceWithApple(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		stubAppleVerifier{
			identity: &authsvc.AppleIdentity{
				Subject: "apple-sub-1",
				Email:   "fan@example.com",
			},
		},
	)
	first, _, err := firstSvc.LoginWithApple(ctx, "token-1", "Alex", "Fan", "", "")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}

	secondSvc := authsvc.NewAuthServiceWithApple(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		&captureSender{},
		stubAppleVerifier{
			identity: &authsvc.AppleIdentity{
				Subject: "apple-sub-1",
				Email:   "",
			},
		},
	)
	second, _, err := secondSvc.LoginWithApple(ctx, "token-2", "", "", "", "")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if first.ID != second.ID {
		t.Fatal("expected the same user for the same apple subject")
	}
}

func TestAppleLogin_NewUserWithoutEmailUnauthorized(t *testing.T) {
	svc, _ := newAppleTestService(stubAppleVerifier{
		identity: &authsvc.AppleIdentity{
			Subject: "apple-sub-1",
			Email:   "",
		},
	})

	_, _, err := svc.LoginWithApple(context.Background(), "token", "", "", "", "")
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestAppleLogin_InvalidTokenUnauthorized(t *testing.T) {
	svc, _ := newAppleTestService(stubAppleVerifier{
		err: errors.NewUnauthorized("Invalid Apple token", nil),
	})

	_, _, err := svc.LoginWithApple(context.Background(), "bad-token", "", "", "", "")
	if err == nil {
		t.Fatal("expected unauthorized")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestAppleLogin_Unconfigured(t *testing.T) {
	svc, _ := newAppleTestService(nil)

	_, _, err := svc.LoginWithApple(context.Background(), "token", "", "", "", "")
	if err == nil {
		t.Fatal("expected unconfigured error")
	}
	if status := appErrorStatus(t, err); status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", status)
	}
}

func TestAppleLogin_InactiveUser(t *testing.T) {
	svc, userRepo := newAppleTestService(stubAppleVerifier{
		identity: &authsvc.AppleIdentity{
			Subject: "apple-sub-1",
			Email:   "fan@example.com",
		},
	})
	aid := "apple-sub-1"
	_ = userRepo.Create(context.Background(), &models.User{
		Email:      "fan@example.com",
		AppleID:    &aid,
		FirstName:  "Alex",
		IsActive:   false,
		IsVerified: true,
	})

	_, _, err := svc.LoginWithApple(context.Background(), "token", "", "", "", "")
	if err == nil {
		t.Fatal("expected inactive error")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}
