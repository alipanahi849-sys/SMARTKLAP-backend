package service

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/guess/dto"
	"clap/internal/modules/guess/models"
	"clap/internal/modules/guess/repository"
	matchmodels "clap/internal/modules/match/models"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// ParticipationPoints is awarded for submitting any quiz answer (contract
// §5.1 shows a fixed participation reward; final correctness points are
// settled after the match).
const ParticipationPoints = 100

// GuessService implements the mobile Guess screens (contract §5).
type GuessService interface {
	// MatchOverview accepts a concrete match ID or nil for the "current" match.
	MatchOverview(ctx context.Context, userID uuid.UUID, matchID *uuid.UUID) (*dto.GuessMatchResponse, error)
	QuizDetail(ctx context.Context, quizID uuid.UUID) (*dto.QuizDetailResponse, error)
	Answer(ctx context.Context, userID, quizID uuid.UUID, choice string) (*dto.AnswerQuizResponse, error)
}

type guessService struct {
	quizRepo  repository.QuizRepository
	matchRepo matchrepo.MatchRepository
}

func NewGuessService(quizRepo repository.QuizRepository, matchRepo matchrepo.MatchRepository) GuessService {
	return &guessService{quizRepo: quizRepo, matchRepo: matchRepo}
}

func (s *guessService) MatchOverview(ctx context.Context, userID uuid.UUID, matchID *uuid.UUID) (*dto.GuessMatchResponse, error) {
	match, err := s.resolveMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	quizzes, err := s.quizRepo.FindByMatch(ctx, match.ID)
	if err != nil {
		return nil, err
	}

	quizIDs := make([]uuid.UUID, len(quizzes))
	for i, q := range quizzes {
		quizIDs[i] = q.ID
	}
	answered, err := s.quizRepo.AnsweredQuizIDs(ctx, userID, quizIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.GuessQuizItem, len(quizzes))
	for i, q := range quizzes {
		items[i] = dto.GuessQuizItem{
			ID:     q.ID,
			Title:  q.Title,
			Points: q.Points,
			IsDone: answered[q.ID],
		}
	}

	return &dto.GuessMatchResponse{
		Match: dto.GuessMatchInfo{
			HomeName:    match.HomeClub.Name,
			HomeRole:    "Home",
			HomeLogoURL: match.HomeClub.LogoURL,
			AwayName:    match.AwayClub.Name,
			AwayRole:    "Away",
			AwayLogoURL: match.AwayClub.LogoURL,
			// Leagues have no logo asset in the current schema; empty until added.
			CompetitionLogoURL: "",
			Date:               match.MatchDateTime.Format("2006-01-02"),
			Time:               match.MatchDateTime.Format("15:04"),
		},
		ParticipationPoints: ParticipationPoints,
		Quizzes:             items,
	}, nil
}

func (s *guessService) QuizDetail(ctx context.Context, quizID uuid.UUID) (*dto.QuizDetailResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	options := make([]dto.QuizOptionItem, len(quiz.Options))
	for i, o := range quiz.Options {
		options[i] = dto.QuizOptionItem{ID: o.ID, Label: o.Label}
	}

	return &dto.QuizDetailResponse{
		ID:      quiz.ID,
		Title:   quiz.Title,
		Type:    quiz.QuizType,
		Options: options,
	}, nil
}

func (s *guessService) Answer(ctx context.Context, userID, quizID uuid.UUID, choice string) (*dto.AnswerQuizResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	match, err := s.matchRepo.FindByID(ctx, quiz.MatchID)
	if err != nil {
		return nil, err
	}
	// Guesses close at kickoff (contract §5.2: 422 when started/finished).
	if match.Status != "scheduled" || !time.Now().UTC().Before(match.MatchDateTime.UTC()) {
		return nil, errors.NewUnprocessable("Match has already started or finished", nil)
	}

	choice = strings.TrimSpace(choice)
	if !s.isValidChoice(quiz, choice) {
		return nil, errors.NewBadRequest("Invalid choice for this quiz", nil)
	}

	if err := s.quizRepo.Answer(ctx, quizID, userID, choice, ParticipationPoints); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("quiz_id", quizID.String()).
		Str("match_id", quiz.MatchID.String()).
		Int("points_earned", ParticipationPoints).
		Msg("quiz_answered")

	return &dto.AnswerQuizResponse{
		Status:       "submitted",
		PointsEarned: ParticipationPoints,
	}, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

// isValidChoice accepts an option value or an option ID. Player quizzes with
// no predefined options accept any non-empty player identifier.
func (s *guessService) isValidChoice(quiz *models.Quiz, choice string) bool {
	if choice == "" {
		return false
	}
	if len(quiz.Options) == 0 {
		return quiz.QuizType == models.QuizTypePlayer
	}
	for _, o := range quiz.Options {
		if strings.EqualFold(o.Value, choice) || o.ID.String() == choice {
			return true
		}
	}
	return false
}

func (s *guessService) resolveMatch(ctx context.Context, matchID *uuid.UUID) (*matchmodels.Match, error) {
	if matchID != nil {
		return s.matchRepo.FindByID(ctx, *matchID)
	}

	// "current": the live match, or the next upcoming one.
	if live, err := s.matchRepo.FindLive(ctx); err == nil && len(live) > 0 {
		return &live[0], nil
	}
	if upcoming, _, err := s.matchRepo.FindUpcoming(ctx, 1, 1); err == nil && len(upcoming) > 0 {
		return &upcoming[0], nil
	}
	return nil, errors.NewNotFound("No active match found", nil)
}
