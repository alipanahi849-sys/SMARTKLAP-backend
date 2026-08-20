package repository

import (
	"context"
	"strings"
	"time"

	authmodels "clap/internal/modules/auth/models"
	"clap/internal/modules/chant/models"
	songmodels "clap/internal/modules/song/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChantCursorAnchor is the list position after which the next page is fetched.
type ChantCursorAnchor struct {
	ScheduledAt time.Time
	ID          uuid.UUID
}

// CompletionTarget is what a user just finished. Online completions carry a
// ChantID; catalog completions carry only the song.
type CompletionTarget struct {
	UserID  uuid.UUID
	ChantID *uuid.UUID
	SongID  uuid.UUID
	Source  string
	Points  int
}

// ProgramCompletion is a points row for the Home scoreboard, joined with the
// chant or song title and the user who earned it.
type ProgramCompletion struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	FirstName    string
	LastName     string
	Title        string
	PointsEarned int
	CreatedAt    time.Time
}

// PendingChant is a scheduled chant the user has not sung yet today.
type PendingChant struct {
	ID          uuid.UUID
	Title       string
	ScheduledAt time.Time
}

type ChantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Chant, error)
	// FindSongByID loads a catalog song; catalog chant IDs are song IDs.
	FindSongByID(ctx context.Context, id uuid.UUID) (*songmodels.Song, error)
	// FindCatalogSongs lists the predefined song library shown on the Chants
	// screen, independent of any match or schedule.
	FindCatalogSongs(ctx context.Context, search string, limit int) ([]songmodels.Song, error)
	// CompletedSongIDs returns which of the given songs the user already earned
	// catalog points for.
	CompletedSongIDs(ctx context.Context, userID uuid.UUID, songIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// IsCompleted reports whether this exact target was already awarded.
	IsCompleted(ctx context.Context, target CompletionTarget) (bool, error)
	// StartListenSession stamps the moment the user received the lyrics.
	StartListenSession(ctx context.Context, userID, songID uuid.UUID, source string) error
	// ListenStartedAt returns that stamp, or nil when the user never opened the
	// lyrics for this song.
	ListenStartedAt(ctx context.Context, userID, songID uuid.UUID, source string) (*time.Time, error)
	// TodayProgramFeed returns today's completions across all users, newest first.
	TodayProgramFeed(ctx context.Context, limit int) ([]ProgramCompletion, error)
	// PendingTodayChants returns today's scheduled chants the user has not
	// completed, soonest first.
	PendingTodayChants(ctx context.Context, userID uuid.UUID, limit int) ([]PendingChant, error)
	// CreateChant schedules an online chant built from a catalog song.
	CreateChant(ctx context.Context, chant *models.Chant) error
	// FindScheduled lists active online chants for the admin panel, soonest first.
	FindScheduled(ctx context.Context, matchID *uuid.UUID, limit int) ([]models.Chant, error)
	// DeactivateChant unschedules an online chant.
	DeactivateChant(ctx context.Context, id uuid.UUID) error
	// FindByMatchAfter returns active chants for a match ordered by schedule time.
	// Pass after=nil for the first page.
	FindByMatchAfter(ctx context.Context, matchID uuid.UUID, search string, limit int, after *ChantCursorAnchor) ([]models.Chant, error)
	// HasIncompleteAtOrBefore reports whether the user has not completed any chant
	// at or before the given anchor in the match list order.
	HasIncompleteAtOrBefore(ctx context.Context, userID, matchID uuid.UUID, search string, anchor *ChantCursorAnchor) (bool, error)
	// CompletedChantIDs returns which of the given chants the user completed.
	CompletedChantIDs(ctx context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// TodayPoints sums the user's chant points earned since local midnight (UTC).
	TodayPoints(ctx context.Context, userID uuid.UUID) (int, error)
	// TodayCompletions returns the user's most recent completions today with
	// their chants preloaded (Home "chant program" card).
	TodayCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChantCompletion, map[uuid.UUID]models.Chant, error)
	// Complete records a completion and atomically credits the user's points.
	// Returns created=false when the target was already awarded (idempotent).
	Complete(ctx context.Context, target CompletionTarget) (totalPoints int, created bool, err error)
	// FindStartingBetween returns active chants scheduled to start within
	// (from, to], ordered by schedule time. Used by the realtime upcoming-chant
	// notifier.
	FindStartingBetween(ctx context.Context, from, to time.Time) ([]models.Chant, error)
	// FindActiveByMatch returns the chant currently in progress for a match, if any.
	FindActiveByMatch(ctx context.Context, matchID uuid.UUID, now time.Time) (*models.Chant, error)
}

