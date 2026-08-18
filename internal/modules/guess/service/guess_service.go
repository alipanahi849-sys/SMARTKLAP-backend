package service

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/guess/dto"
	"clap/internal/modules/guess/models"
	guessrepo "clap/internal/modules/guess/repository"
	matchmodels "clap/internal/modules/match/models"
	settingsmodels "clap/internal/modules/settings/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
)

type matchFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*matchmodels.Match, error)
	FindCurrentForClub(ctx context.Context, clubID uuid.UUID) (*matchmodels.Match, error)
}

type lineupLister interface {
	ListLineup(ctx context.Context, matchID uuid.UUID) ([]matchmodels.MatchLineupPlayer, error)
}

type settingsGetter interface {
	Get(ctx context.Context) (*settingsmodels.AppSettings, error)
}

type GuessService interface {
	MatchOverview(ctx context.Context, userID uuid.UUID, matchID string) (*dto.MatchOverviewResponse, error)
	QuizDetail(ctx context.Context, userID, quizID uuid.UUID) (*dto.QuizDetailResponse, error)
	Answer(ctx context.Context, userID, quizID uuid.UUID, req *dto.AnswerQuizRequest) (*dto.AnswerQuizResponse, error)
}

type guessService struct {
	guess    guessrepo.GuessRepository
	matches  matchFinder
	details  lineupLister
	settings settingsGetter
	now      func() time.Time
}

