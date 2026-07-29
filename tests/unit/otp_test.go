package unit

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"clap/internal/modules/auth/models"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/database"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

type stubUserRepo struct {
	mu      sync.Mutex
	byEmail map[string]*models.User
	byID    map[uuid.UUID]*models.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{
		byEmail: map[string]*models.User{},
		byID:    map[uuid.UUID]*models.User{},
	}
}

func (r *stubUserRepo) Create(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	return nil
}

func (r *stubUserRepo) FindByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, sharederrors.ErrUserNotFound
}

func (r *stubUserRepo) FindByEmail(_ context.Context, email string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, sharederrors.ErrUserNotFound
}

func (r *stubUserRepo) Update(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	return nil
}

func (r *stubUserRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (r *stubUserRepo) List(context.Context, int, int) ([]models.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) AddRole(context.Context, uuid.UUID, uuid.UUID) error    { return nil }
func (r *stubUserRepo) RemoveRole(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *stubUserRepo) GetUserRoles(context.Context, uuid.UUID) ([]models.Role, error) {
	return nil, nil
}

func (r *stubUserRepo) AddPoints(_ context.Context, userID uuid.UUID, delta int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[userID]
	if !ok {
		return 0, sharederrors.ErrUserNotFound
	}
	u.Points += delta
	return u.Points, nil
}

func (r *stubUserRepo) CountActive(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *stubUserRepo) CountWithMorePoints(_ context.Context, points int) (int64, error) {
	var n int64
	for _, u := range r.byID {
		if u.Points > points {
			n++
		}
	}
	return n, nil
}

func (r *stubUserRepo) TopByPoints(_ context.Context, limit int) ([]models.User, error) {
	var users []models.User
	for _, u := range r.byID {
		users = append(users, *u)
	}
	if len(users) > limit {
		users = users[:limit]
	}
	return users, nil
}

type stubRoleRepo struct{}

func (stubRoleRepo) Create(context.Context, *models.Role) error { return nil }
func (stubRoleRepo) FindByID(context.Context, uuid.UUID) (*models.Role, error) {
	return &models.Role{BaseModel: database.BaseModel{ID: uuid.New()}, Name: models.RoleUser}, nil
}
func (stubRoleRepo) FindByName(_ context.Context, name string) (*models.Role, error) {
	return &models.Role{BaseModel: database.BaseModel{ID: uuid.New()}, Name: name}, nil
}
func (stubRoleRepo) Update(context.Context, *models.Role) error { return nil }
func (stubRoleRepo) Delete(context.Context, uuid.UUID) error    { return nil }
func (stubRoleRepo) List(context.Context, int, int) ([]models.Role, int64, error) {
	return nil, 0, nil
}

type stubRefreshTokenRepo struct{}

func (stubRefreshTokenRepo) Create(context.Context, *models.RefreshToken) error { return nil }
func (stubRefreshTokenRepo) FindByID(context.Context, uuid.UUID) (*models.RefreshToken, error) {
	return nil, sharederrors.NewNotFound("not found", nil)
}
func (stubRefreshTokenRepo) FindByToken(context.Context, string) (*models.RefreshToken, error) {
	return nil, sharederrors.NewNotFound("not found", nil)
}
func (stubRefreshTokenRepo) FindByUserID(context.Context, uuid.UUID) ([]models.RefreshToken, error) {
	return nil, nil
}
func (stubRefreshTokenRepo) Update(context.Context, *models.RefreshToken) error { return nil }
func (stubRefreshTokenRepo) Delete(context.Context, uuid.UUID) error            { return nil }
func (stubRefreshTokenRepo) Revoke(context.Context, uuid.UUID) error            { return nil }
func (stubRefreshTokenRepo) RevokeAllForUser(context.Context, uuid.UUID) error  { return nil }
func (stubRefreshTokenRepo) DeleteExpired(context.Context) error                { return nil }

// captureSender records the last OTP code sent so tests can verify it.
type captureSender struct {
	mu       sync.Mutex
	lastCode string
	sent     int
}

func (s *captureSender) SendOTP(_ context.Context, _ string, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCode = code
	s.sent++
	return nil
}

func (s *captureSender) LastCode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastCode
}

