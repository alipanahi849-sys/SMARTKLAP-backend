package unit

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"clap/internal/modules/auth/models"
	authrepo "clap/internal/modules/auth/repository"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/database"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

type stubUserRepo struct {
	mu       sync.Mutex
	byEmail  map[string]*models.User
	byID     map[uuid.UUID]*models.User
	byGoogle map[string]*models.User
	byApple  map[string]*models.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{
		byEmail:  map[string]*models.User{},
		byID:     map[uuid.UUID]*models.User{},
		byGoogle: map[string]*models.User{},
		byApple:  map[string]*models.User{},
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
	if user.GoogleID != nil && *user.GoogleID != "" {
		r.byGoogle[*user.GoogleID] = user
	}
	if user.AppleID != nil && *user.AppleID != "" {
		r.byApple[*user.AppleID] = user
	}
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

func (r *stubUserRepo) FindByGoogleID(_ context.Context, googleID string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byGoogle[googleID]; ok {
		return u, nil
	}
	return nil, sharederrors.ErrUserNotFound
}

func (r *stubUserRepo) FindByAppleID(_ context.Context, appleID string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byApple[appleID]; ok {
		return u, nil
	}
	return nil, sharederrors.ErrUserNotFound
}

func (r *stubUserRepo) Update(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// FindByID returns the stored pointer, so Email may already be mutated.
	// Drop every email key that still points at this user, then reindex.
	for email, u := range r.byEmail {
		if u.ID == user.ID {
			delete(r.byEmail, email)
		}
	}
	for gid, u := range r.byGoogle {
		if u.ID == user.ID {
			delete(r.byGoogle, gid)
		}
	}
	for aid, u := range r.byApple {
		if u.ID == user.ID {
			delete(r.byApple, aid)
		}
	}
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	if user.GoogleID != nil && *user.GoogleID != "" {
		r.byGoogle[*user.GoogleID] = user
	}
	if user.AppleID != nil && *user.AppleID != "" {
		r.byApple[*user.AppleID] = user
	}
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

func (r *stubUserRepo) SpendPoints(_ context.Context, userID uuid.UUID, amount int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[userID]
	if !ok {
		return 0, sharederrors.ErrUserNotFound
	}
	if u.Points < amount {
		return 0, sharederrors.NewUnprocessable("Insufficient points balance", nil)
	}
	u.Points -= amount
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

func (r *stubUserRepo) TopByPointsAfter(_ context.Context, limit int, _ *authrepo.LeaderboardCursorAnchor) ([]models.User, error) {
	var users []models.User
	for _, u := range r.byID {
		users = append(users, *u)
	}
	if len(users) > limit {
		users = users[:limit]
	}
	return users, nil
}

func (r *stubUserRepo) LeaderboardRank(_ context.Context, _ int, _ time.Time, _ uuid.UUID) (int, error) {
	return 1, nil
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

	result, err := svc.Register(context.Background(), "Alex Fan", "Fan@Example.com")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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
	// Pending only — user must not exist until verify-otp succeeds.
	if _, err := userRepo.FindByEmail(context.Background(), "fan@example.com"); err == nil {
		t.Fatal("user must not be persisted before OTP verify")
	}
}

func TestOTP_RegisterDuplicateEmailConflicts(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if _, _, err := svc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "", ""); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	_, err := svc.Register(ctx, "Alex", "fan@example.com")
	if err == nil {
		t.Fatal("expected conflict for duplicate email")
	}
	if status := appErrorStatus(t, err); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
}

func TestOTP_RegisterRequiresName(t *testing.T) {
	svc, _, _ := newOTPTestService()

	_, err := svc.Register(context.Background(), "   ", "fan@example.com")
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

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
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

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
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

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
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

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
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

	_, err := svc.Login(context.Background(), "ghost@example.com")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestOTP_LoginCooldownEnforced(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, _, err := svc.VerifyOTP(ctx, "fan@example.com", sender.LastCode(), "", ""); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if _, err := svc.Login(ctx, "fan@example.com"); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Immediate second login hits the 30s cooldown.
	_, err := svc.Login(ctx, "fan@example.com")
	if err == nil {
		t.Fatal("expected cooldown error")
	}
	if status := appErrorStatus(t, err); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", status)
	}
}

func TestOTP_RegisterResendCooldownEnforced(t *testing.T) {
	svc, _, _ := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "fan@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	_, err := svc.Register(ctx, "Alex", "fan@example.com")
	if err == nil {
		t.Fatal("expected cooldown error on register resend")
	}
	if status := appErrorStatus(t, err); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", status)
	}
}

func TestOTP_ChangeEmailHappyPath(t *testing.T) {
	svc, userRepo, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "old@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	user, _, err := svc.VerifyOTP(ctx, "old@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	if _, err := svc.RequestChangeEmail(ctx, user.ID, "New@Example.com"); err != nil {
		t.Fatalf("request change email failed: %v", err)
	}
	updated, tokens, err := svc.VerifyChangeEmail(ctx, user.ID, "new@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify change email failed: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Fatalf("expected new email, got %q", updated.Email)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatal("expected fresh tokens after email change")
	}
	userRepo.mu.Lock()
	for email, u := range userRepo.byEmail {
		t.Logf("byEmail[%q]=%s emailField=%q", email, u.ID, u.Email)
	}
	userRepo.mu.Unlock()
	if _, err := userRepo.FindByEmail(ctx, "old@example.com"); err == nil {
		t.Fatal("old email should no longer resolve")
	}
	if _, err := userRepo.FindByEmail(ctx, "new@example.com"); err != nil {
		t.Fatalf("new email should resolve: %v", err)
	}
}

func TestOTP_ChangeEmailRejectsTakenAddress(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Other", "taken@example.com"); err != nil {
		t.Fatalf("register other failed: %v", err)
	}
	if _, _, err := svc.VerifyOTP(ctx, "taken@example.com", sender.LastCode(), "", ""); err != nil {
		t.Fatalf("verify other failed: %v", err)
	}

	if _, err := svc.Register(ctx, "Alex", "me@example.com"); err != nil {
		t.Fatalf("register me failed: %v", err)
	}
	me, _, err := svc.VerifyOTP(ctx, "me@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify me failed: %v", err)
	}

	_, err = svc.RequestChangeEmail(ctx, me.ID, "taken@example.com")
	if err == nil {
		t.Fatal("expected conflict for taken email")
	}
	if status := appErrorStatus(t, err); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
}

