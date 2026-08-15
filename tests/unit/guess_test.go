package unit

import (
	"context"
	"testing"
	"time"

	clubmodels "clap/internal/modules/club/models"
	guessdto "clap/internal/modules/guess/dto"
	guessmodels "clap/internal/modules/guess/models"
	guesssvc "clap/internal/modules/guess/service"
	leaguemodels "clap/internal/modules/league/models"
	matchmodels "clap/internal/modules/match/models"
	settingsmodels "clap/internal/modules/settings/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGuessRepo struct {
	quizzes []guessmodels.Quiz
	answers map[string]guessmodels.QuizAnswer
}

func newStubGuessRepo() *stubGuessRepo {
	return &stubGuessRepo{answers: map[string]guessmodels.QuizAnswer{}}
}

func answerKey(quizID, userID uuid.UUID) string {
	return quizID.String() + ":" + userID.String()
}

func (r *stubGuessRepo) ListByMatchID(_ context.Context, matchID uuid.UUID) ([]guessmodels.Quiz, error) {
	var out []guessmodels.Quiz
	for _, quiz := range r.quizzes {
		if quiz.MatchID == matchID && quiz.IsActive {
			cp := quiz
			cp.Options = append([]guessmodels.QuizOption(nil), quiz.Options...)
			out = append(out, cp)
		}
	}
	return out, nil
}

func (r *stubGuessRepo) FindByID(_ context.Context, quizID uuid.UUID) (*guessmodels.Quiz, error) {
	for _, quiz := range r.quizzes {
		if quiz.ID == quizID && quiz.IsActive {
			cp := quiz
			cp.Options = append([]guessmodels.QuizOption(nil), quiz.Options...)
			return &cp, nil
		}
	}
	return nil, sharederrors.NewNotFound("Quiz not found", nil)
}

func (r *stubGuessRepo) CreateWithOptions(_ context.Context, quiz *guessmodels.Quiz, options []guessmodels.QuizOption) error {
	if quiz.ID == uuid.Nil {
		quiz.ID = uuid.New()
	}
	quiz.IsActive = true
	for i := range options {
		if options[i].ID == uuid.Nil {
			options[i].ID = uuid.New()
		}
		options[i].QuizID = quiz.ID
	}
	quiz.Options = append([]guessmodels.QuizOption(nil), options...)
	r.quizzes = append(r.quizzes, *quiz)
	return nil
}

func (r *stubGuessRepo) AnsweredQuizIDs(_ context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	for _, quizID := range quizIDs {
		if _, ok := r.answers[answerKey(quizID, userID)]; ok {
			out[quizID] = true
		}
	}
	return out, nil
}

func (r *stubGuessRepo) FindAnswer(_ context.Context, quizID, userID uuid.UUID) (*guessmodels.QuizAnswer, error) {
	answer, ok := r.answers[answerKey(quizID, userID)]
	if !ok {
		return nil, nil
	}
	cp := answer
	return &cp, nil
}

func (r *stubGuessRepo) SubmitAnswer(_ context.Context, answer *guessmodels.QuizAnswer) (bool, error) {
	key := answerKey(answer.QuizID, answer.UserID)
	if _, ok := r.answers[key]; ok {
		return false, nil
	}
	if answer.ID == uuid.Nil {
		answer.ID = uuid.New()
	}
	r.answers[key] = *answer
	return true, nil
}

type stubMatchFinder struct {
	match *matchmodels.Match
}

func (r stubMatchFinder) FindByID(_ context.Context, id uuid.UUID) (*matchmodels.Match, error) {
	if r.match == nil || r.match.ID != id {
		return nil, sharederrors.NewNotFound("Match not found", nil)
	}
	cp := *r.match
	return &cp, nil
}

func (r stubMatchFinder) FindCurrentForClub(_ context.Context, clubID uuid.UUID) (*matchmodels.Match, error) {
	if r.match == nil {
		return nil, nil
	}
	if r.match.HomeClubID != clubID && r.match.AwayClubID != clubID {
		return nil, nil
	}
	cp := *r.match
	return &cp, nil
}

type stubLineupLister struct {
	players []matchmodels.MatchLineupPlayer
}

func (r stubLineupLister) ListLineup(context.Context, uuid.UUID) ([]matchmodels.MatchLineupPlayer, error) {
	return append([]matchmodels.MatchLineupPlayer(nil), r.players...), nil
}

type stubSettingsGetter struct {
	clubID *uuid.UUID
}

func (r stubSettingsGetter) Get(context.Context) (*settingsmodels.AppSettings, error) {
	return &settingsmodels.AppSettings{ID: 1, FeaturedClubID: r.clubID}, nil
}

func sampleGuessMatch(clubID uuid.UUID, status string, kickoff time.Time) *matchmodels.Match {
	matchID := uuid.New()
	awayID := uuid.New()
	return &matchmodels.Match{
		ID:            matchID,
		HomeClubID:    clubID,
		AwayClubID:    awayID,
		MatchDateTime: kickoff,
		Status:        status,
		HomeClub:      clubmodels.Club{ID: clubID, Name: "FC Barcelona", LogoURL: "https://cdn.example/barca.png"},
		AwayClub:      clubmodels.Club{ID: awayID, Name: "SP Burgos", LogoURL: "https://cdn.example/burgos.png"},
		League:        leaguemodels.League{LogoURL: "https://cdn.example/uefa.png"},
	}
}

