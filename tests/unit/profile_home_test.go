package unit

import (
	"context"
	"net/http"
	"testing"

	authmodels "clap/internal/modules/auth/models"
	clubmodels "clap/internal/modules/club/models"
	statsmodels "clap/internal/modules/stats/models"
	statssvc "clap/internal/modules/stats/service"
	userdto "clap/internal/modules/user/dto"
	usersvc "clap/internal/modules/user/service"
	"clap/internal/shared/database"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── shared stubs ─────────────────────────────────────────────────────────────

type stubClubRepo struct {
	clubs map[uuid.UUID]clubmodels.Club
}

func newStubClubRepo() *stubClubRepo {
	return &stubClubRepo{clubs: map[uuid.UUID]clubmodels.Club{}}
}

func (r *stubClubRepo) Create(_ context.Context, c *clubmodels.Club) error {
	r.clubs[c.ID] = *c
	return nil
}

func (r *stubClubRepo) FindByID(_ context.Context, id uuid.UUID) (*clubmodels.Club, error) {
	if c, ok := r.clubs[id]; ok {
		return &c, nil
	}
	return nil, sharederrors.NewNotFound("Club not found", nil)
}

func (r *stubClubRepo) FindByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]clubmodels.Club, error) {
	result := map[uuid.UUID]clubmodels.Club{}
	for _, id := range ids {
		if c, ok := r.clubs[id]; ok {
			result[id] = c
		}
	}
	return result, nil
}

func (r *stubClubRepo) FindAll(context.Context, int, int, map[string]string, string, string) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (r *stubClubRepo) Search(context.Context, string, int, int) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (r *stubClubRepo) Update(context.Context, *clubmodels.Club) error { return nil }
func (r *stubClubRepo) Delete(context.Context, uuid.UUID) error        { return nil }

type stubPlayerRepo struct {
	players map[uuid.UUID]*statsmodels.Player
}

func (r *stubPlayerRepo) FindByID(_ context.Context, id uuid.UUID) (*statsmodels.Player, error) {
	if p, ok := r.players[id]; ok {
		return p, nil
	}
	return nil, sharederrors.NewNotFound("Player not found", nil)
}

func (r *stubPlayerRepo) FindByClubIDs(_ context.Context, clubIDs []uuid.UUID) ([]statsmodels.Player, error) {
	var result []statsmodels.Player
	for _, p := range r.players {
		for _, id := range clubIDs {
			if p.ClubID == id {
				result = append(result, *p)
			}
		}
	}
	return result, nil
}

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

func TestProfile_UpdateMeRejectsTakenEmail(t *testing.T) {
	svc, userRepo := newProfileFixture()
	seedUser(userRepo, "Other", "taken@example.com", 0)
	me := seedUser(userRepo, "Me", "me@example.com", 0)

	taken := "taken@example.com"
	_, err := svc.UpdateMe(context.Background(), me.ID, &userdto.UpdateMobileProfileRequest{Email: &taken})
	if err == nil {
		t.Fatal("expected error for taken email")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", status)
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

// ─── player tests ─────────────────────────────────────────────────────────────

func TestPlayer_DetailParsesRadarStats(t *testing.T) {
	clubRepo := newStubClubRepo()
	club := clubmodels.Club{Name: "SP Burgos"}
	club.ID = uuid.New()
	clubRepo.clubs[club.ID] = club

	playerRepo := &stubPlayerRepo{players: map[uuid.UUID]*statsmodels.Player{}}
	player := &statsmodels.Player{
		ID:         uuid.New(),
		ClubID:     club.ID,
		Name:       "Dani Alves",
		RadarStats: `[{"label":"Speed","value":88},{"label":"Passing","value":91}]`,
		IsActive:   true,
	}
	playerRepo.players[player.ID] = player

	svc := statssvc.NewPlayerService(playerRepo, clubRepo)
	detail, err := svc.PlayerDetail(context.Background(), player.ID, nil)
	if err != nil {
		t.Fatalf("PlayerDetail failed: %v", err)
	}
	if detail.Club != "SP Burgos" {
		t.Fatalf("expected club name, got %q", detail.Club)
	}
	if len(detail.RadarStats) != 2 || detail.RadarStats[0].Label != "Speed" {
		t.Fatalf("unexpected radar stats: %+v", detail.RadarStats)
	}
}

func TestPlayer_DetailUnknownNotFound(t *testing.T) {
	svc := statssvc.NewPlayerService(&stubPlayerRepo{players: map[uuid.UUID]*statsmodels.Player{}}, newStubClubRepo())

	_, err := svc.PlayerDetail(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}
