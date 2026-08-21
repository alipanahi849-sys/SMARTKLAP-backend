package unit

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sort"
	"testing"
	"time"

	chantdto "clap/internal/modules/chant/dto"
	chantmodels "clap/internal/modules/chant/models"
	chantrepo "clap/internal/modules/chant/repository"
	chantsvc "clap/internal/modules/chant/service"
	matchmodels "clap/internal/modules/match/models"
	settingsmodels "clap/internal/modules/settings/models"
	settingsrepo "clap/internal/modules/settings/repository"
	songmodels "clap/internal/modules/song/models"
	usermodels "clap/internal/modules/user/models"
	videodto "clap/internal/modules/video/dto"
	videomodels "clap/internal/modules/video/models"
	videorepo "clap/internal/modules/video/repository"
	videosvc "clap/internal/modules/video/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── match stubs (shared by chant tests) ──────────────────────────────────────

type stubMatchRepo struct {
	matches map[uuid.UUID]*matchmodels.Match
}

func newStubMatchRepo() *stubMatchRepo {
	return &stubMatchRepo{matches: map[uuid.UUID]*matchmodels.Match{}}
}

func (r *stubMatchRepo) Create(_ context.Context, m *matchmodels.Match) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	r.matches[m.ID] = m
	return nil
}

func (r *stubMatchRepo) FindByID(_ context.Context, id uuid.UUID) (*matchmodels.Match, error) {
	if m, ok := r.matches[id]; ok {
		return m, nil
	}
	return nil, sharederrors.NewNotFound("Match not found", nil)
}

func (r *stubMatchRepo) FindAll(context.Context, int, int, map[string]string, string, string) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *stubMatchRepo) FindBySeason(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *stubMatchRepo) FindByLeague(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *stubMatchRepo) FindByClub(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}

func (r *stubMatchRepo) FindUpcoming(_ context.Context, _, _ int) ([]matchmodels.Match, int64, error) {
	var result []matchmodels.Match
	for _, m := range r.matches {
		if m.Status == "scheduled" {
			result = append(result, *m)
		}
	}
	return result, int64(len(result)), nil
}

