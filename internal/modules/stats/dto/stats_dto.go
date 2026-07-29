package dto

import "github.com/google/uuid"

// RadarStat is one axis of the player radar chart (contract §9.2).
type RadarStat struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// PlayerDetailResponse is GET /players/{player_id}.
type PlayerDetailResponse struct {
	ID                 uuid.UUID   `json:"id"`
	Name               string      `json:"name"`
	JerseyNumber       int         `json:"jersey_number"`
	Club               string      `json:"club"`
	Age                int         `json:"age"`
	PreferredFoot      string      `json:"preferred_foot"`
	Nationality        string      `json:"nationality"`
	HeightCm           int         `json:"height_cm"`
	WeightKg           int         `json:"weight_kg"`
	WeakFootPercentage int         `json:"weak_foot_percentage"`
	PhotoURL           string      `json:"photo_url"`
	RadarStats         []RadarStat `json:"radar_stats"`
	Formation          string      `json:"formation"`
}
