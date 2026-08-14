package football

import "testing"

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
