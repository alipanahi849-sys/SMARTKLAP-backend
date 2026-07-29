package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	guesshandler "clap/internal/modules/guess/handler"
	guessmodels "clap/internal/modules/guess/models"
	guesssvc "clap/internal/modules/guess/service"
	matchmodels "clap/internal/modules/match/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/middleware"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// These tests exercise the full HTTP stack (router → auth middleware →
// handler → service) with in-memory repositories, so they run without a
// database.

// ─── in-memory repositories ───────────────────────────────────────────────────

type memQuizRepo struct {
	quizzes  map[uuid.UUID]*guessmodels.Quiz
	answered map[uuid.UUID]map[uuid.UUID]bool
}

func newMemQuizRepo() *memQuizRepo {
	return &memQuizRepo{
		quizzes:  map[uuid.UUID]*guessmodels.Quiz{},
		answered: map[uuid.UUID]map[uuid.UUID]bool{},
	}
}

func (r *memQuizRepo) FindByID(_ context.Context, id uuid.UUID) (*guessmodels.Quiz, error) {
	if q, ok := r.quizzes[id]; ok {
		return q, nil
	}
	return nil, sharederrors.NewNotFound("Quiz not found", nil)
}

func (r *memQuizRepo) FindByMatch(_ context.Context, matchID uuid.UUID) ([]guessmodels.Quiz, error) {
	var result []guessmodels.Quiz
	for _, q := range r.quizzes {
		if q.MatchID == matchID {
			result = append(result, *q)
		}
	}
	return result, nil
}

func (r *memQuizRepo) AnsweredQuizIDs(_ context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	for _, id := range quizIDs {
		if users, ok := r.answered[id]; ok && users[userID] {
			result[id] = true
		}
	}
	return result, nil
}

func (r *memQuizRepo) Answer(_ context.Context, quizID, userID uuid.UUID, _ string, _ int) error {
	if users, ok := r.answered[quizID]; ok && users[userID] {
		return sharederrors.NewConflict("Quiz already answered", nil)
	}
	if r.answered[quizID] == nil {
		r.answered[quizID] = map[uuid.UUID]bool{}
	}
	r.answered[quizID][userID] = true
	return nil
}

type memMatchRepo struct {
	matches map[uuid.UUID]*matchmodels.Match
}

func newMemMatchRepo() *memMatchRepo {
	return &memMatchRepo{matches: map[uuid.UUID]*matchmodels.Match{}}
}

func (r *memMatchRepo) Create(_ context.Context, m *matchmodels.Match) error {
	r.matches[m.ID] = m
	return nil
}

func (r *memMatchRepo) FindByID(_ context.Context, id uuid.UUID) (*matchmodels.Match, error) {
	if m, ok := r.matches[id]; ok {
		return m, nil
	}
	return nil, sharederrors.NewNotFound("Match not found", nil)
}

func (r *memMatchRepo) FindAll(context.Context, int, int, map[string]string, string, string) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *memMatchRepo) FindBySeason(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *memMatchRepo) FindByLeague(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *memMatchRepo) FindByClub(context.Context, uuid.UUID, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *memMatchRepo) FindUpcoming(context.Context, int, int) ([]matchmodels.Match, int64, error) {
	return nil, 0, nil
}
func (r *memMatchRepo) FindLive(context.Context) ([]matchmodels.Match, error) { return nil, nil }
func (r *memMatchRepo) Update(_ context.Context, m *matchmodels.Match) error {
	r.matches[m.ID] = m
	return nil
}
func (r *memMatchRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.matches, id)
	return nil
}

// ─── fixture ──────────────────────────────────────────────────────────────────

type guessAPIFixture struct {
	router    *gin.Engine
	quizRepo  *memQuizRepo
	matchRepo *memMatchRepo
	token     string
	userID    uuid.UUID
}