func newOTPTestService() (authsvc.AuthService, *stubUserRepo, *captureSender) {
	userRepo := newStubUserRepo()
	sender := &captureSender{}
	svc := authsvc.NewAuthServiceWithOTP(
		userRepo,
		stubRoleRepo{},
		stubRefreshTokenRepo{},
		authsvc.NewMemoryOTPStore(),
		sender,
	)
	return svc, userRepo, sender
}

func appErrorStatus(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := err.(*sharederrors.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	return appErr.StatusCode
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestOTP_RegisterSendsCode(t *testing.T) {
	svc, userRepo, sender := newOTPTestService()

	result, err := svc.RegisterOTP(context.Background(), "Alex Fan", "Fan@Example.com")
	if err != nil {
		t.Fatalf("RegisterOTP failed: %v", err)
	}
	if !result.OTPSent {
		t.Fatal("expected OTPSent=true")
	}
	if result.Email != "fan@example.com" {
		t.Fatalf("expected normalized email, got %q", result.Email)
	}
	if sender.LastCode() == "" {
		t.Fatal("expected an OTP code to be sent")
	}
	if _, err := userRepo.FindByEmail(context.Background(), "fan@example.com"); err != nil {
		t.Fatalf("user was not persisted: %v", err)
	}
}

func TestOTP_RegisterDuplicateEmailConflicts(t *testing.T) {
	svc, _, _ := newOTPTestService()

	if _, err := svc.RegisterOTP(context.Background(), "Alex", "fan@example.com"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	_, err := svc.RegisterOTP(context.Background(), "Alex", "fan@example.com")
	if err == nil {
		t.Fatal("expected conflict for duplicate email")
	}
	if status := appErrorStatus(t, err); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
}

func TestOTP_RegisterRequiresName(t *testing.T) {
	svc, _, _ := newOTPTestService()

	_, err := svc.RegisterOTP(context.Background(), "   ", "fan@example.com")
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestOTP_VerifyIssuesTokens(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.RegisterOTP(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	user, tokens, err := svc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected a full token pair")
	}
	if !user.IsVerified {
		t.Fatal("expected user to be marked verified")
	}
}

func TestOTP_VerifyWrongCodeUnauthorized(t *testing.T) {
	svc, _, _ := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.RegisterOTP(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err := svc.VerifyOTP(ctx, "fan@example.com", "0000-wrong", "", "")
	if err == nil {
		t.Fatal("expected error for wrong code")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestOTP_VerifyCodeIsSingleUse(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.RegisterOTP(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	code := sender.LastCode()

	if _, _, err := svc.VerifyOTP(ctx, "fan@example.com", code, "", ""); err != nil {
		t.Fatalf("first verify failed: %v", err)
	}
	_, _, err := svc.VerifyOTP(ctx, "fan@example.com", code, "", "")
	if err == nil {
		t.Fatal("expected second verify with the same code to fail")
	}
}

func TestOTP_VerifyMaxAttemptsThenLocked(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.RegisterOTP(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		if _, _, err := svc.VerifyOTP(ctx, "fan@example.com", "wrong", "", ""); err == nil {
			t.Fatal("expected wrong-code error")
		}
	}
	// The record is deleted after max attempts — even the right code fails now.
	_, _, err := svc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "", "")
	if err == nil {
		t.Fatal("expected verification to be locked after max attempts")
	}
}

func TestOTP_LoginUnknownEmailNotFound(t *testing.T) {
	svc, _, _ := newOTPTestService()

	_, err := svc.LoginOTP(context.Background(), "ghost@example.com")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestOTP_ResendCooldownEnforced(t *testing.T) {
	svc, _, _ := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.RegisterOTP(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Immediate resend hits the 30s cooldown set by register.
	_, err := svc.ResendOTP(ctx, "fan@example.com")
	if err == nil {
		t.Fatal("expected cooldown error")
	}
	if status := appErrorStatus(t, err); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", status)
	}
}
