package football

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// MapStatus converts an API-Football short status into our match status.
func MapStatus(short string) string {
	switch strings.ToUpper(strings.TrimSpace(short)) {
	case "1H", "2H", "ET", "BT", "P", "LIVE", "INT", "SUSP":
		return "live"
	case "HT":
		return "halftime"
	case "FT", "AET", "PEN":
		return "finished"
	case "CANC", "ABD", "AWD", "WO":
		return "cancelled"
	default:
		return "scheduled"
	}
}

// FormatMinute renders elapsed + extra time the way the stadium UI expects.
func FormatMinute(elapsed, extra int, status string) string {
	switch status {
	case "halftime":
		return "HT"
	case "finished":
		if extra > 0 {
			return strconv.Itoa(elapsed) + "+" + strconv.Itoa(extra)
		}
		if elapsed > 0 {
			return strconv.Itoa(elapsed)
		}
		return "FT"
	case "live":
		if extra > 0 {
			return strconv.Itoa(elapsed) + "+" + strconv.Itoa(extra)
		}
		if elapsed <= 0 {
			return "1"
		}
		return strconv.Itoa(elapsed)
	default:
		return ""
	}
}

// MapEventType converts an API-Football event into the mobile timeline type.
func MapEventType(eventType, detail string) string {
	t := strings.ToLower(strings.TrimSpace(eventType))
	d := strings.ToLower(strings.TrimSpace(detail))
	switch t {
	case "goal":
		return "goal"
	case "card":
		if strings.Contains(d, "red") {
			return "red"
		}
		return "yellow"
	case "subst":
		return "change"
	}
	if strings.Contains(d, "corner") || t == "corner" {
		return "corner"
	}
	return ""
}

// MapPosition maps API-Football position codes to the UI group titles.
func MapPosition(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "G", "GK", "GOALKEEPER", "GOALER":
		return "Goaler"
	case "D", "DF", "DEFENDER":
		return "Defender"
	case "M", "MF", "MIDFIELDER":
		return "Midfielder"
	case "F", "FW", "ATTACKER", "FORWARD":
		return "Forward"
	default:
		if raw == "" {
			return "Midfielder"
		}
		return raw
	}
}

// PositionSortOrder keeps squad groups in the same order as the mock UI.
func PositionSortOrder(position string) int {
	switch MapPosition(position) {
	case "Forward":
		return 0
	case "Midfielder":
		return 1
	case "Defender":
		return 2
	case "Goaler":
		return 3
	default:
		return 4
	}
}

var statLabelMap = map[string]string{
	"total shots":       "Total shots",
	"shots total":       "Total shots",
	"shots on goal":     "Shots on target",
	"shots on target":   "Shots on target",
	"ball possession":   "Possession",
	"passes":            "Passess",
	"total passes":      "Passess",
	"passes %":          "Pass accuracy",
	"passes percentage": "Pass accuracy",
	"fouls":             "Fouls",
	"yellow cards":      "Yellow card",
}

var preferredStatOrder = []string{
	"Total shots",
	"Shots on target",
	"Possession",
	"Passess",
	"Pass accuracy",
	"Fouls",
	"Yellow card",
}

// MapStatLabel remaps provider labels onto the stadium Statistics tab copy.
func MapStatLabel(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if mapped, ok := statLabelMap[key]; ok {
		return mapped
	}
	return strings.TrimSpace(raw)
}

// ParseStatValue turns "55%" / "12" / nil into an integer for StatBar.
func ParseStatValue(raw any) int {
	switch v := raw.(type) {
	case nil:
		return 0
	case int:
		return v
	case float64:
		return int(v)
	case string:
		s := strings.TrimSpace(strings.TrimSuffix(v, "%"))
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

// PreferredStatOrder is the Statistics tab order from the stadium mock.
func PreferredStatOrder() []string {
	return append([]string(nil), preferredStatOrder...)
}

// SelectFixtureWindow keeps live matches, the next upcoming fixtures, and the
// most recent finished ones from a full-season payload.
func SelectFixtureWindow(all []Fixture, next, last int, now time.Time) []Fixture {
	if next < 0 {
		next = 0
	}
	if last < 0 {
		last = 0
	}

	var live, upcoming, finished []Fixture
	for _, item := range all {
		switch item.Status {
		case "live", "halftime":
			live = append(live, item)
		case "finished", "cancelled":
			finished = append(finished, item)
		default:
			kickoff, err := time.Parse(time.RFC3339, item.Kickoff)
			if err == nil && kickoff.Before(now) {
				finished = append(finished, item)
			} else {
				upcoming = append(upcoming, item)
			}
		}
	}

	sort.Slice(live, func(i, j int) bool { return live[i].Kickoff < live[j].Kickoff })
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].Kickoff < upcoming[j].Kickoff })
	sort.Slice(finished, func(i, j int) bool { return finished[i].Kickoff > finished[j].Kickoff })

	if next > 0 && len(upcoming) > next {
		upcoming = upcoming[:next]
	}
	if last > 0 && len(finished) > last {
		finished = finished[:last]
	}

	out := make([]Fixture, 0, len(live)+len(upcoming)+len(finished))
	out = append(out, live...)
	out = append(out, upcoming...)
	out = append(out, finished...)
	return out
}