func newGuessAPIFixture(t *testing.T) *guessAPIFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	quizRepo := newMemQuizRepo()
	matchRepo := newMemMatchRepo()
	h := guesshandler.NewGuessHandler(guesssvc.NewGuessService(quizRepo, matchRepo))

	router := gin.New()
	v1 := router.Group("/api/v1")
	guess := v1.Group("/guess")
	guess.Use(middleware.Auth())
	{
		guess.GET("/matches/:match_id", h.MatchOverview)
		guess.GET("/quizzes/:quiz_id", h.QuizDetail)
		guess.POST("/quizzes/:quiz_id/answer", h.Answer)
	}

	userID := uuid.New()
	token, _, err := utils.GenerateAccessToken(userID, "fan@example.com", []string{"user"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return &guessAPIFixture{
		router:    router,
		quizRepo:  quizRepo,
		matchRepo: matchRepo,
		token:     token,
		userID:    userID,
	}
}

func (f *guessAPIFixture) seedQuiz(kickoffIn time.Duration) *guessmodels.Quiz {
	match := &matchmodels.Match{
		ID:            uuid.New(),
		Status:        "scheduled",
		MatchDateTime: time.Now().Add(kickoffIn),
		HomeClubID:    uuid.New(),
		AwayClubID:    uuid.New(),
	}
	f.matchRepo.matches[match.ID] = match

	quiz := &guessmodels.Quiz{
		ID:       uuid.New(),
		MatchID:  match.ID,
		Title:    "Guess the game result",
		QuizType: guessmodels.QuizTypeResult,
		Points:   500,
		IsActive: true,
		Options: []guessmodels.QuizOption{
			{ID: uuid.New(), Label: "Home wins", Value: "home"},
			{ID: uuid.New(), Label: "Draw", Value: "draw"},
		},
	}
	f.quizRepo.quizzes[quiz.ID] = quiz
	return quiz
}

func (f *guessAPIFixture) request(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestGuessAPI_RequiresAuthentication(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(2 * time.Hour)

	rec := f.request(t, http.MethodGet, "/api/v1/guess/quizzes/"+quiz.ID.String(), "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	rec = f.request(t, http.MethodGet, "/api/v1/guess/quizzes/"+quiz.ID.String(), "", "not-a-jwt")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", rec.Code)
	}
}

func TestGuessAPI_QuizDetailSuccess(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(2 * time.Hour)

	rec := f.request(t, http.MethodGet, "/api/v1/guess/quizzes/"+quiz.ID.String(), "", f.token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Title   string `json:"title"`
			Options []struct {
				Label string `json:"label"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if envelope.Data.Title != "Guess the game result" {
		t.Fatalf("unexpected title: %q", envelope.Data.Title)
	}
	if len(envelope.Data.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(envelope.Data.Options))
	}
}

func TestGuessAPI_QuizDetailInvalidUUID(t *testing.T) {
	f := newGuessAPIFixture(t)

	rec := f.request(t, http.MethodGet, "/api/v1/guess/quizzes/not-a-uuid", "", f.token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGuessAPI_QuizDetailNotFound(t *testing.T) {
	f := newGuessAPIFixture(t)

	rec := f.request(t, http.MethodGet, "/api/v1/guess/quizzes/"+uuid.NewString(), "", f.token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGuessAPI_AnswerFullFlow(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(2 * time.Hour)
	path := "/api/v1/guess/quizzes/" + quiz.ID.String() + "/answer"

	// Success.
	rec := f.request(t, http.MethodPost, path, `{"choice":"home"}`, f.token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Second answer conflicts.
	rec = f.request(t, http.MethodPost, path, `{"choice":"draw"}`, f.token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGuessAPI_AnswerValidation(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(2 * time.Hour)
	path := "/api/v1/guess/quizzes/" + quiz.ID.String() + "/answer"

	// Missing choice → binding failure.
	rec := f.request(t, http.MethodPost, path, `{}`, f.token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing choice, got %d", rec.Code)
	}

	// Choice that is not an option.
	rec = f.request(t, http.MethodPost, path, `{"choice":"nonsense"}`, f.token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid choice, got %d", rec.Code)
	}
}

func TestGuessAPI_AnswerAfterKickoff(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(-1 * time.Hour) // already started
	path := "/api/v1/guess/quizzes/" + quiz.ID.String() + "/answer"

	rec := f.request(t, http.MethodPost, path, `{"choice":"home"}`, f.token)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGuessAPI_MatchOverview(t *testing.T) {
	f := newGuessAPIFixture(t)
	quiz := f.seedQuiz(2 * time.Hour)

	rec := f.request(t, http.MethodGet, "/api/v1/guess/matches/"+quiz.MatchID.String(), "", f.token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Invalid UUID (and not the literal "current") → 400.
	rec = f.request(t, http.MethodGet, "/api/v1/guess/matches/bogus", "", f.token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
