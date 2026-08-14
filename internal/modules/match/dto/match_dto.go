package dto

import "github.com/google/uuid"

type TeamSummary struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
	Score   *int   `json:"score,omitempty"`
}

type CurrentMatchEnvelope struct {
	Match *CurrentMatchResponse `json:"match"`
}

type CurrentMatchResponse struct {
	ID                 uuid.UUID   `json:"id"`
	Status             string      `json:"status"`
	Minute             string      `json:"minute"`
	Stadium            string      `json:"stadium"`
	KickoffAt          string      `json:"kickoff_at"`
	CompetitionLogoURL string      `json:"competition_logo_url,omitempty"`
	HomeTeam           TeamSummary `json:"home_team"`
	AwayTeam           TeamSummary `json:"away_team"`
}

type MatchListEnvelope struct {
	Items []CurrentMatchResponse `json:"items"`
}

type StatItem struct {
	Label string `json:"label"`
	Home  int    `json:"home"`
	Away  int    `json:"away"`
}

type TimelineItem struct {
	Kind        string `json:"kind"`
	Minute      string `json:"minute,omitempty"`
	Score       string `json:"score,omitempty"`
	Side        string `json:"side,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Sub         string `json:"sub,omitempty"`
	Highlighted bool   `json:"highlighted,omitempty"`
}

type SquadPlayer struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Name     string     `json:"name"`
	Position string     `json:"position"`
	PhotoURL string     `json:"photo_url"`
}

type SquadGroup struct {
	Title   string        `json:"title"`
	Players []SquadPlayer `json:"players"`
}

type MatchDetailResponse struct {
	ID                 uuid.UUID      `json:"id"`
	Status             string         `json:"status"`
	HomeTeam           TeamSummary    `json:"home_team"`
	AwayTeam           TeamSummary    `json:"away_team"`
	Score              string         `json:"score"`
	Minute             string         `json:"minute"`
	Stadium            string         `json:"stadium"`
	CompetitionLogoURL string         `json:"competition_logo_url"`
	Stats              []StatItem     `json:"stats"`
	Timeline           []TimelineItem `json:"timeline"`
	HomeSquads         []SquadGroup   `json:"home_squads"`
	AwaySquads         []SquadGroup   `json:"away_squads"`
}

type PlayerDetailResponse struct {
	ID                 uuid.UUID   `json:"id"`
	Name               string      `json:"name"`
	JerseyNumber       int         `json:"jersey_number"`
	Club               string      `json:"club"`
	ClubLogoURL        string      `json:"club_logo_url"`
	Age                int         `json:"age"`
	PreferredFoot      string      `json:"preferred_foot"`
	Nationality        string      `json:"nationality"`
	HeightCM           int         `json:"height_cm"`
	WeightKG           int         `json:"weight_kg"`
	WeakFootPercentage int         `json:"weak_foot_percentage"`
	PhotoURL           string      `json:"photo_url"`
	RadarStats         []RadarStat `json:"radar_stats"`
	Formation          string      `json:"formation"`
}

type RadarStat struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type ProviderTeam struct {
	ProviderTeamID string     `json:"provider_team_id"`
	Name           string     `json:"name"`
	ShortName      string     `json:"short_name"`
	Country        string     `json:"country"`
	LogoURL        string     `json:"logo_url"`
	VenueName      string     `json:"venue_name"`
	ClubID         *uuid.UUID `json:"club_id,omitempty"`
}

type FeaturedClubResponse struct {
	ClubID         *uuid.UUID `json:"club_id"`
	Name           string     `json:"name,omitempty"`
	LogoURL        string     `json:"logo_url,omitempty"`
	ProviderTeamID string     `json:"provider_team_id,omitempty"`
	Provider       string     `json:"provider,omitempty"`
}

type SetFeaturedClubRequest struct {
	ClubID         *uuid.UUID `json:"club_id"`
	ProviderTeamID string     `json:"provider_team_id"`
}

type SyncResponse struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
}