func NewGuessService(
	guess guessrepo.GuessRepository,
	matches matchFinder,
	details lineupLister,
	settings settingsGetter,
) GuessService {
	return &guessService{
		guess:    guess,
		matches:  matches,
		details:  details,
		settings: settings,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *guessService) MatchOverview(ctx context.Context, userID uuid.UUID, matchID string) (*dto.MatchOverviewResponse, error) {
	match, err := s.resolveMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureDefaultQuizzes(ctx, match); err != nil {
		return nil, err
	}

	quizzes, err := s.guess.ListByMatchID(ctx, match.ID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(quizzes))
	for _, quiz := range quizzes {
		ids = append(ids, quiz.ID)
	}
	answers, err := s.guess.FindAnswers(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	summaries := make([]dto.GuessQuizSummary, 0, len(quizzes))
	for _, quiz := range quizzes {
		summary := dto.GuessQuizSummary{
			ID:     quiz.ID,
			Title:  quiz.Title,
			Points: quiz.Points,
		}
		if answer, ok := answers[quiz.ID]; ok {
			summary.IsDone = true
			summary.SelectedChoice = answer.Choice
			summary.SelectedLabel = labelForChoice(quiz.Options, answer.Choice)
		}
		summaries = append(summaries, summary)
	}

	return &dto.MatchOverviewResponse{
		Match:               toMatchInfo(match),
		IsActive:            guessingOpen(match, s.now()),
		ParticipationPoints: models.ParticipationPoints,
		Quizzes:             summaries,
	}, nil
}

func (s *guessService) QuizDetail(ctx context.Context, userID, quizID uuid.UUID) (*dto.QuizDetailResponse, error) {
	quiz, err := s.guess.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	match, err := s.matches.FindByID(ctx, quiz.MatchID)
	if err != nil {
		return nil, err
	}

	answer, err := s.guess.FindAnswer(ctx, quiz.ID, userID)
	if err != nil {
		return nil, err
	}

	options := make([]dto.GuessQuizOption, 0, len(quiz.Options))
	for _, option := range quiz.Options {
		options = append(options, dto.GuessQuizOption{
			ID:    option.ID,
			Label: option.Label,
			Value: option.Value,
		})
	}

	resp := &dto.QuizDetailResponse{
		ID:       quiz.ID,
		MatchID:  quiz.MatchID,
		Title:    quiz.Title,
		Question: models.QuestionForType(quiz.QuizType, quiz.Title),
		QuizType: quiz.QuizType,
		Points:   quiz.Points,
		IsDone:   answer != nil,
		IsOpen:   guessingOpen(match, s.now()),
		Options:  options,
	}
	if answer != nil {
		resp.SelectedChoice = answer.Choice
		resp.SelectedLabel = labelForChoice(quiz.Options, answer.Choice)
	}
	return resp, nil
}

func (s *guessService) Answer(ctx context.Context, userID, quizID uuid.UUID, req *dto.AnswerQuizRequest) (*dto.AnswerQuizResponse, error) {
	choice := strings.TrimSpace(req.Choice)
	if choice == "" {
		return nil, errors.NewBadRequest("choice is required", nil)
	}

	quiz, err := s.guess.FindByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	match, err := s.matches.FindByID(ctx, quiz.MatchID)
	if err != nil {
		return nil, err
	}
	if !guessingOpen(match, s.now()) {
		return nil, errors.NewUnprocessable("Guessing is closed for this match", nil)
	}

	canonical, ok := resolveChoice(quiz.Options, choice)
	if !ok {
		return nil, errors.NewBadRequest("Invalid choice", nil)
	}

	answer := &models.QuizAnswer{
		QuizID:       quiz.ID,
		UserID:       userID,
		Choice:       canonical,
		PointsEarned: models.ParticipationPoints,
	}
	created, err := s.guess.SubmitAnswer(ctx, answer)
	if err != nil {
		return nil, err
	}
	if !created {
		return &dto.AnswerQuizResponse{
			Status:       "updated",
			PointsEarned: 0,
		}, nil
	}

	return &dto.AnswerQuizResponse{
		Status:       "submitted",
		PointsEarned: models.ParticipationPoints,
	}, nil
}

func (s *guessService) resolveMatch(ctx context.Context, matchID string) (*matchmodels.Match, error) {
	raw := strings.ToLower(strings.TrimSpace(matchID))
	if raw == "" || raw == "current" {
		return s.currentMatch(ctx)
	}
	id, err := uuid.Parse(strings.TrimSpace(matchID))
	if err != nil {
		return nil, errors.NewBadRequest("Invalid match ID", nil)
	}
	return s.matches.FindByID(ctx, id)
}

func (s *guessService) currentMatch(ctx context.Context) (*matchmodels.Match, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if settings.FeaturedClubID == nil {
		return nil, errors.NewNotFound("Match not found", nil)
	}
	match, err := s.matches.FindCurrentForClub(ctx, *settings.FeaturedClubID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, errors.NewNotFound("Match not found", nil)
	}
	return match, nil
}

func (s *guessService) ensureDefaultQuizzes(ctx context.Context, match *matchmodels.Match) error {
	existing, err := s.guess.ListByMatchID(ctx, match.ID)
	if err != nil {
		return err
	}

	hasResult := false
	hasPlayer := false
	for _, quiz := range existing {
		switch quiz.QuizType {
		case models.QuizTypeResult:
			hasResult = true
		case models.QuizTypePlayer:
			hasPlayer = true
		}
	}

	if !hasResult {
		quiz := &models.Quiz{
			MatchID:  match.ID,
			Title:    models.ResultQuizTitle,
			QuizType: models.QuizTypeResult,
			Points:   models.ResultQuizPoints,
			IsActive: true,
		}
		options := []models.QuizOption{
			{Label: clubDisplayName(match.HomeClub.Name, match.HomeClub.ShortName) + " wins", Value: models.OptionHome, SortOrder: 0},
			{Label: clubDisplayName(match.AwayClub.Name, match.AwayClub.ShortName) + " wins", Value: models.OptionAway, SortOrder: 1},
			{Label: "Draw their match", Value: models.OptionDraw, SortOrder: 2},
		}
		if err := s.guess.CreateWithOptions(ctx, quiz, options); err != nil {
			return err
		}
	}

	if hasPlayer {
		return nil
	}

	lineup, err := s.details.ListLineup(ctx, match.ID)
	if err != nil {
		return err
	}
	options := playerOptions(lineup)
	if len(options) == 0 {
		return nil
	}
	quiz := &models.Quiz{
		MatchID:  match.ID,
		Title:    models.PlayerQuizTitle,
		QuizType: models.QuizTypePlayer,
		Points:   models.PlayerQuizPoints,
		IsActive: true,
	}
	return s.guess.CreateWithOptions(ctx, quiz, options)
}

func playerOptions(lineup []matchmodels.MatchLineupPlayer) []models.QuizOption {
	starters := make([]matchmodels.MatchLineupPlayer, 0, len(lineup))
	for _, player := range lineup {
		if player.IsStarter {
			starters = append(starters, player)
		}
	}
	source := starters
	if len(source) == 0 {
		source = lineup
	}

	seen := make(map[string]struct{})
	options := make([]models.QuizOption, 0, len(source))
	for _, player := range source {
		label := strings.TrimSpace(player.Name)
		if label == "" {
			continue
		}
		value := player.ID.String()
		if player.PlayerID != nil && *player.PlayerID != uuid.Nil {
			value = player.PlayerID.String()
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		options = append(options, models.QuizOption{
			Label:     label,
			Value:     value,
			SortOrder: len(options),
		})
	}
	return options
}

func labelForChoice(options []models.QuizOption, choice string) string {
	for _, option := range options {
		if option.Value == choice || option.ID.String() == choice {
			return option.Label
		}
	}
	return choice
}

func resolveChoice(options []models.QuizOption, choice string) (string, bool) {
	for _, option := range options {
		if option.Value == choice || option.ID.String() == choice {
			return option.Value, true
		}
	}
	return "", false
}

func guessingOpen(match *matchmodels.Match, now time.Time) bool {
	if match == nil || match.Status != "scheduled" {
		return false
	}
	return match.MatchDateTime.After(now)
}

func toMatchInfo(match *matchmodels.Match) dto.GuessMatchInfo {
	kickoff := match.MatchDateTime.UTC()
	logo := match.CompetitionLogoURL
	if logo == "" {
		logo = match.League.LogoURL
	}
	return dto.GuessMatchInfo{
		ID:                 match.ID,
		HomeName:           clubDisplayName(match.HomeClub.Name, match.HomeClub.ShortName),
		HomeRole:           "Home",
		HomeLogoURL:        match.HomeClub.LogoURL,
		AwayName:           clubDisplayName(match.AwayClub.Name, match.AwayClub.ShortName),
		AwayRole:           "Away",
		AwayLogoURL:        match.AwayClub.LogoURL,
		CompetitionLogoURL: logo,
		Date:               kickoff.Format("2006-01-02"),
		Time:               kickoff.Format("15:04"),
		KickoffAt:          kickoff.Format(time.RFC3339),
		Status:             match.Status,
	}
}

func clubDisplayName(name, shortName string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(shortName)
}
