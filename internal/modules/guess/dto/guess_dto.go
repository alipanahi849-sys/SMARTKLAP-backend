package dto

import "github.com/google/uuid"

type GuessMatchInfo struct {
	ID                 uuid.UUID `json:"id"`
	HomeName           string    `json:"home_name"`
	HomeRole           string    `json:"home_role"`
	HomeLogoURL        string    `json:"home_logo_url"`
	AwayName           string    `json:"away_name"`
	AwayRole           string    `json:"away_role"`
	AwayLogoURL        string    `json:"away_logo_url"`
	CompetitionLogoURL string    `json:"competition_logo_url,omitempty"`
	Date               string    `json:"date"`
	Time               string    `json:"time"`
	KickoffAt          string    `json:"kickoff_at"`
	Status             string    `json:"status"`
}

type GuessQuizSummary struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Points         int       `json:"points"`
	IsDone         bool      `json:"is_done"`
	SelectedChoice string    `json:"selected_choice,omitempty"`
	SelectedLabel  string    `json:"selected_label,omitempty"`
}

type MatchOverviewResponse struct {
	Match               GuessMatchInfo     `json:"match"`
	IsActive            bool               `json:"is_active"`
	ParticipationPoints int                `json:"participation_points"`
	Quizzes             []GuessQuizSummary `json:"quizzes"`
}

type GuessQuizOption struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
	Value string    `json:"value"`
}

type QuizDetailResponse struct {
	ID             uuid.UUID         `json:"id"`
	MatchID        uuid.UUID         `json:"match_id"`
	Title          string            `json:"title"`
	Question       string            `json:"question"`
	QuizType       string            `json:"quiz_type"`
	Points         int               `json:"points"`
	IsDone         bool              `json:"is_done"`
	IsOpen         bool              `json:"is_open"`
	SelectedChoice string            `json:"selected_choice,omitempty"`
	SelectedLabel  string            `json:"selected_label,omitempty"`
	Options        []GuessQuizOption `json:"options"`
}

type AnswerQuizRequest struct {
	Choice string `json:"choice" binding:"required"`
}

type AnswerQuizResponse struct {
	Status       string `json:"status"`
	PointsEarned int    `json:"points_earned"`
}
