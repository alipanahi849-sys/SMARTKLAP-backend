package unit

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"
	"time"

	chantmodels "clap/internal/modules/chant/models"
	chantdto "clap/internal/modules/chant/dto"
	chantrepo "clap/internal/modules/chant/repository"
	chantsvc "clap/internal/modules/chant/service"
	matchmodels "clap/internal/modules/match/models"
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

type stubChantRepo struct {
	chants      map[uuid.UUID]*chantmodels.Chant
	completions map[uuid.UUID]map[uuid.UUID]int // chantID → userID → points
	userPoints  map[uuid.UUID]int
}

var _ chantrepo.ChantRepository = (*stubChantRepo)(nil)

func newStubChantRepo() *stubChantRepo {
	return &stubChantRepo{
		chants:      map[uuid.UUID]*chantmodels.Chant{},
		completions: map[uuid.UUID]map[uuid.UUID]int{},
		userPoints:  map[uuid.UUID]int{},
	}
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

func (r *stubChantRepo) HasIncompleteAtOrBefore(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _ *chantrepo.ChantCursorAnchor) (bool, error) {
	return false, nil
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
		if users, ok := r.completions[id]; ok {
			if _, completed := users[userID]; completed {
				done[id] = true
			}
		}
	}
	return done, nil
}

func (r *stubChantRepo) TodayPoints(_ context.Context, userID uuid.UUID) (int, error) {
	total := 0
	for _, users := range r.completions {
		total += users[userID]
	}
	return total, nil
}

func (r *stubChantRepo) TodayCompletions(_ context.Context, _ uuid.UUID, _ int) ([]chantmodels.ChantCompletion, map[uuid.UUID]chantmodels.Chant, error) {
	return nil, map[uuid.UUID]chantmodels.Chant{}, nil
}

func (r *stubChantRepo) Complete(_ context.Context, chantID, userID uuid.UUID, points int) (int, bool, error) {
	if users, ok := r.completions[chantID]; ok {
		if _, completed := users[userID]; completed {
			return r.userPoints[userID], false, nil
		}
	}
	if r.completions[chantID] == nil {
		r.completions[chantID] = map[uuid.UUID]int{}
	}
	r.completions[chantID][userID] = points
	r.userPoints[userID] += points
	return r.userPoints[userID], true, nil
}

// ─── chant tests ──────────────────────────────────────────────────────────────

func TestChant_CompleteAwardsPointsOnce(t *testing.T) {
	chantRepo := newStubChantRepo()
	matchRepo := newStubMatchRepo()
	svc := chantsvc.NewChantService(chantRepo, matchRepo, nil, nil)

	chant := &chantmodels.Chant{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		Title:   "Chant number 1",
		Points:  100,
	}
	chantRepo.chants[chant.ID] = chant
	userID := uuid.New()

	resp, err := svc.Complete(context.Background(), userID, chant.ID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if !resp.IsDone || resp.PointsEarned != 100 || resp.TotalPoints != 100 {
		t.Fatalf("unexpected response: %+v", resp)
	}

	resp, err = svc.Complete(context.Background(), userID, chant.ID)
	if err != nil {
		t.Fatalf("second Complete failed: %v", err)
	}
	if !resp.IsDone || resp.PointsEarned != 0 || resp.TotalPoints != 100 {
		t.Fatalf("unexpected idempotent response: %+v", resp)
	}
}

func TestChant_CompleteUnknownChantNotFound(t *testing.T) {
	svc := chantsvc.NewChantService(newStubChantRepo(), newStubMatchRepo(), nil, nil)

	_, err := svc.Complete(context.Background(), uuid.New(), uuid.New())
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
	svc := chantsvc.NewChantService(chantRepo, matchRepo, nil, nil)

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
