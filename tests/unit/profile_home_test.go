package unit

import (
	"context"
	"net/http"
	"testing"
	"time"

	authmodels "clap/internal/modules/auth/models"
	chantsvc "clap/internal/modules/chant/service"
	clubmodels "clap/internal/modules/club/models"
	homesvc "clap/internal/modules/mobilehome/service"
	newsmodels "clap/internal/modules/news/models"
	newssvc "clap/internal/modules/news/service"
	shopmodels "clap/internal/modules/shop/models"
	shopsvc "clap/internal/modules/shop/service"
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

type stubNewsRepo struct {
	items []newsmodels.News
}

func (r *stubNewsRepo) FindAll(_ context.Context, _, _ int) ([]newsmodels.News, int64, error) {
	return r.items, int64(len(r.items)), nil
}

func (r *stubNewsRepo) FindPreview(_ context.Context, limit int) ([]newsmodels.News, error) {
	if len(r.items) > limit {
		return r.items[:limit], nil
	}
	return r.items, nil
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

func TestProfile_GetMeIncludesRank(t *testing.T) {
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
	if profile.Points != 500 {
		t.Fatalf("expected 500 points, got %d", profile.Points)
	}
	if profile.Rank.Position != 2 {
		t.Fatalf("expected rank 2, got %d", profile.Rank.Position)
	}
	if profile.Rank.Total != 3 {
		t.Fatalf("expected total 3, got %d", profile.Rank.Total)
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

// ─── home tests ───────────────────────────────────────────────────────────────

func TestHome_StadiumAggregates(t *testing.T) {
	userRepo := newStubUserRepo()
	matchRepo := newStubMatchRepo()
	clubRepo := newStubClubRepo()
	chantRepo := newStubChantRepo()
	snackRepo := &stubSnackRepo{snacks: map[uuid.UUID]shopmodels.Snack{}}
	productRepo := &stubProductRepo{products: map[uuid.UUID]shopmodels.Product{}}
	cartRepo := newStubCartRepo()
	orderRepo := newStubOrderRepo(cartRepo)
	newsRepo := &stubNewsRepo{}

	me := seedUser(userRepo, "Me", "me@example.com", 289)

	shopService := shopsvc.NewShopService(snackRepo, productRepo, cartRepo, orderRepo)
	chantService := chantsvc.NewChantService(chantRepo, matchRepo, nil, nil)
	newsService := newssvc.NewNewsService(newsRepo)

	svc := homesvc.NewHomeService(userRepo, matchRepo, clubRepo, chantService, shopService, newsService)

	resp, err := svc.Stadium(context.Background(), me.ID)
	if err != nil {
		t.Fatalf("Stadium failed: %v", err)
	}
	if resp.UserSummary.Points != 289 {
		t.Fatalf("expected 289 points, got %d", resp.UserSummary.Points)
	}
	if resp.LiveMatch != nil {
		t.Fatal("expected live_match to be nil when nothing is live")
	}
	if resp.ChantProgram.TodayTarget != chantsvc.DefaultDailyTarget {
		t.Fatalf("unexpected today_target: %d", resp.ChantProgram.TodayTarget)
	}
}

func TestHome_ClubAggregates(t *testing.T) {
	userRepo := newStubUserRepo()
	matchRepo := newStubMatchRepo()
	clubRepo := newStubClubRepo()

	home := clubmodels.Club{Name: "SP Burgos", LogoURL: "https://cdn/logo1.png"}
	home.ID = uuid.New()
	away := clubmodels.Club{Name: "FC Barcelona", LogoURL: "https://cdn/logo2.png"}
	away.ID = uuid.New()
	clubRepo.clubs[home.ID] = home
	clubRepo.clubs[away.ID] = away

	match := newScheduledMatch(matchRepo, time.Now().Add(36*time.Hour))
	match.HomeClubID = home.ID
	match.AwayClubID = away.ID

	snackRepo := &stubSnackRepo{snacks: map[uuid.UUID]shopmodels.Snack{}}
	productRepo := &stubProductRepo{products: map[uuid.UUID]shopmodels.Product{}}
	addProduct(productRepo, "Sport T-shirt", 3250, `["M"]`)
	cartRepo := newStubCartRepo()
	newsRepo := &stubNewsRepo{items: []newsmodels.News{{ID: uuid.New(), Title: "Injuries in Sunday game", PublishedAt: time.Now(), IsActive: true}}}

	svc := homesvc.NewHomeService(
		userRepo, matchRepo, clubRepo,
		chantsvc.NewChantService(newStubChantRepo(), matchRepo, nil, nil),
		shopsvc.NewShopService(snackRepo, productRepo, cartRepo, newStubOrderRepo(cartRepo)),
		newssvc.NewNewsService(newsRepo),
	)

	resp, err := svc.Club(context.Background())
	if err != nil {
		t.Fatalf("Club failed: %v", err)
	}
	if len(resp.UpcomingMatches) != 1 {
		t.Fatalf("expected 1 upcoming match, got %d", len(resp.UpcomingMatches))
	}
	um := resp.UpcomingMatches[0]
	if um.HomeName != "SP Burgos" || um.AwayName != "FC Barcelona" {
		t.Fatalf("unexpected team names: %q vs %q", um.HomeName, um.AwayName)
	}
	if um.Status != "upcoming" {
		t.Fatalf("expected status upcoming, got %q", um.Status)
	}
	if um.CountdownSeconds <= 0 {
		t.Fatal("expected a positive countdown")
	}
	if len(resp.ClubStore) != 1 {
		t.Fatalf("expected 1 store item, got %d", len(resp.ClubStore))
	}
	if len(resp.ClubNews) != 1 {
		t.Fatalf("expected 1 news item, got %d", len(resp.ClubNews))
	}
}
