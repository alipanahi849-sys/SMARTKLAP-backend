package unit

import (
	"context"
	"net/http"
	"testing"

	authmodels "clap/internal/modules/auth/models"
	userdto "clap/internal/modules/user/dto"
	usersvc "clap/internal/modules/user/service"
	"clap/internal/shared/database"

	"github.com/google/uuid"
)

// ─── profile tests ────────────────────────────────────────────────────────────

func newProfileFixture() (usersvc.MobileProfileService, *stubUserRepo) {
	userRepo := newStubUserRepo()
	svc := usersvc.NewMobileProfileService(userRepo, stubProfileRepo{}, newMemoryStorage())
	return svc, userRepo
}

func seedUser(repo *stubUserRepo, name, email string, points int) *authmodels.User {
	u := &authmodels.User{
		BaseModel: database.BaseModel{ID: uuid.New()},
		Email:     email,
		FirstName: name,
		IsActive:  true,
		Points:    points,
	}
	_ = repo.Create(context.Background(), u)
	return u
}

func TestProfile_GetMeIncludesIdentity(t *testing.T) {
	svc, userRepo := newProfileFixture()

	seedUser(userRepo, "Leader", "leader@example.com", 900)
	me := seedUser(userRepo, "Me", "me@example.com", 500)
	seedUser(userRepo, "Behind", "behind@example.com", 100)

	profile, err := svc.GetMe(context.Background(), me.ID)
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}
	if profile.ID != me.ID {
		t.Fatalf("expected id %s, got %s", me.ID, profile.ID)
	}
	if profile.Name != "Me" {
		t.Fatalf("expected name Me, got %q", profile.Name)
	}
	if profile.Points != 500 {
		t.Fatalf("expected 500 points, got %d", profile.Points)
	}
}

func TestProfile_UpdateMeRejectsEmptyName(t *testing.T) {
	svc, userRepo := newProfileFixture()
	me := seedUser(userRepo, "Me", "me@example.com", 0)

	empty := "  "
	_, err := svc.UpdateMe(context.Background(), me.ID, &userdto.UpdateMobileProfileRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestProfile_UpdateMeRejectsEmailChange(t *testing.T) {
	svc, userRepo := newProfileFixture()
	me := seedUser(userRepo, "Me", "me@example.com", 0)

	next := "new@example.com"
	_, err := svc.UpdateMe(context.Background(), me.ID, &userdto.UpdateMobileProfileRequest{Email: &next})
	if err == nil {
		t.Fatal("expected error directing clients to auth change-email")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestProfile_GetMeUnknownUserNotFound(t *testing.T) {
	svc, _ := newProfileFixture()

	_, err := svc.GetMe(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}
