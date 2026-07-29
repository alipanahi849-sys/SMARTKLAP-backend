package dto

import "github.com/google/uuid"

// GuessMatchInfo is the header card of the Guess Match screen (contract §5.1).
type GuessMatchInfo struct {
	HomeName           string `json:"home_name"`
	HomeRole           string `json:"home_role"`
	HomeLogoURL        string `json:"home_logo_url"`
	AwayName           string `json:"away_name"`
	AwayRole           string `json:"away_role"`
	AwayLogoURL        string `json:"away_logo_url"`
	CompetitionLogoURL string `json:"competition_logo_url"`
	Date               string `json:"date"`
	Time               string `json:"time"`
}

// GuessQuizItem is a quiz row in the Guess Match screen.
type GuessQuizItem struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Points int       `json:"points"`
	IsDone bool      `json:"is_done"`
}

// GuessMatchResponse is GET /guess/matches/{match_id}.
type GuessMatchResponse struct {
	Match               GuessMatchInfo  `json:"match"`
	ParticipationPoints int             `json:"participation_points"`
	Quizzes             []GuessQuizItem `json:"quizzes"`
}

// QuizOptionItem is one selectable option (contract §5.2).
type QuizOptionItem struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
}

// QuizDetailResponse is GET /guess/quizzes/{quiz_id}.
type QuizDetailResponse struct {
	ID      uuid.UUID        `json:"id"`
	Title   string           `json:"title"`
	Type    string           `json:"type"`
	Options []QuizOptionItem `json:"options"`
}

// AnswerQuizRequest is POST /guess/quizzes/{quiz_id}/answer.
type AnswerQuizRequest struct {
	Choice string `json:"choice" binding:"required,max=255"`
}

// AnswerQuizResponse confirms a submission.
type AnswerQuizResponse struct {
	Status       string `json:"status"`
	PointsEarned int    `json:"points_earned"`
}