func TestOTP_ChangeEmailRejectsSameEmail(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "me@example.com"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	me, _, err := svc.VerifyOTP(ctx, "me@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	_, err = svc.RequestChangeEmail(ctx, me.ID, "me@example.com")
	if err == nil {
		t.Fatal("expected bad request for unchanged email")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestOTP_ChangeEmailWrongUserRejected(t *testing.T) {
	svc, _, sender := newOTPTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "Alex", "a@example.com"); err != nil {
		t.Fatalf("register a failed: %v", err)
	}
	a, _, err := svc.VerifyOTP(ctx, "a@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify a failed: %v", err)
	}
	if _, err := svc.Register(ctx, "Bob", "b@example.com"); err != nil {
		t.Fatalf("register b failed: %v", err)
	}
	b, _, err := svc.VerifyOTP(ctx, "b@example.com", sender.LastCode(), "", "")
	if err != nil {
		t.Fatalf("verify b failed: %v", err)
	}

	if _, err := svc.RequestChangeEmail(ctx, a.ID, "new@example.com"); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	code := sender.LastCode()

	// Bob must not be able to consume Alex's change-email OTP.
	_, _, err = svc.VerifyChangeEmail(ctx, b.ID, "new@example.com", code, "", "")
	if err == nil {
		t.Fatal("expected unauthorized for wrong user")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}
