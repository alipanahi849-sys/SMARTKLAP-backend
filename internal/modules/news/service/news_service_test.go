package service

import (
	"context"
	"testing"
	"time"

	clubmodels "clap/internal/modules/club/models"
	"clap/internal/modules/news/dto"
	settingsmodels "clap/internal/modules/settings/models"
	"clap/pkg/newsfeed"

	"github.com/google/uuid"
)

type stubFeed struct {
	articles []newsfeed.Article
	pages    int
}

func (s stubFeed) Name() string  { return "stub" }
func (s stubFeed) Enabled() bool { return true }

func (s stubFeed) Search(context.Context, string, int, int) (*newsfeed.SearchResult, error) {
	return &newsfeed.SearchResult{Items: s.articles, Page: 1, TotalPages: s.pages}, nil
}

func (s stubFeed) Get(_ context.Context, providerID string) (*newsfeed.Article, error) {
	for i := range s.articles {
		if s.articles[i].ProviderID == providerID {
			item := s.articles[i]
			return &item, nil
		}
	}
	return nil, nil
}

type stubSettings struct {
	settings *settingsmodels.AppSettings
}

func (s stubSettings) Get(context.Context) (*settingsmodels.AppSettings, error) {
	return s.settings, nil
}

func (s stubSettings) Save(_ context.Context, settings *settingsmodels.AppSettings) error {
	*s.settings = *settings
	return nil
}

type stubClubs struct {
	byID map[uuid.UUID]*clubmodels.Club
}

func (s stubClubs) Create(context.Context, *clubmodels.Club) error { return nil }
func (s stubClubs) FindByID(_ context.Context, id uuid.UUID) (*clubmodels.Club, error) {
	return s.byID[id], nil
}
func (s stubClubs) FindByIDs(context.Context, []uuid.UUID) (map[uuid.UUID]clubmodels.Club, error) {
	return nil, nil
}
func (s stubClubs) FindAll(context.Context, int, int, map[string]string, string, string) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (s stubClubs) Search(context.Context, string, int, int) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (s stubClubs) FindByProviderTeamID(context.Context, string, string) (*clubmodels.Club, error) {
	return nil, nil
}
func (s stubClubs) Update(context.Context, *clubmodels.Club) error { return nil }
func (s stubClubs) Delete(context.Context, uuid.UUID) error        { return nil }

func TestListUsesProviderArticlesForNewsClub(t *testing.T) {
	clubID := uuid.MustParse("a3000000-0000-4000-8000-000000000099")
	club := &clubmodels.Club{ID: clubID, Name: "Manchester United"}
	published := time.Date(2026, 7, 14, 18, 20, 15, 0, time.UTC)
	article := newsfeed.Article{
		ProviderID:  "football/2026/jul/14/tielemans-united",
		Title:       "Tielemans joins Manchester United",
		BodyHTML:    "<p>Manchester United have confirmed the signing.</p>",
		ImageURL:    "https://media.guim.co.uk/thumb.jpg",
		PublishedAt: published,
	}

	svc := NewNewsService(
		stubFeed{articles: []newsfeed.Article{article}, pages: 1},
		nil,
		stubSettings{settings: &settingsmodels.AppSettings{ID: 1, NewsClubID: &clubID, NewsClub: club}},
		stubClubs{byID: map[uuid.UUID]*clubmodels.Club{clubID: club}},
		nil,
	)

	result, err := svc.List(context.Background(), dto.NewsListFilters{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("len=%d", len(result.Items))
	}
	if result.Items[0].Title != article.Title {
		t.Fatalf("title=%q", result.Items[0].Title)
	}
	if result.Items[0].ImageURL != article.ImageURL {
		t.Fatalf("image=%q", result.Items[0].ImageURL)
	}

	detail, err := svc.GetByID(context.Background(), result.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.BodyHTML != article.BodyHTML {
		t.Fatalf("body=%q", detail.BodyHTML)
	}
	if detail.ClubID == nil || *detail.ClubID != clubID {
		t.Fatalf("club=%v", detail.ClubID)
	}
}

func TestListFallsBackToFeaturedClub(t *testing.T) {
	clubID := uuid.MustParse("a3000000-0000-4000-8000-000000000088")
	club := &clubmodels.Club{ID: clubID, Name: "Arsenal"}
	svc := NewNewsService(
		stubFeed{articles: []newsfeed.Article{{
			ProviderID: "football/2026/aug/01/arsenal",
			Title:      "Arsenal news",
		}}, pages: 1},
		nil,
		stubSettings{settings: &settingsmodels.AppSettings{ID: 1, FeaturedClubID: &clubID, FeaturedClub: club}},
		stubClubs{byID: map[uuid.UUID]*clubmodels.Club{clubID: club}},
		nil,
	)
	result, err := svc.List(context.Background(), dto.NewsListFilters{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Title != "Arsenal news" {
		t.Fatalf("%+v", result.Items)
	}
}

type stubProvisioner struct {
	club *clubmodels.Club
}

func (s stubProvisioner) EnsureClubFromProvider(context.Context, string) (*clubmodels.Club, error) {
	return s.club, nil
}

func TestSetNewsClubFromName(t *testing.T) {
	settings := &settingsmodels.AppSettings{ID: 1}
	created := &clubmodels.Club{}
	svc := NewNewsService(
		stubFeed{},
		nil,
		stubSettings{settings: settings},
		&capturingClubs{created: created},
		nil,
	)

	got, err := svc.SetNewsClub(context.Background(), dto.SetNewsClubRequest{Name: "Arsenal"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Arsenal" || settings.NewsClubID == nil {
		t.Fatalf("got=%+v settings=%+v", got, settings)
	}
	if created.Name != "Arsenal" || !created.IsActive {
		t.Fatalf("created=%+v", created)
	}
}

type capturingClubs struct {
	created *clubmodels.Club
}

func (c *capturingClubs) Create(_ context.Context, club *clubmodels.Club) error {
	club.ID = uuid.MustParse("a3000000-0000-4000-8000-000000000055")
	*c.created = *club
	return nil
}
func (c *capturingClubs) FindByID(context.Context, uuid.UUID) (*clubmodels.Club, error) {
	return nil, nil
}
func (c *capturingClubs) FindByIDs(context.Context, []uuid.UUID) (map[uuid.UUID]clubmodels.Club, error) {
	return nil, nil
}
func (c *capturingClubs) FindAll(context.Context, int, int, map[string]string, string, string) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (c *capturingClubs) Search(context.Context, string, int, int) ([]clubmodels.Club, int64, error) {
	return nil, 0, nil
}
func (c *capturingClubs) FindByProviderTeamID(context.Context, string, string) (*clubmodels.Club, error) {
	return nil, nil
}
func (c *capturingClubs) Update(context.Context, *clubmodels.Club) error { return nil }
func (c *capturingClubs) Delete(context.Context, uuid.UUID) error        { return nil }