func TestGuessService_MatchOverviewCreatesResultQuiz(t *testing.T) {
	userID := uuid.New()
	clubID := uuid.New()
	match := sampleGuessMatch(clubID, "scheduled", time.Now().UTC().Add(2*time.Hour))
	repo := newStubGuessRepo()
	svc := guesssvc.NewGuessService(repo, stubMatchFinder{match: match}, stubLineupLister{}, stubSettingsGetter{clubID: &clubID})

	resp, err := svc.MatchOverview(context.Background(), userID, "current")
	require.NoError(t, err)
	assert.Equal(t, match.ID, resp.Match.ID)
	assert.Equal(t, "FC Barcelona", resp.Match.HomeName)
	assert.Equal(t, "SP Burgos", resp.Match.AwayName)
	assert.Equal(t, 100, resp.ParticipationPoints)
	assert.True(t, resp.IsActive)
	require.Len(t, resp.Quizzes, 1)
	assert.Equal(t, "Result of the game", resp.Quizzes[0].Title)
	assert.Equal(t, 600, resp.Quizzes[0].Points)
	assert.False(t, resp.Quizzes[0].IsDone)
}

func TestGuessService_MatchOverviewInactiveAfterKickoff(t *testing.T) {
	userID := uuid.New()
	clubID := uuid.New()
	match := sampleGuessMatch(clubID, "live", time.Now().UTC().Add(-10*time.Minute))
	repo := newStubGuessRepo()
	svc := guesssvc.NewGuessService(repo, stubMatchFinder{match: match}, stubLineupLister{}, stubSettingsGetter{clubID: &clubID})

	resp, err := svc.MatchOverview(context.Background(), userID, "current")
	require.NoError(t, err)
	assert.False(t, resp.IsActive)
}

func TestGuessService_AnswerAwardsParticipationPoints(t *testing.T) {
	userID := uuid.New()
	clubID := uuid.New()
	match := sampleGuessMatch(clubID, "scheduled", time.Now().UTC().Add(2*time.Hour))
	repo := newStubGuessRepo()
	svc := guesssvc.NewGuessService(repo, stubMatchFinder{match: match}, stubLineupLister{}, stubSettingsGetter{clubID: &clubID})

	overview, err := svc.MatchOverview(context.Background(), userID, match.ID.String())
	require.NoError(t, err)
	require.Len(t, overview.Quizzes, 1)

	resp, err := svc.Answer(context.Background(), userID, overview.Quizzes[0].ID, &guessdto.AnswerQuizRequest{Choice: "home"})
	require.NoError(t, err)
	assert.Equal(t, "submitted", resp.Status)
	assert.Equal(t, 100, resp.PointsEarned)

	_, err = svc.Answer(context.Background(), userID, overview.Quizzes[0].ID, &guessdto.AnswerQuizRequest{Choice: "away"})
	require.Error(t, err)
	var appErr *sharederrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
}

func TestGuessService_AnswerRejectedAfterKickoff(t *testing.T) {
	userID := uuid.New()
	clubID := uuid.New()
	match := sampleGuessMatch(clubID, "live", time.Now().UTC().Add(-10*time.Minute))
	repo := newStubGuessRepo()
	home := guessmodels.QuizOption{ID: uuid.New(), Label: "FC Barcelona wins", Value: "home"}
	quiz := guessmodels.Quiz{
		ID:       uuid.New(),
		MatchID:  match.ID,
		Title:    "Result of the game",
		QuizType: guessmodels.QuizTypeResult,
		Points:   600,
		IsActive: true,
		Options:  []guessmodels.QuizOption{home},
	}
	repo.quizzes = []guessmodels.Quiz{quiz}
	svc := guesssvc.NewGuessService(repo, stubMatchFinder{match: match}, stubLineupLister{}, stubSettingsGetter{clubID: &clubID})

	_, err := svc.Answer(context.Background(), userID, quiz.ID, &guessdto.AnswerQuizRequest{Choice: "home"})
	require.Error(t, err)
	var appErr *sharederrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 422, appErr.StatusCode)
}

func TestGuessService_QuizDetailIncludesOptions(t *testing.T) {
	userID := uuid.New()
	clubID := uuid.New()
	match := sampleGuessMatch(clubID, "scheduled", time.Now().UTC().Add(3*time.Hour))
	repo := newStubGuessRepo()
	svc := guesssvc.NewGuessService(repo, stubMatchFinder{match: match}, stubLineupLister{}, stubSettingsGetter{clubID: &clubID})

	overview, err := svc.MatchOverview(context.Background(), userID, match.ID.String())
	require.NoError(t, err)

	detail, err := svc.QuizDetail(context.Background(), userID, overview.Quizzes[0].ID)
	require.NoError(t, err)
	assert.Equal(t, "What will be the result of this game?", detail.Question)
	require.Len(t, detail.Options, 3)
	assert.Equal(t, "home", detail.Options[0].Value)
	assert.Equal(t, "away", detail.Options[1].Value)
	assert.Equal(t, "draw", detail.Options[2].Value)
	assert.False(t, detail.IsDone)
}