type chantRepository struct {
	db *gorm.DB
}

func NewChantRepository(db *gorm.DB) ChantRepository {
	return &chantRepository{db: db}
}

func (r *chantRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Chant, error) {
	var chant models.Chant
	err := r.db.WithContext(ctx).Preload("Song").First(&chant, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Chant not found", nil)
		}
		return nil, errors.NewInternal("Failed to find chant", err)
	}
	return &chant, nil
}

func (r *chantRepository) FindByMatchAfter(ctx context.Context, matchID uuid.UUID, search string, limit int, after *ChantCursorAnchor) ([]models.Chant, error) {
	q := r.db.WithContext(ctx).Preload("Song").
		Where("match_id = ? AND is_active = ?", matchID, true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("title ILIKE ?", "%"+s+"%")
	}
	if after != nil {
		q = q.Where(
			"(scheduled_at > ?) OR (scheduled_at = ? AND id > ?)",
			after.ScheduledAt, after.ScheduledAt, after.ID,
		)
	}

	var chants []models.Chant
	if err := q.Order("scheduled_at ASC, id ASC").
		Limit(limit).
		Find(&chants).Error; err != nil {
		return nil, errors.NewInternal("Failed to list chants", err)
	}
	return chants, nil
}

func (r *chantRepository) HasIncompleteAtOrBefore(ctx context.Context, userID, matchID uuid.UUID, search string, anchor *ChantCursorAnchor) (bool, error) {
	if anchor == nil {
		return false, nil
	}

	q := r.db.WithContext(ctx).Table("chants c").
		Joins("LEFT JOIN chant_completions cc ON cc.chant_id = c.id AND cc.user_id = ? AND cc.source = ?", userID, models.SourceOnline).
		Where("c.match_id = ? AND c.is_active = ? AND cc.id IS NULL", matchID, true).
		Where(
			"(c.scheduled_at < ?) OR (c.scheduled_at = ? AND c.id <= ?)",
			anchor.ScheduledAt, anchor.ScheduledAt, anchor.ID,
		)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("c.title ILIKE ?", "%"+s+"%")
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, errors.NewInternal("Failed to check chant progress", err)
	}
	return count > 0, nil
}

func (r *chantRepository) CompletedChantIDs(ctx context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	done := make(map[uuid.UUID]bool, len(chantIDs))
	if len(chantIDs) == 0 {
		return done, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND source = ? AND chant_id IN ?", userID, models.SourceOnline, chantIDs).
		Pluck("chant_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load completions", err)
	}
	for _, id := range ids {
		done[id] = true
	}
	return done, nil
}

func (r *chantRepository) TodayPoints(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	err := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND created_at >= ?", userID, startOfDay).
		Select("COALESCE(SUM(points_earned), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, errors.NewInternal("Failed to sum today's points", err)
	}
	return int(total), nil
}

func (r *chantRepository) TodayCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChantCompletion, map[uuid.UUID]models.Chant, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)

	var completions []models.ChantCompletion
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, startOfDay).
		Order("created_at DESC").
		Limit(limit).
		Find(&completions).Error; err != nil {
		return nil, nil, errors.NewInternal("Failed to load completions", err)
	}

	chantByID := make(map[uuid.UUID]models.Chant, len(completions))
	ids := make([]uuid.UUID, 0, len(completions))
	for _, c := range completions {
		if c.ChantID != nil {
			ids = append(ids, *c.ChantID)
		}
	}
	if len(ids) > 0 {
		var chants []models.Chant
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&chants).Error; err != nil {
			return nil, nil, errors.NewInternal("Failed to load chants", err)
		}
		for _, c := range chants {
			chantByID[c.ID] = c
		}
	}
	return completions, chantByID, nil
}

