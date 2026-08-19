package repository

import (
	"testing"

	"clap/internal/modules/match/models"
)

func TestAssembleClubSlider(t *testing.T) {
	pastOlder := models.Match{ProviderMatchID: "past-older"}
	pastNewer := models.Match{ProviderMatchID: "past-newer"}
	next := models.Match{ProviderMatchID: "next"}
	fut1 := models.Match{ProviderMatchID: "fut1"}
	fut2 := models.Match{ProviderMatchID: "fut2"}
	fut3 := models.Match{ProviderMatchID: "fut3"}
	live := models.Match{ProviderMatchID: "live"}

	t.Run("two past one next two future", func(t *testing.T) {
		got := assembleClubSlider(
			[]models.Match{pastNewer, pastOlder},
			nil,
			[]models.Match{next, fut1, fut2, fut3},
			2, 2,
		)
		assertIDs(t, got, "past-older", "past-newer", "next", "fut1", "fut2")
	})

	t.Run("live is the next slot", func(t *testing.T) {
		got := assembleClubSlider(
			[]models.Match{pastNewer, pastOlder},
			[]models.Match{live},
			[]models.Match{next, fut1, fut2},
			2, 2,
		)
		assertIDs(t, got, "past-older", "past-newer", "live", "next", "fut1")
	})

	t.Run("partial window", func(t *testing.T) {
		got := assembleClubSlider(
			[]models.Match{pastNewer},
			nil,
			[]models.Match{next},
			2, 2,
		)
		assertIDs(t, got, "past-newer", "next")
	})
}

func assertIDs(t *testing.T, got []models.Match, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), ids(got))
	}
	for i, id := range want {
		if got[i].ProviderMatchID != id {
			t.Fatalf("item[%d]=%q want %q (%v)", i, got[i].ProviderMatchID, id, ids(got))
		}
	}
}

func ids(items []models.Match) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ProviderMatchID
	}
	return out
}
