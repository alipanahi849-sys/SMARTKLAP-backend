package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Panel section keys used by the dashboard sidebar / route guards.
const (
	PanelDashboard   = "dashboard"
	PanelTeams       = "teams"
	PanelChants      = "chants"
	PanelEvents      = "events"
	PanelShop        = "shop"
	PanelVideos      = "videos"
	PanelQuiz        = "quiz"
	PanelUsers       = "users"
	PanelLeaderboard = "leaderboard"
	PanelRealtime    = "realtime"
	PanelProfile     = "profile"
)

// AllPanelPermissions is the canonical list shown in the admin UI.
var AllPanelPermissions = []string{
	PanelDashboard,
	PanelTeams,
	PanelChants,
	PanelEvents,
	PanelShop,
	PanelVideos,
	PanelQuiz,
	PanelUsers,
	PanelLeaderboard,
	PanelRealtime,
	PanelProfile,
}

// StringList is a JSON array of strings stored as JSONB.
type StringList []string

func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *StringList) Scan(value any) error {
	if value == nil {
		*s = StringList{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported panel_permissions type %T", value)
	}
	if len(raw) == 0 {
		*s = StringList{}
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*s = StringList(out)
	return nil
}

// Contains reports whether key is present.
func (s StringList) Contains(key string) bool {
	for _, item := range s {
		if item == key {
			return true
		}
	}
	return false
}