func (r *chantRepository) FindStartingBetween(ctx context.Context, from, to time.Time) ([]models.Chant, error) {
	var chants []models.Chant
	if err := r.db.WithContext(ctx).Preload("Song").
		Where("is_active = ? AND scheduled_at > ? AND scheduled_at <= ?", true, from, to).
		Order("scheduled_at ASC").
		Find(&chants).Error; err != nil {
		return nil, errors.NewInternal("Failed to list upcoming chants", err)
	}
	return chants, nil
}

func (r *chantRepository) FindActiveByMatch(ctx context.Context, matchID uuid.UUID, now time.Time) (*models.Chant, error) {
	var chant models.Chant
	err := r.db.WithContext(ctx).Preload("Song").
		Where(
			"match_id = ? AND is_active = ? AND scheduled_at <= ? AND scheduled_at + make_interval(secs => duration_seconds) > ?",
			matchID, true, now, now,
		).
		Order("scheduled_at DESC").
		First(&chant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternal("Failed to find active chant", err)
	}
	return &chant, nil
}

func (r *chantRepository) FindSongByID(ctx context.Context, id uuid.UUID) (*songmodels.Song, error) {
	var song songmodels.Song
	err := r.db.WithContext(ctx).First(&song, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Chant not found", nil)
		}
		return nil, errors.NewInternal("Failed to find song", err)
	}
	return &song, nil
}

func (r *chantRepository) FindCatalogSongs(ctx context.Context, search string, limit int) ([]songmodels.Song, error) {
	q := r.db.WithContext(ctx).Where("is_active = ?", true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("title ILIKE ? OR artist ILIKE ?", "%"+s+"%", "%"+s+"%")
	}

	var songs []songmodels.Song
	if err := q.Order("title ASC, id ASC").Limit(limit).Find(&songs).Error; err != nil {
		return nil, errors.NewInternal("Failed to list chant catalog", err)
	}
	return songs, nil
}

func (r *chantRepository) CompletedSongIDs(ctx context.Context, userID uuid.UUID, songIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	done := make(map[uuid.UUID]bool, len(songIDs))
	if len(songIDs) == 0 {
		return done, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND source = ? AND song_id IN ?", userID, models.SourceCatalog, songIDs).
		Pluck("song_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load completions", err)
	}
	for _, id := range ids {
		done[id] = true
	}
	return done, nil
}

func (r *chantRepository) IsCompleted(ctx context.Context, target CompletionTarget) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND source = ?", target.UserID, target.Source)
	if target.Source == models.SourceCatalog {
		q = q.Where("song_id = ?", target.SongID)
	} else {
		q = q.Where("chant_id = ?", target.ChantID)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, errors.NewInternal("Failed to check completion", err)
	}
	return count > 0, nil
}

func (r *chantRepository) StartListenSession(ctx context.Context, userID, songID uuid.UUID, source string) error {
	// Stamped from Go in UTC so the comparison in the service does not depend on
	// the database server's timezone.
	startedAt := time.Now().UTC()

	const upsert = `
		INSERT INTO chant_listen_sessions (user_id, song_id, source, started_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, song_id, source) DO UPDATE SET started_at = EXCLUDED.started_at`

	if err := r.db.WithContext(ctx).Exec(upsert, userID, songID, source, startedAt).Error; err != nil {
		return errors.NewInternal("Failed to start listen session", err)
	}
	return nil
}

func (r *chantRepository) ListenStartedAt(ctx context.Context, userID, songID uuid.UUID, source string) (*time.Time, error) {
	var session models.ChantListenSession
	err := r.db.WithContext(ctx).
		First(&session, "user_id = ? AND song_id = ? AND source = ?", userID, songID, source).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternal("Failed to load listen session", err)
	}
	return &session.StartedAt, nil
}

func (r *chantRepository) TodayProgramFeed(ctx context.Context, limit int) ([]ProgramCompletion, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)

	const query = `
		SELECT cc.id,
		       cc.user_id,
		       u.first_name,
		       u.last_name,
		       COALESCE(c.title, s.title, '') AS title,
		       cc.points_earned,
		       cc.created_at
		FROM chant_completions cc
		JOIN users u       ON u.id = cc.user_id
		LEFT JOIN chants c ON c.id = cc.chant_id
		LEFT JOIN songs s  ON s.id = cc.song_id
		WHERE cc.created_at >= ?
		ORDER BY cc.created_at DESC
		LIMIT ?`

	var rows []ProgramCompletion
	if err := r.db.WithContext(ctx).Raw(query, startOfDay, limit).Scan(&rows).Error; err != nil {
		return nil, errors.NewInternal("Failed to load chant scores", err)
	}
	return rows, nil
}

