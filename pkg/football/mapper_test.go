package football

import (
	"testing"
	"time"
)

func TestMapStatus(t *testing.T) {
	cases := map[string]string{
		"1H":   "live",
		"HT":   "halftime",
		"FT":   "finished",
		"NS":   "scheduled",
		"CANC": "cancelled",
		"LIVE": "live",
	}
	for in, want := range cases {
		if got := MapStatus(in); got != want {
			t.Fatalf("MapStatus(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFormatMinute(t *testing.T) {
	if got := FormatMinute(34, 0, "live"); got != "34" {
		t.Fatalf("live minute=%q", got)
	}
	if got := FormatMinute(90, 6, "live"); got != "90+6" {
		t.Fatalf("extra minute=%q", got)
	}
	if got := FormatMinute(45, 0, "halftime"); got != "HT" {
		t.Fatalf("ht=%q", got)
	}
}

func TestMapEventType(t *testing.T) {
	if got := MapEventType("Goal", "Normal Goal"); got != "goal" {
		t.Fatalf("goal=%q", got)
	}
	if got := MapEventType("Card", "Red Card"); got != "red" {
		t.Fatalf("red=%q", got)
	}
	if got := MapEventType("subst", "Substitution 1"); got != "change" {
		t.Fatalf("change=%q", got)
	}
}

func TestParseStatValue(t *testing.T) {
	if got := ParseStatValue("55%"); got != 55 {
		t.Fatalf("percent=%d", got)
	}
	if got := ParseStatValue(12.0); got != 12 {
		t.Fatalf("float=%d", got)
	}
	if got := ParseStatValue(nil); got != 0 {
		t.Fatalf("nil=%d", got)
	}
}

func TestSelectFixtureWindow(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	all := []Fixture{
		{ProviderID: "old", Status: "finished", Kickoff: "2024-05-01T19:00:00Z"},
		{ProviderID: "recent", Status: "finished", Kickoff: "2024-05-25T19:00:00Z"},
		{ProviderID: "older", Status: "finished", Kickoff: "2024-04-01T19:00:00Z"},
		{ProviderID: "soon", Status: "scheduled", Kickoff: "2026-08-16T19:00:00Z"},
		{ProviderID: "later", Status: "scheduled", Kickoff: "2026-08-23T19:00:00Z"},
		{ProviderID: "live", Status: "live", Kickoff: "2026-08-15T11:00:00Z"},
	}
	got := SelectFixtureWindow(all, 1, 2, now)
	if len(got) != 4 {
		t.Fatalf("len=%d want 4: %+v", len(got), ids(got))
	}
	if got[0].ProviderID != "live" {
		t.Fatalf("first=%q want live", got[0].ProviderID)
	}
	if got[1].ProviderID != "soon" {
		t.Fatalf("upcoming=%q want soon", got[1].ProviderID)
	}
	if got[2].ProviderID != "recent" || got[3].ProviderID != "old" {
		t.Fatalf("finished=%v want recent,old", ids(got[2:]))
	}
}

func ids(items []Fixture) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ProviderID
	}
	return out
}
