package service

import (
	"context"
	"testing"
	"time"

	chantdto "clap/internal/modules/chant/dto"
	chantmodels "clap/internal/modules/chant/models"
	realtimedto "clap/internal/modules/realtime/dto"

	"github.com/google/uuid"
)

type stubUpcomingChants struct {
	chants []chantmodels.Chant
}

func (s *stubUpcomingChants) FindStartingBetween(context.Context, time.Time, time.Time) ([]chantmodels.Chant, error) {
	return s.chants, nil
}

type stubLyricsProvider struct {
	err error
}

func (s stubLyricsProvider) Lyrics(context.Context, uuid.UUID, string) (*chantdto.ChantLyricsResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &chantdto.ChantLyricsResponse{Title: "lyrics"}, nil
}

type recordingBroadcast struct {
	calls int
	last  *realtimedto.EventEnvelope
}

func (r *recordingBroadcast) BroadcastEnvelope(_ context.Context, env *realtimedto.EventEnvelope) error {
	r.calls++
	r.last = env
	return nil
}

type recordingWelcome struct {
	calls int
	last  []byte
}

func (w *recordingWelcome) SetWelcomeMessage(data []byte) {
	w.calls++
	w.last = data
}

type recordingPusher struct {
	calls int
	title string
	err   error
}

func (p *recordingPusher) NotifyChantCountdown(_ context.Context, _, _, title string, _ time.Time, _ time.Duration) error {
	p.calls++
	p.title = title
	return p.err
}

func TestChantUpcomingNotifierSendsPushOnce(t *testing.T) {
	t.Parallel()

	chantID := uuid.New()
	matchID := uuid.New()
	repo := &stubUpcomingChants{chants: []chantmodels.Chant{{
		ID:          chantID,
		MatchID:     matchID,
		Title:       "Yellow Submarine",
		ScheduledAt: time.Now().UTC().Add(90 * time.Second),
		Points:      10,
	}}}
	pusher := &recordingPusher{}
	n := NewChantUpcomingNotifier(repo, stubLyricsProvider{}, nil, &recordingBroadcast{}, &recordingWelcome{}, pusher, time.Second, 2*time.Minute)

	now := time.Now().UTC()
	n.tick(context.Background(), now)
	n.tick(context.Background(), now.Add(5*time.Second))

	if pusher.calls != 1 {
		t.Fatalf("push calls = %d, want 1", pusher.calls)
	}
	if pusher.title != "Yellow Submarine" {
		t.Fatalf("title = %q, want song name", pusher.title)
	}
	if !n.pending[chantID].pushSent {
		t.Fatal("expected pushSent after successful send")
	}
}

func TestChantUpcomingNotifierRetriesPushOnFailure(t *testing.T) {
	t.Parallel()

	chantID := uuid.New()
	repo := &stubUpcomingChants{chants: []chantmodels.Chant{{
		ID:          chantID,
		MatchID:     uuid.New(),
		Title:       "Hey Jude",
		ScheduledAt: time.Now().UTC().Add(90 * time.Second),
	}}}
	pusher := &recordingPusher{err: context.DeadlineExceeded}
	n := NewChantUpcomingNotifier(repo, stubLyricsProvider{}, nil, &recordingBroadcast{}, &recordingWelcome{}, pusher, time.Second, 2*time.Minute)

	now := time.Now().UTC()
	n.tick(context.Background(), now)
	if pusher.calls != 1 {
		t.Fatalf("push calls after fail = %d, want 1", pusher.calls)
	}
	if n.pending[chantID].pushSent {
		t.Fatal("pushSent should stay false after failure")
	}

	pusher.err = nil
	n.tick(context.Background(), now.Add(5*time.Second))
	if pusher.calls != 2 {
		t.Fatalf("push calls after retry = %d, want 2", pusher.calls)
	}
	if !n.pending[chantID].pushSent {
		t.Fatal("expected pushSent after successful retry")
	}
}

func TestChantUpcomingNotifierSkipsPushWhenDisabled(t *testing.T) {
	t.Parallel()

	repo := &stubUpcomingChants{chants: []chantmodels.Chant{{
		ID:          uuid.New(),
		MatchID:     uuid.New(),
		Title:       "Let It Be",
		ScheduledAt: time.Now().UTC().Add(90 * time.Second),
	}}}
	n := NewChantUpcomingNotifier(repo, stubLyricsProvider{}, nil, &recordingBroadcast{}, nil, nil, time.Second, 2*time.Minute)
	n.tick(context.Background(), time.Now().UTC())
}

func TestChantUpcomingNotifierBroadcastsOnce(t *testing.T) {
	t.Parallel()

	chantID := uuid.New()
	matchID := uuid.New()
	repo := &stubUpcomingChants{chants: []chantmodels.Chant{{
		ID:          chantID,
		MatchID:     matchID,
		Title:       "Seven Nation Army",
		ScheduledAt: time.Now().UTC().Add(90 * time.Second),
		Points:      25,
	}}}
	pub := &recordingBroadcast{}
	welcome := &recordingWelcome{}
	n := NewChantUpcomingNotifier(repo, stubLyricsProvider{}, nil, pub, welcome, nil, time.Second, 2*time.Minute)

	now := time.Now().UTC()
	n.tick(context.Background(), now)
	n.tick(context.Background(), now.Add(5*time.Second))
	n.tick(context.Background(), now.Add(10*time.Second))

	if pub.calls != 1 {
		t.Fatalf("broadcast calls = %d, want 1", pub.calls)
	}
	if pub.last == nil || pub.last.Type != realtimedto.EventTypeChantUpcoming {
		t.Fatalf("expected chant.upcoming envelope, got %#v", pub.last)
	}
	if len(welcome.last) == 0 {
		t.Fatal("expected welcome snapshot after broadcast")
	}
	if !n.pending[chantID].broadcastSent {
		t.Fatal("expected broadcastSent")
	}
}

func TestChantUpcomingNotifierClearsWelcomeAfterWindow(t *testing.T) {
	t.Parallel()

	chantID := uuid.New()
	repo := &stubUpcomingChants{chants: []chantmodels.Chant{{
		ID:          chantID,
		MatchID:     uuid.New(),
		Title:       "Wonderwall",
		ScheduledAt: time.Now().UTC().Add(30 * time.Second),
	}}}
	welcome := &recordingWelcome{}
	n := NewChantUpcomingNotifier(repo, stubLyricsProvider{}, nil, &recordingBroadcast{}, welcome, nil, time.Second, 2*time.Minute)

	now := time.Now().UTC()
	n.tick(context.Background(), now)
	if len(welcome.last) == 0 {
		t.Fatal("expected welcome after broadcast")
	}

	repo.chants = nil
	n.tick(context.Background(), now.Add(31*time.Second))
	if len(welcome.last) != 0 {
		t.Fatal("expected welcome cleared after chant left the window")
	}
}