func (r *chantRepository) PendingTodayChants(ctx context.Context, userID uuid.UUID, limit int) ([]PendingChant, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	const query = `
		SELECT c.id, c.title, c.scheduled_at
		FROM chants c
		LEFT JOIN chant_completions cc
		       ON cc.chant_id = c.id AND cc.user_id = ? AND cc.source = 'online'
		WHERE c.is_active = TRUE
		  AND c.deleted_at IS NULL
		  AND c.scheduled_at >= ?
		  AND c.scheduled_at < ?
		  AND cc.id IS NULL
		ORDER BY c.scheduled_at ASC
		LIMIT ?`

	var rows []PendingChant
	if err := r.db.WithContext(ctx).Raw(query, userID, startOfDay, endOfDay, limit).Scan(&rows).Error; err != nil {
		return nil, errors.NewInternal("Failed to load pending chants", err)
	}
	return rows, nil
}

func (r *chantRepository) CreateChant(ctx context.Context, chant *models.Chant) error {
	if err := r.db.WithContext(ctx).Create(chant).Error; err != nil {
		return errors.NewInternal("Failed to schedule online chant", err)
	}
	return nil
}

func (r *chantRepository) FindScheduled(ctx context.Context, matchID *uuid.UUID, limit int) ([]models.Chant, error) {
	q := r.db.WithContext(ctx).Preload("Song").Where("is_active = ?", true)
	if matchID != nil {
		q = q.Where("match_id = ?", *matchID)
	}

	var chants []models.Chant
	if err := q.Order("scheduled_at ASC, id ASC").Limit(limit).Find(&chants).Error; err != nil {
		return nil, errors.NewInternal("Failed to list scheduled chants", err)
	}
	return chants, nil
}

func (r *chantRepository) DeactivateChant(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&models.Chant{}).
		Where("id = ?", id).
		UpdateColumn("is_active", false)
	if res.Error != nil {
		return errors.NewInternal("Failed to unschedule chant", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Chant not found", nil)
	}
	return nil
}

// insertCompletion writes the row, letting the partial unique index decide
// whether this is a first award. Catalog rows conflict on (user, song) and
// online rows on (user, chant), so the same song can be earned once from the
// library and once again as a scheduled chant.
func insertCompletion(tx *gorm.DB, target CompletionTarget) (int64, error) {
	const insert = `
		INSERT INTO chant_completions (chant_id, song_id, source, user_id, points_earned)
		VALUES (?, ?, ?, ?, ?)`

	conflict := " ON CONFLICT (user_id, chant_id) WHERE source = 'online' DO NOTHING"
	if target.Source == models.SourceCatalog {
		conflict = " ON CONFLICT (user_id, song_id) WHERE source = 'catalog' DO NOTHING"
	}

	res := tx.Exec(insert+conflict,
		target.ChantID, target.SongID, target.Source, target.UserID, target.Points)
	return res.RowsAffected, res.Error
}

func (r *chantRepository) Complete(ctx context.Context, target CompletionTarget) (int, bool, error) {
	var totalPoints int
	created := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, insertErr := insertCompletion(tx, target)
		if insertErr != nil {
			return errors.NewInternal("Failed to record completion", insertErr)
		}
		userID := target.UserID
		points := target.Points
		if rows == 0 {
			if err := tx.Model(&authmodels.User{}).
				Where("id = ?", userID).
				Pluck("points", &totalPoints).Error; err != nil {
				return errors.NewInternal("Failed to read points balance", err)
			}
			return nil
		}

		created = true
		if err := tx.Model(&authmodels.User{}).
			Where("id = ?", userID).
			UpdateColumn("points", gorm.Expr("points + ?", points)).Error; err != nil {
			return errors.NewInternal("Failed to award points", err)
		}

		if err := tx.Model(&authmodels.User{}).
			Where("id = ?", userID).
			Pluck("points", &totalPoints).Error; err != nil {
			return errors.NewInternal("Failed to read points balance", err)
		}
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return totalPoints, created, nil
}
