package unit

import (
	"context"
	"net/http"
	"testing"
	"time"

	guessmodels "clap/internal/modules/guess/models"
	guesssvc "clap/internal/modules/guess/service"
	matchmodels "clap/internal/modules/match/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

type stubQuizRepo struct {
	quizzes  map[uuid.UUID]*guessmodels.Quiz
	answered map[uuid.UUID]map[uuid.UUID]bool // quizID → userID → answered
}

func newStubQuizRepo() *stubQuizRepo {
	return &stubQuizRepo{
		quizzes:  map[uuid.UUID]*guessmodels.Quiz{},
		answered: map[uuid.UUID]map[uuid.UUID]bool{},
	}
}

func (r *stubQuizRepo) FindByID(_ context.Context, id uuid.UUID) (*guessmodels.Quiz, error) {
	if q, ok := r.quizzes[id]; ok {
		return q, nil
	}
	return nil, sharederrors.NewNotFound("Quiz not found", nil)
}

func (r *stubQuizRepo) FindByMatch(_ context.Context, matchID uuid.UUID) ([]guessmodels.Quiz, error) {
	var result []guessmodels.Quiz
	for _, q := range r.quizzes {
		if q.MatchID == matchID {
			result = append(result, *q)
		}
	}
	return result, nil
}

func (r *stubQuizRepo) AnsweredQuizIDs(_ context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	for _, id := range quizIDs {
		if users, ok := r.answered[id]; ok && users[userID] {
			result[id] = true
		}
	}
	return result, nil
}

func (r *stubQuizRepo) Answer(_ context.Context, quizID, userID uuid.UUID, _ string, _ int) error {
	if users, ok := r.answered[quizID]; ok && users[userID] {
		return sharederrors.NewConflict("Quiz already answered", nil)
	}
	if r.answered[quizID] == nil {
		r.answered[quizID] = map[uuid.UUID]bool{}
	}
	r.answered[quizID][userID] = true
	return nil
}

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

func (r *stubMatchRepo) Update(_ context.Context, m *matchmodels.Match) error {
	r.matches[m.ID] = m
	return nil
}

func (r *stubMatchRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.matches, id)
	return nil
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

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

func newResultQuiz(repo *stubQuizRepo, matchID uuid.UUID) *guessmodels.Quiz {
	q := &guessmodels.Quiz{
		ID:       uuid.New(),
		MatchID:  matchID,
		Title:    "Guess the game result",
		QuizType: guessmodels.QuizTypeResult,
		Points:   500,
		IsActive: true,
		Options: []guessmodels.QuizOption{
			{ID: uuid.New(), Label: "Home wins", Value: "home"},
			{ID: uuid.New(), Label: "Draw", Value: "draw"},
			{ID: uuid.New(), Label: "Away wins", Value: "away"},
		},
	}
	repo.quizzes[q.ID] = q
	return q
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestGuess_AnswerAwardsParticipationPoints(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	quiz := newResultQuiz(quizRepo, match.ID)
	userID := uuid.New()

	resp, err := svc.Answer(context.Background(), userID, quiz.ID, "home")
	if err != nil {
		t.Fatalf("Answer failed: %v", err)
	}
	if resp.PointsEarned != guesssvc.ParticipationPoints {
		t.Fatalf("expected %d points, got %d", guesssvc.ParticipationPoints, resp.PointsEarned)
	}
}

func TestGuess_AnswerTwiceConflicts(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	quiz := newResultQuiz(quizRepo, match.ID)
	userID := uuid.New()

	if _, err := svc.Answer(context.Background(), userID, quiz.ID, "home"); err != nil {
		t.Fatalf("first answer failed: %v", err)
	}
	_, err := svc.Answer(context.Background(), userID, quiz.ID, "draw")
	if err == nil {
		t.Fatal("expected conflict for second answer")
	}
	if status := appErrorStatus(t, err); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
}

func TestGuess_AnswerAfterKickoffUnprocessable(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(-1*time.Hour)) // kicked off
	quiz := newResultQuiz(quizRepo, match.ID)

	_, err := svc.Answer(context.Background(), uuid.New(), quiz.ID, "home")
	if err == nil {
		t.Fatal("expected error after kickoff")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", status)
	}
}

func TestGuess_AnswerInvalidChoiceRejected(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	quiz := newResultQuiz(quizRepo, match.ID)

	_, err := svc.Answer(context.Background(), uuid.New(), quiz.ID, "not-an-option")
	if err == nil {
		t.Fatal("expected validation error for invalid choice")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestGuess_AnswerUnknownQuizNotFound(t *testing.T) {
	svc := guesssvc.NewGuessService(newStubQuizRepo(), newStubMatchRepo())

	_, err := svc.Answer(context.Background(), uuid.New(), uuid.New(), "home")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestGuess_MatchOverviewMarksAnswered(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	match := newScheduledMatch(matchRepo, time.Now().Add(2*time.Hour))
	quiz := newResultQuiz(quizRepo, match.ID)
	userID := uuid.New()

	if _, err := svc.Answer(context.Background(), userID, quiz.ID, "home"); err != nil {
		t.Fatalf("answer failed: %v", err)
	}

	overview, err := svc.MatchOverview(context.Background(), userID, &match.ID)
	if err != nil {
		t.Fatalf("MatchOverview failed: %v", err)
	}
	if len(overview.Quizzes) != 1 {
		t.Fatalf("expected 1 quiz, got %d", len(overview.Quizzes))
	}
	if !overview.Quizzes[0].IsDone {
		t.Fatal("expected quiz to be marked done")
	}
}

func TestGuess_MatchOverviewCurrentResolvesUpcoming(t *testing.T) {
	quizRepo := newStubQuizRepo()
	matchRepo := newStubMatchRepo()
	svc := guesssvc.NewGuessService(quizRepo, matchRepo)

	newScheduledMatch(matchRepo, time.Now().Add(3*time.Hour))

	overview, err := svc.MatchOverview(context.Background(), uuid.New(), nil)
	if err != nil {
		t.Fatalf("MatchOverview(current) failed: %v", err)
	}
	if overview.ParticipationPoints != guesssvc.ParticipationPoints {
		t.Fatalf("unexpected participation points: %d", overview.ParticipationPoints)
	}
}

func TestGuess_MatchOverviewNoMatchNotFound(t *testing.T) {
	svc := guesssvc.NewGuessService(newStubQuizRepo(), newStubMatchRepo())

	_, err := svc.MatchOverview(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Fatal("expected not-found when no match exists")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}