func (r *stubMatchRepo) FindLive(context.Context) ([]matchmodels.Match, error) {
	var result []matchmodels.Match
	for _, m := range r.matches {
		if m.Status == "live" {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (r *stubMatchRepo) FindByProviderMatchID(_ context.Context, _, id string) (*matchmodels.Match, error) {
	for _, m := range r.matches {
		if m.ProviderMatchID == id {
			return m, nil
		}
	}
	return nil, nil
}

func (r *stubMatchRepo) FindCurrentForClub(_ context.Context, clubID uuid.UUID) (*matchmodels.Match, error) {
	for _, m := range r.matches {
		if m.HomeClubID == clubID || m.AwayClubID == clubID {
			return m, nil
		}
	}
	return nil, nil
}

func (r *stubMatchRepo) ListForClub(_ context.Context, clubID uuid.UUID, pastCount, futureCount int) ([]matchmodels.Match, error) {
	var result []matchmodels.Match
	for _, m := range r.matches {
		if m.HomeClubID == clubID || m.AwayClubID == clubID {
			result = append(result, *m)
		}
	}
	limit := pastCount + futureCount + 1
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubMatchRepo) FindLiveByClub(_ context.Context, clubID uuid.UUID) ([]matchmodels.Match, error) {
	var result []matchmodels.Match
	for _, m := range r.matches {
		if (m.HomeClubID == clubID || m.AwayClubID == clubID) && (m.Status == "live" || m.Status == "halftime") {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (r *stubMatchRepo) Update(_ context.Context, m *matchmodels.Match) error {
	r.matches[m.ID] = m
	return nil
}

func (r *stubMatchRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.matches, id)
	return nil
}

func newScheduledMatch(repo *stubMatchRepo, kickoff time.Time) *matchmodels.Match {
	m := &matchmodels.Match{
		ID:            uuid.New(),
		Status:        "scheduled",
		MatchDateTime: kickoff,
		HomeClubID:    uuid.New(),
		AwayClubID:    uuid.New(),
	}
	repo.matches[m.ID] = m
	return m
}

// ─── chant stubs ──────────────────────────────────────────────────────────────

// completionKey identifies an award the way the partial unique indexes do:
// online chants by chant, catalog songs by song.
type completionKey struct {
	target uuid.UUID
	user   uuid.UUID
	source string
}

type listenKey struct {
	user   uuid.UUID
	song   uuid.UUID
	source string
}

// stubAttempt mirrors a chant_completions row: the slot is taken either way,
// but only a completed one is worth points.
type stubAttempt struct {
	status string
	points int
	at     time.Time
}

type stubChantRepo struct {
	chants      map[uuid.UUID]*chantmodels.Chant
	songs       map[uuid.UUID]*songmodels.Song
	completions map[completionKey]stubAttempt
	listens     map[listenKey]time.Time
	userPoints  map[uuid.UUID]int
}

var _ chantrepo.ChantRepository = (*stubChantRepo)(nil)

func newStubChantRepo() *stubChantRepo {
	return &stubChantRepo{
		chants:      map[uuid.UUID]*chantmodels.Chant{},
		songs:       map[uuid.UUID]*songmodels.Song{},
		completions: map[completionKey]stubAttempt{},
		listens:     map[listenKey]time.Time{},
		userPoints:  map[uuid.UUID]int{},
	}
}

func keyFor(target chantrepo.CompletionTarget) completionKey {
	key := completionKey{user: target.UserID, source: target.Source}
	if target.Source == chantmodels.SourceCatalog {
		key.target = target.SongID
		return key
	}
	if target.ChantID != nil {
		key.target = *target.ChantID
	}
	return key
}

// stubSettingsRepo lets a test pin the admin-configurable point values.
type stubSettingsRepo struct {
	settings settingsmodels.AppSettings
}

var _ settingsrepo.SettingsRepository = (*stubSettingsRepo)(nil)

func newStubSettingsRepo(songPoints, onlinePoints int) *stubSettingsRepo {
	return &stubSettingsRepo{settings: settingsmodels.AppSettings{
		ID:                1,
		ChantSongPoints:   songPoints,
		ChantOnlinePoints: onlinePoints,
		ChantDailyTarget:  500,
	}}
}

func (r *stubSettingsRepo) Get(_ context.Context) (*settingsmodels.AppSettings, error) {
	copied := r.settings
	return &copied, nil
}

func (r *stubSettingsRepo) Save(_ context.Context, settings *settingsmodels.AppSettings) error {
	r.settings = *settings
	return nil
}

func (r *stubChantRepo) FindByID(_ context.Context, id uuid.UUID) (*chantmodels.Chant, error) {
	if c, ok := r.chants[id]; ok {
		return c, nil
	}
	return nil, sharederrors.NewNotFound("Chant not found", nil)
}

func (r *stubChantRepo) FindByMatchAfter(_ context.Context, matchID uuid.UUID, _ string, limit int, _ *chantrepo.ChantCursorAnchor) ([]chantmodels.Chant, error) {
	var result []chantmodels.Chant
	for _, c := range r.chants {
		if c.MatchID == matchID {
			result = append(result, *c)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubChantRepo) FindStartingBetween(_ context.Context, from, to time.Time) ([]chantmodels.Chant, error) {
	var result []chantmodels.Chant
	for _, c := range r.chants {
		if c.IsActive && c.ScheduledAt.After(from) && !c.ScheduledAt.After(to) {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (r *stubChantRepo) FindActiveByMatch(_ context.Context, matchID uuid.UUID, now time.Time) (*chantmodels.Chant, error) {
	for _, c := range r.chants {
		if !c.IsActive || c.MatchID != matchID {
			continue
		}
		end := c.ScheduledAt.Add(time.Duration(c.DurationSeconds) * time.Second)
		if !c.ScheduledAt.After(now) && end.After(now) {
			return c, nil
		}
	}
	return nil, nil
}

func (r *stubChantRepo) CompletedChantIDs(_ context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	done := map[uuid.UUID]bool{}
	for _, id := range chantIDs {
		attempt, ok := r.completions[completionKey{target: id, user: userID, source: chantmodels.SourceOnline}]
		if ok && attempt.status == chantmodels.StatusCompleted {
			done[id] = true
		}
	}
	return done, nil
}

func (r *stubChantRepo) TodayPoints(_ context.Context, userID uuid.UUID) (int, error) {
	total := 0
	for key, attempt := range r.completions {
		if key.user == userID {
			total += attempt.points
		}
	}
	return total, nil
}

func (r *stubChantRepo) TodayCompletions(_ context.Context, _ uuid.UUID, _ int) ([]chantmodels.ChantCompletion, map[uuid.UUID]chantmodels.Chant, error) {
	return nil, map[uuid.UUID]chantmodels.Chant{}, nil
}

func (r *stubChantRepo) Complete(_ context.Context, target chantrepo.CompletionTarget) (int, bool, error) {
	key := keyFor(target)
	if _, settled := r.completions[key]; settled {
		return r.userPoints[target.UserID], false, nil
	}
	r.completions[key] = stubAttempt{status: chantmodels.StatusCompleted, points: target.Points, at: time.Now().UTC()}
	r.userPoints[target.UserID] += target.Points
	return r.userPoints[target.UserID], true, nil
}

func (r *stubChantRepo) Cancel(_ context.Context, target chantrepo.CompletionTarget) (bool, error) {
	key := keyFor(target)
	if _, settled := r.completions[key]; settled {
		return false, nil
	}
	r.completions[key] = stubAttempt{status: chantmodels.StatusCancelled, at: time.Now().UTC()}
	return true, nil
}

func (r *stubChantRepo) FindSongByID(_ context.Context, id uuid.UUID) (*songmodels.Song, error) {
	if s, ok := r.songs[id]; ok {
		return s, nil
	}
	return nil, sharederrors.NewNotFound("Chant not found", nil)
}

func (r *stubChantRepo) FindCatalogSongs(_ context.Context, _ string, limit int) ([]songmodels.Song, error) {
	var result []songmodels.Song
	for _, s := range r.songs {
		result = append(result, *s)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubChantRepo) CompletedSongIDs(_ context.Context, userID uuid.UUID, songIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	done := map[uuid.UUID]bool{}
	for _, id := range songIDs {
		attempt, ok := r.completions[completionKey{target: id, user: userID, source: chantmodels.SourceCatalog}]
		if ok && attempt.status == chantmodels.StatusCompleted {
			done[id] = true
		}
	}
	return done, nil
}

func (r *stubChantRepo) IsCompleted(_ context.Context, target chantrepo.CompletionTarget) (bool, error) {
	_, ok := r.completions[keyFor(target)]
	return ok, nil
}

func (r *stubChantRepo) StartListenSession(_ context.Context, userID, songID uuid.UUID, source string) error {
	r.listens[listenKey{user: userID, song: songID, source: source}] = time.Now().UTC()
	return nil
}

func (r *stubChantRepo) ListenStartedAt(_ context.Context, userID, songID uuid.UUID, source string) (*time.Time, error) {
	if at, ok := r.listens[listenKey{user: userID, song: songID, source: source}]; ok {
		return &at, nil
	}
	return nil, nil
}

func (r *stubChantRepo) TodayProgramFeed(_ context.Context, userID uuid.UUID, limit int) ([]chantrepo.ProgramCompletion, error) {
	var rows []chantrepo.ProgramCompletion
	for key, attempt := range r.completions {
		if key.user != userID {
			continue
		}
		title := ""
		if key.source == chantmodels.SourceCatalog {
			if song, ok := r.songs[key.target]; ok {
				title = song.Title
			}
		} else if chant, ok := r.chants[key.target]; ok {
			title = chant.Title
		}
		rows = append(rows, chantrepo.ProgramCompletion{
			ID:           uuid.New(),
			Title:        title,
			Status:       attempt.status,
			PointsEarned: attempt.points,
			CreatedAt:    attempt.at,
		})
	}
	// Newest attempts survive the limit, then the feed reads oldest first.
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.Before(rows[j].CreatedAt) })
	return rows, nil
}

func (r *stubChantRepo) PendingChantsForMatch(_ context.Context, userID, matchID uuid.UUID, limit int) ([]chantrepo.PendingChant, error) {
	var rows []chantrepo.PendingChant
	for _, c := range r.chants {
		if !c.IsActive || c.MatchID != matchID {
			continue
		}
		key := completionKey{target: c.ID, user: userID, source: chantmodels.SourceOnline}
		if _, settled := r.completions[key]; settled {
			continue
		}
		rows = append(rows, chantrepo.PendingChant{ID: c.ID, Title: c.Title, ScheduledAt: c.ScheduledAt})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ScheduledAt.Before(rows[j].ScheduledAt) })
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (r *stubChantRepo) CreateChant(_ context.Context, chant *chantmodels.Chant) error {
	if chant.ID == uuid.Nil {
		chant.ID = uuid.New()
	}
	r.chants[chant.ID] = chant
	return nil
}

func (r *stubChantRepo) FindScheduled(_ context.Context, matchID *uuid.UUID, limit int) ([]chantmodels.Chant, error) {
	var result []chantmodels.Chant
	for _, c := range r.chants {
		if !c.IsActive {
			continue
		}
		if matchID != nil && c.MatchID != *matchID {
			continue
		}
		result = append(result, *c)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubChantRepo) DeactivateChant(_ context.Context, id uuid.UUID) error {
	c, ok := r.chants[id]
	if !ok {
		return sharederrors.NewNotFound("Chant not found", nil)
	}
	c.IsActive = false
	return nil
}

// ─── chant tests ──────────────────────────────────────────────────────────────

// newChantService wires the service with admin point values of 100 for catalog
// songs and 250 for online chants.
func newChantService(chantRepo *stubChantRepo, matchRepo *stubMatchRepo) chantsvc.ChantService {
	return chantsvc.NewChantService(chantRepo, matchRepo, nil, newStubSettingsRepo(100, 250), nil)
}

func TestChant_CompleteAwardsOnlinePointsOnce(t *testing.T) {
	chantRepo := newStubChantRepo()
	svc := newChantService(chantRepo, newStubMatchRepo())

	chant := &chantmodels.Chant{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		SongID:  uuid.New(),
		Title:   "Chant number 1",
		Points:  100,
	}
	chantRepo.chants[chant.ID] = chant
	userID := uuid.New()

	// The award comes from the admin setting, not the chant's own column.
	resp, err := svc.Complete(context.Background(), userID, chant.ID, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if !resp.IsDone || resp.PointsEarned != 250 || resp.TotalPoints != 250 {
		t.Fatalf("unexpected response: %+v", resp)
	}

	resp, err = svc.Complete(context.Background(), userID, chant.ID, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("second Complete failed: %v", err)
	}
	if !resp.IsDone || resp.PointsEarned != 0 || resp.TotalPoints != 250 {
		t.Fatalf("unexpected idempotent response: %+v", resp)
	}
}

func TestChant_CancelBurnsTheChantForPoints(t *testing.T) {
	chantRepo := newStubChantRepo()
	svc := newChantService(chantRepo, newStubMatchRepo())

	chant := &chantmodels.Chant{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		SongID:  uuid.New(),
		Title:   "Chant number 1",
		Points:  100,
	}
	chantRepo.chants[chant.ID] = chant
	userID := uuid.New()

	if err := svc.Cancel(context.Background(), userID, chant.ID); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	// Walking out settles the chant, so singing it afterwards pays nothing.
	resp, err := svc.Complete(context.Background(), userID, chant.ID, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("Complete after cancel failed: %v", err)
	}
	if resp.PointsEarned != 0 || resp.TotalPoints != 0 {
		t.Fatalf("cancelled chant still paid out: %+v", resp)
	}

	// The Chants list must not badge it as earned either.
	done, err := chantRepo.CompletedChantIDs(context.Background(), userID, []uuid.UUID{chant.ID})
	if err != nil {
		t.Fatalf("CompletedChantIDs failed: %v", err)
	}
	if done[chant.ID] {
		t.Fatal("cancelled chant reported as completed")
	}
}

func TestChant_CancelLosesToAFinishedChant(t *testing.T) {
	chantRepo := newStubChantRepo()
	svc := newChantService(chantRepo, newStubMatchRepo())

	chant := &chantmodels.Chant{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		SongID:  uuid.New(),
		Title:   "Chant number 1",
		Points:  100,
	}
	chantRepo.chants[chant.ID] = chant
	userID := uuid.New()

	if _, err := svc.Complete(context.Background(), userID, chant.ID, chantmodels.SourceOnline); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	// Leaving the screen after the song ended must not undo the award.
	if err := svc.Cancel(context.Background(), userID, chant.ID); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	total, err := chantRepo.TodayPoints(context.Background(), userID)
	if err != nil {
		t.Fatalf("TodayPoints failed: %v", err)
	}
	if total != 250 {
		t.Fatalf("expected the award to survive cancellation, got %d", total)
	}
	done, err := chantRepo.CompletedChantIDs(context.Background(), userID, []uuid.UUID{chant.ID})
	if err != nil {
		t.Fatalf("CompletedChantIDs failed: %v", err)
	}
	if !done[chant.ID] {
		t.Fatal("finished chant stopped counting as completed")
	}
}

func TestChant_CatalogCompleteAwardsSongPointsOncePerSong(t *testing.T) {
	chantRepo := newStubChantRepo()
	svc := newChantService(chantRepo, newStubMatchRepo())

	song := &songmodels.Song{ID: uuid.New(), Title: "We will rock you", IsActive: true}
	chantRepo.songs[song.ID] = song
	userID := uuid.New()

	resp, err := svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.PointsEarned != 100 || resp.TotalPoints != 100 {
		t.Fatalf("unexpected catalog award: %+v", resp)
	}

	resp, err = svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err != nil {
		t.Fatalf("second Complete failed: %v", err)
	}
	if resp.PointsEarned != 0 || resp.TotalPoints != 100 {
		t.Fatalf("catalog song paid out twice: %+v", resp)
	}
}

func TestChant_CompleteRejectedBeforeSongEnds(t *testing.T) {
	chantRepo := newStubChantRepo()
	svc := newChantService(chantRepo, newStubMatchRepo())

	song := &songmodels.Song{ID: uuid.New(), Title: "Long song", Duration: 180, IsActive: true}
	chantRepo.songs[song.ID] = song
	userID := uuid.New()

	// Lyrics were opened a moment ago, so a three-minute song cannot be over.
	if err := chantRepo.StartListenSession(context.Background(), userID, song.ID, chantmodels.SourceCatalog); err != nil {
		t.Fatalf("StartListenSession failed: %v", err)
	}

	_, err := svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err == nil {
		t.Fatal("expected the short listen to be rejected")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", status)
	}

	// Backdate the session past the song length and the award goes through.
	chantRepo.listens[listenKey{user: userID, song: song.ID, source: chantmodels.SourceCatalog}] =
		time.Now().UTC().Add(-4 * time.Minute)

	resp, err := svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err != nil {
		t.Fatalf("Complete after full listen failed: %v", err)
	}
	if resp.PointsEarned != 100 {
		t.Fatalf("expected 100 points after a full listen, got %d", resp.PointsEarned)
	}
}

func TestChant_CompleteUnknownChantNotFound(t *testing.T) {
	svc := newChantService(newStubChantRepo(), newStubMatchRepo())

	_, err := svc.Complete(context.Background(), uuid.New(), uuid.New(), chantmodels.SourceOnline)
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestChant_ListGroupsIntoSections(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := newChantService(chantRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	chantRepo.chants[uuid.New()] = &chantmodels.Chant{
		ID:          uuid.New(),
		MatchID:     match.ID,
		Title:       "Today chant",
		ScheduledAt: time.Now().UTC().Add(time.Hour),
	}
	for id, c := range chantRepo.chants {
		c.ID = id
	}

	resp, err := svc.List(context.Background(), uuid.New(), chantdto.ChantListFilters{
		MatchID: &match.ID,
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Sections) == 0 {
		t.Fatal("expected at least one section")
	}
}

func TestChant_ListSeparatesCatalogFromOnlineChants(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := newChantService(chantRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	chant := &chantmodels.Chant{
		ID:          uuid.New(),
		MatchID:     match.ID,
		SongID:      uuid.New(),
		Title:       "Scheduled chant",
		ScheduledAt: time.Now().UTC().Add(time.Hour),
		IsActive:    true,
	}
	chantRepo.chants[chant.ID] = chant

	song := &songmodels.Song{ID: uuid.New(), Title: "Library song", Duration: 120, IsActive: true}
	chantRepo.songs[song.ID] = song

	resp, err := svc.List(context.Background(), uuid.New(), chantdto.ChantListFilters{
		MatchID: &match.ID,
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var online, catalog *chantdto.ChantItem
	for _, section := range resp.Sections {
		for i, item := range section.Items {
			switch item.Source {
			case chantmodels.SourceOnline:
				online = &section.Items[i]
			case chantmodels.SourceCatalog:
				catalog = &section.Items[i]
			}
		}
	}

	if online == nil {
		t.Fatal("expected the scheduled chant in the list")
	}
	if catalog == nil {
		t.Fatal("expected the song catalog in the list")
	}
	// is_preview would open the silent lyrics-only view; the Chants screen must
	// hand out playable chants so a full listen can be scored.
	if online.IsPreview || catalog.IsPreview {
		t.Fatal("everything on the Chants screen must be playable, not silent")
	}
	if online.SongPoints != 250 || catalog.SongPoints != 100 {
		t.Fatalf("online and catalog must score differently: %d vs %d", online.SongPoints, catalog.SongPoints)
	}
}

// The Home programme is the fan's to-do list: every online chant defined for the
// match belongs on it, however far off kickoff is, and each one leaves as soon
// as it is settled.
func TestChant_ProgramListsMatchChantsUntilTheyAreSettled(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := newChantService(chantRepo, matchRepo)

	kickoff := time.Now().UTC().Add(72 * time.Hour)
	match := newScheduledMatch(matchRepo, kickoff)
	first := &chantmodels.Chant{
		ID:          uuid.New(),
		MatchID:     match.ID,
		SongID:      uuid.New(),
		Title:       "Chant one",
		ScheduledAt: kickoff,
		IsActive:    true,
	}
	second := &chantmodels.Chant{
		ID:          uuid.New(),
		MatchID:     match.ID,
		SongID:      uuid.New(),
		Title:       "Chant two",
		ScheduledAt: kickoff.Add(30 * time.Minute),
		IsActive:    true,
	}
	chantRepo.chants[first.ID] = first
	chantRepo.chants[second.ID] = second

	userID := uuid.New()
	resp, err := svc.Program(context.Background(), userID, 20)
	if err != nil {
		t.Fatalf("Program failed: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected both scheduled chants, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != first.ID.String() {
		t.Fatalf("pending chants must be listed soonest first, got %q", resp.Items[0].Title)
	}
	if resp.Items[0].IsDone || resp.Items[0].Points != 250 {
		t.Fatalf("a chant still to sing must be pending and worth the online points: %+v", resp.Items[0])
	}

	if _, err := svc.Complete(context.Background(), userID, first.ID, chantmodels.SourceOnline); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	resp, err = svc.Program(context.Background(), userID, 20)
	if err != nil {
		t.Fatalf("Program after completion failed: %v", err)
	}
	for _, item := range resp.Items {
		if item.ID == first.ID.String() {
			t.Fatal("a sung chant must leave the programme's pending list")
		}
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected the sung chant plus the one left to sing, got %+v", resp.Items)
	}
	// What is left to sing leads the card; the day's history sits under it.
	if resp.Items[0].ID != second.ID.String() || resp.Items[0].IsDone {
		t.Fatalf("the unsung chant must lead the programme: %+v", resp.Items[0])
	}
	if resp.Items[1].Title != "Chant one" || !resp.Items[1].IsDone {
		t.Fatalf("the settled chant must follow it: %+v", resp.Items[1])
	}
}

func TestChant_SetOnlineChantFromCatalogSong(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := newChantService(chantRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	song := &songmodels.Song{ID: uuid.New(), Title: "Library song", Duration: 120, IsActive: true}
	chantRepo.songs[song.ID] = song

	created, err := svc.SetOnlineChant(context.Background(), uuid.New(), chantdto.SetOnlineChantRequest{
		SongID:      song.ID,
		MatchID:     match.ID,
		ScheduledAt: time.Now().UTC().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("SetOnlineChant failed: %v", err)
	}
	if created.SongID != song.ID || created.Title != song.Title {
		t.Fatalf("online chant should inherit the song: %+v", created)
	}
	if created.DurationSeconds != song.Duration {
		t.Fatalf("expected duration %d, got %d", song.Duration, created.DurationSeconds)
	}
	if created.Points != 250 {
		t.Fatalf("expected the configured online points, got %d", created.Points)
	}
}

// Scheduling a song as an online chant is a fresh event every time, so the
// "already earned this" rule is scoped to one scheduled chant. It never spreads
// to the song itself, which would let one live performance spoil the next.
func TestChant_EachScheduledChantIsItsOwnEarningOpportunity(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := newChantService(chantRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	song := &songmodels.Song{ID: uuid.New(), Title: "Rap God", Duration: 120, IsActive: true}
	chantRepo.songs[song.ID] = song
	adminID := uuid.New()
	userID := uuid.New()

	// Scheduled in the past so the full-listen guard is already satisfied.
	schedule := func() uuid.UUID {
		created, err := svc.SetOnlineChant(context.Background(), adminID, chantdto.SetOnlineChantRequest{
			SongID:      song.ID,
			MatchID:     match.ID,
			ScheduledAt: time.Now().UTC().Add(-5 * time.Minute),
		})
		if err != nil {
			t.Fatalf("SetOnlineChant failed: %v", err)
		}
		return created.ID
	}

	first := schedule()
	resp, err := svc.Complete(context.Background(), userID, first, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.PointsEarned != 250 {
		t.Fatalf("expected the first chant to pay 250, got %d", resp.PointsEarned)
	}

	// Same song, scheduled again: the fan gets another shot at the points.
	second := schedule()
	resp, err = svc.Complete(context.Background(), userID, second, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("Complete on the re-scheduled chant failed: %v", err)
	}
	if resp.PointsEarned != 250 || resp.TotalPoints != 500 {
		t.Fatalf("re-scheduled chant should pay again: %+v", resp)
	}

	// Walking out of a third one settles only that one.
	third := schedule()
	if err := svc.Cancel(context.Background(), userID, third); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	fourth := schedule()
	resp, err = svc.Complete(context.Background(), userID, fourth, chantmodels.SourceOnline)
	if err != nil {
		t.Fatalf("Complete after cancelling a different chant failed: %v", err)
	}
	if resp.PointsEarned != 250 {
		t.Fatalf("a cancelled chant must not spoil the next one: %+v", resp)
	}

	// The library entry for the same song is still a separate, once-only award.
	// Catalog songs have no schedule to fall back on, so stand in for the fan
	// having opened the lyrics long enough ago to have heard the whole track.
	chantRepo.listens[listenKey{user: userID, song: song.ID, source: chantmodels.SourceCatalog}] =
		time.Now().UTC().Add(-5 * time.Minute)

	resp, err = svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err != nil {
		t.Fatalf("catalog Complete failed: %v", err)
	}
	if resp.PointsEarned != 100 {
		t.Fatalf("expected the catalog award to be untouched, got %d", resp.PointsEarned)
	}
	resp, err = svc.Complete(context.Background(), userID, song.ID, chantmodels.SourceCatalog)
	if err != nil {
		t.Fatalf("second catalog Complete failed: %v", err)
	}
	if resp.PointsEarned != 0 {
		t.Fatalf("the catalog song must pay only once, got %d", resp.PointsEarned)
	}
}

func TestChant_UpdatePointsSettings(t *testing.T) {
	settings := newStubSettingsRepo(100, 250)
	svc := chantsvc.NewChantService(newStubChantRepo(), newStubMatchRepo(), nil, settings, nil)

	songPoints := 40
	onlinePoints := 900
	updated, err := svc.UpdatePointsSettings(context.Background(), chantdto.UpdateChantPointsRequest{
		ChantSongPoints:   &songPoints,
		ChantOnlinePoints: &onlinePoints,
	})
	if err != nil {
		t.Fatalf("UpdatePointsSettings failed: %v", err)
	}
	if updated.ChantSongPoints != 40 || updated.ChantOnlinePoints != 900 {
		t.Fatalf("unexpected settings: %+v", updated)
	}
	// Unspecified values are left alone.
	if updated.ChantDailyTarget != 500 {
		t.Fatalf("daily target should be untouched, got %d", updated.ChantDailyTarget)
	}
}

// ─── video stubs ──────────────────────────────────────────────────────────────

type stubVideoRepo struct {
	videos map[uuid.UUID]*videomodels.Video
	likes  map[uuid.UUID]map[uuid.UUID]bool
	views  map[uuid.UUID]map[uuid.UUID]bool
}

func newStubVideoRepo() *stubVideoRepo {
	return &stubVideoRepo{
		videos: map[uuid.UUID]*videomodels.Video{},
		likes:  map[uuid.UUID]map[uuid.UUID]bool{},
		views:  map[uuid.UUID]map[uuid.UUID]bool{},
	}
}

func (r *stubVideoRepo) Create(_ context.Context, v *videomodels.Video) error {
	r.videos[v.ID] = v
	return nil
}

func (r *stubVideoRepo) FindByID(_ context.Context, id uuid.UUID) (*videomodels.Video, error) {
	if v, ok := r.videos[id]; ok {
		return v, nil
	}
	return nil, sharederrors.NewNotFound("Video not found", nil)
}

func (r *stubVideoRepo) FeedAfter(_ context.Context, limit int, _ *videorepo.VideoCursorAnchor) ([]videomodels.Video, error) {
	var result []videomodels.Video
	for _, v := range r.videos {
		if v.Status == videomodels.StatusPublished {
			result = append(result, *v)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubVideoRepo) ByUserAfter(_ context.Context, userID uuid.UUID, limit int, _ *videorepo.VideoCursorAnchor) ([]videomodels.Video, error) {
	var result []videomodels.Video
	for _, v := range r.videos {
		if v.UserID == userID {
			result = append(result, *v)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *stubVideoRepo) LikedVideoIDs(_ context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	liked := map[uuid.UUID]bool{}
	for _, id := range videoIDs {
		if users, ok := r.likes[id]; ok && users[userID] {
			liked[id] = true
		}
	}
	return liked, nil
}

func (r *stubVideoRepo) Like(_ context.Context, videoID, userID uuid.UUID) (bool, error) {
	if users, ok := r.likes[videoID]; ok && users[userID] {
		return false, nil
	}
	if r.likes[videoID] == nil {
		r.likes[videoID] = map[uuid.UUID]bool{}
	}
	r.likes[videoID][userID] = true
	r.videos[videoID].LikesCount++
	return true, nil
}

func (r *stubVideoRepo) Unlike(_ context.Context, videoID, userID uuid.UUID) (bool, error) {
	if users, ok := r.likes[videoID]; ok && users[userID] {
		delete(users, userID)
		r.videos[videoID].LikesCount--
		return true, nil
	}
	return false, nil
}

func (r *stubVideoRepo) SeenVideoIDs(_ context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	seen := map[uuid.UUID]bool{}
	for _, id := range videoIDs {
		if users, ok := r.views[id]; ok && users[userID] {
			seen[id] = true
		}
	}
	return seen, nil
}

func (r *stubVideoRepo) MarkSeen(_ context.Context, videoID, userID uuid.UUID) (bool, error) {
	if users, ok := r.views[videoID]; ok && users[userID] {
		return false, nil
	}
	if r.views[videoID] == nil {
		r.views[videoID] = map[uuid.UUID]bool{}
	}
	r.views[videoID][userID] = true
	r.videos[videoID].ViewsCount++
	return true, nil
}

type stubProfileRepo struct{}

func (stubProfileRepo) Create(context.Context, *usermodels.Profile) error { return nil }
func (stubProfileRepo) FindByID(context.Context, uuid.UUID) (*usermodels.Profile, error) {
	return nil, sharederrors.NewNotFound("Profile not found", nil)
}
func (stubProfileRepo) FindByUserID(context.Context, uuid.UUID) (*usermodels.Profile, error) {
	return nil, sharederrors.NewNotFound("Profile not found", nil)
}
func (stubProfileRepo) FindByUserIDs(context.Context, []uuid.UUID) (map[uuid.UUID]*usermodels.Profile, error) {
	return map[uuid.UUID]*usermodels.Profile{}, nil
}
func (stubProfileRepo) Update(context.Context, *usermodels.Profile) error { return nil }
func (stubProfileRepo) Delete(context.Context, uuid.UUID) error           { return nil }

// memoryStorage is an in-memory StorageProvider for upload tests.
type memoryStorage struct {
	objects map[string][]byte
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{objects: map[string][]byte{}}
}

func (s *memoryStorage) Upload(_ context.Context, key string, reader io.Reader, _ string, _ int64) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.objects[key] = data
	return nil
}

func (s *memoryStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *memoryStorage) GenerateSignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://storage.test/" + key, nil
}

func (s *memoryStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := s.objects[key]
	return ok, nil
}

func (s *memoryStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.objects[key])), nil
}

// makeFileHeader builds a real multipart.FileHeader for upload tests.
func makeFileHeader(t *testing.T, filename, contentType string, size int) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="media"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("CreatePart failed: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("x"), size)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(size) + 1024)
	if err != nil {
		t.Fatalf("ReadForm failed: %v", err)
	}
	return form.File["media"][0]
}

// ─── video tests ──────────────────────────────────────────────────────────────

func TestVideo_UploadPublishesAndExtractsHashtags(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	file := makeFileHeader(t, "clip.mp4", "video/mp4", 1024)
	resp, err := svc.Upload(context.Background(), uuid.New(), file, "video", "Great goal #Burgos #chant")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if resp.Status != videomodels.StatusPublished {
		t.Fatalf("expected published, got %q", resp.Status)
	}
	if resp.VideoURL == nil || *resp.VideoURL == "" {
		t.Fatal("expected a video URL")
	}

	stored := videoRepo.videos[resp.ID]
	if stored.Tags != `["Burgos","chant"]` {
		t.Fatalf("unexpected tags: %s", stored.Tags)
	}
}

func TestVideo_UploadRejectsWrongType(t *testing.T) {
	svc := videosvc.NewVideoService(newStubVideoRepo(), stubProfileRepo{}, newMemoryStorage(), 50)

	file := makeFileHeader(t, "clip.mp4", "video/mp4", 1024)
	_, err := svc.Upload(context.Background(), uuid.New(), file, "audio", "")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestVideo_UploadRejectsUnsupportedFormat(t *testing.T) {
	svc := videosvc.NewVideoService(newStubVideoRepo(), stubProfileRepo{}, newMemoryStorage(), 50)

	file := makeFileHeader(t, "clip.avi", "video/x-msvideo", 1024)
	_, err := svc.Upload(context.Background(), uuid.New(), file, "video", "")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", status)
	}
}

func TestVideo_UploadRejectsOversizedFile(t *testing.T) {
	// 1 MB limit; upload 2 MB.
	svc := videosvc.NewVideoService(newStubVideoRepo(), stubProfileRepo{}, newMemoryStorage(), 1)

	file := makeFileHeader(t, "clip.mp4", "video/mp4", 2*1024*1024)
	_, err := svc.Upload(context.Background(), uuid.New(), file, "video", "")
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if status := appErrorStatus(t, err); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", status)
	}
}

func TestVideo_LikeIsIdempotentAndCounted(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	video := &videomodels.Video{ID: uuid.New(), UserID: uuid.New(), Status: videomodels.StatusPublished, Tags: "[]"}
	videoRepo.videos[video.ID] = video
	userID := uuid.New()

	if err := svc.Like(context.Background(), userID, video.ID); err != nil {
		t.Fatalf("Like failed: %v", err)
	}
	if err := svc.Like(context.Background(), userID, video.ID); err != nil {
		t.Fatalf("second Like failed (should be idempotent): %v", err)
	}
	if video.LikesCount != 1 {
		t.Fatalf("expected likes_count 1, got %d", video.LikesCount)
	}

	if err := svc.Unlike(context.Background(), userID, video.ID); err != nil {
		t.Fatalf("Unlike failed: %v", err)
	}
	if video.LikesCount != 0 {
		t.Fatalf("expected likes_count 0, got %d", video.LikesCount)
	}
}

func TestVideo_LikeUnknownVideoNotFound(t *testing.T) {
	svc := videosvc.NewVideoService(newStubVideoRepo(), stubProfileRepo{}, newMemoryStorage(), 50)

	err := svc.Like(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestVideo_FeedMarksLikedVideos(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	video := &videomodels.Video{ID: uuid.New(), UserID: uuid.New(), Status: videomodels.StatusPublished, Tags: "[]"}
	videoRepo.videos[video.ID] = video
	userID := uuid.New()

	if err := svc.Like(context.Background(), userID, video.ID); err != nil {
		t.Fatalf("Like failed: %v", err)
	}

	feed, err := svc.Feed(context.Background(), userID, videodto.VideoListFilters{Limit: 20})
	if err != nil {
		t.Fatalf("Feed failed: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(feed.Items))
	}
	if !feed.Items[0].IsLiked {
		t.Fatal("expected item to be marked liked")
	}
	if feed.Items[0].LikesCount != 1 {
		t.Fatalf("expected likes_count 1, got %d", feed.Items[0].LikesCount)
	}
}

func TestVideo_MarkSeenIsIdempotentAndCounted(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	video := &videomodels.Video{ID: uuid.New(), UserID: uuid.New(), Status: videomodels.StatusPublished, Tags: "[]"}
	videoRepo.videos[video.ID] = video
	userID := uuid.New()

	resp, err := svc.MarkSeen(context.Background(), userID, video.ID)
	if err != nil {
		t.Fatalf("MarkSeen failed: %v", err)
	}
	if !resp.FirstSeen || !resp.IsSeen || resp.ViewsCount != 1 {
		t.Fatalf("unexpected first MarkSeen response: %+v", resp)
	}
	resp, err = svc.MarkSeen(context.Background(), userID, video.ID)
	if err != nil {
		t.Fatalf("second MarkSeen failed (should be idempotent): %v", err)
	}
	if resp.FirstSeen || !resp.IsSeen || resp.ViewsCount != 1 {
		t.Fatalf("unexpected second MarkSeen response: %+v", resp)
	}
	if video.ViewsCount != 1 {
		t.Fatalf("expected views_count 1, got %d", video.ViewsCount)
	}
}

func TestVideo_MarkSeenCountsEachUserOnce(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	video := &videomodels.Video{ID: uuid.New(), UserID: uuid.New(), Status: videomodels.StatusPublished, Tags: "[]"}
	videoRepo.videos[video.ID] = video
	userA := uuid.New()
	userB := uuid.New()

	if _, err := svc.MarkSeen(context.Background(), userA, video.ID); err != nil {
		t.Fatalf("MarkSeen userA failed: %v", err)
	}
	if _, err := svc.MarkSeen(context.Background(), userB, video.ID); err != nil {
		t.Fatalf("MarkSeen userB failed: %v", err)
	}
	if video.ViewsCount != 2 {
		t.Fatalf("expected views_count 2, got %d", video.ViewsCount)
	}
}

func TestVideo_MarkSeenUnknownVideoNotFound(t *testing.T) {
	svc := videosvc.NewVideoService(newStubVideoRepo(), stubProfileRepo{}, newMemoryStorage(), 50)

	_, err := svc.MarkSeen(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestVideo_FeedMarksSeenVideos(t *testing.T) {
	videoRepo := newStubVideoRepo()
	svc := videosvc.NewVideoService(videoRepo, stubProfileRepo{}, newMemoryStorage(), 50)

	video := &videomodels.Video{ID: uuid.New(), UserID: uuid.New(), Status: videomodels.StatusPublished, Tags: "[]"}
	videoRepo.videos[video.ID] = video
	userID := uuid.New()

	if _, err := svc.MarkSeen(context.Background(), userID, video.ID); err != nil {
		t.Fatalf("MarkSeen failed: %v", err)
	}

	feed, err := svc.Feed(context.Background(), userID, videodto.VideoListFilters{Limit: 20})
	if err != nil {
		t.Fatalf("Feed failed: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(feed.Items))
	}
	if !feed.Items[0].IsSeen {
		t.Fatal("expected item to be marked seen")
	}
	if feed.Items[0].ViewsCount != 1 {
		t.Fatalf("expected views_count 1, got %d", feed.Items[0].ViewsCount)
	}
}
