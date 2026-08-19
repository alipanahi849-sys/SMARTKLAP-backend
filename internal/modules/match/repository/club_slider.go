package repository

import "clap/internal/modules/match/models"

// assembleClubSlider builds the home slider window:
// 2 past (oldest of the pair first), 1 next (live or nearest scheduled), 2 future.
func assembleClubSlider(finishedNewestFirst, live, upcomingSoonestFirst []models.Match, pastCount, futureCount int) []models.Match {
	if pastCount < 0 {
		pastCount = 0
	}
	if futureCount < 0 {
		futureCount = 0
	}

	past := append([]models.Match(nil), finishedNewestFirst...)
	if pastCount > 0 && len(past) > pastCount {
		past = past[:pastCount]
	}
	for i, j := 0, len(past)-1; i < j; i, j = i+1, j-1 {
		past[i], past[j] = past[j], past[i]
	}

	upcoming := append([]models.Match(nil), upcomingSoonestFirst...)
	var next *models.Match
	if len(live) > 0 {
		next = &live[0]
	} else if len(upcoming) > 0 {
		item := upcoming[0]
		next = &item
		upcoming = upcoming[1:]
	}
	if futureCount > 0 && len(upcoming) > futureCount {
		upcoming = upcoming[:futureCount]
	}

	out := make([]models.Match, 0, len(past)+1+len(upcoming))
	out = append(out, past...)
	if next != nil {
		out = append(out, *next)
	}
	return append(out, upcoming...)
}
