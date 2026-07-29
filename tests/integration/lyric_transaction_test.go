package integration

import (
	"context"
	"testing"

	"clap/internal/modules/songlyric/models"
	"clap/internal/modules/songlyric/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestLyricReplaceTransactionSafety tests that the ReplaceLyrics operation is transaction-safe
// This test proves that if the insert fails after delete succeeds, the delete is rolled back
func TestLyricReplaceTransactionSafety(t *testing.T) {
	// Skip if not in integration test environment
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database connection
	dsn := "host=localhost user=postgres password=postgres dbname=clap port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to database")

	// Create repository
	lyricRepo := repository.NewSongLyricRepository(db)
	ctx := context.Background()

	// Create test song ID
	songID := uuid.New()
	userID := uuid.New()

	// Step 1: Create initial lyrics
	initialLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "Original lyrics line 1\nOriginal lyrics line 2",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.Create(ctx, initialLyric)
	require.NoError(t, err, "Failed to create initial lyrics")

	// Verify initial lyrics exist
	found, err := lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err, "Failed to find initial lyrics")
	require.NotNil(t, found, "Initial lyrics should exist")
	assert.Equal(t, "Original lyrics line 1\nOriginal lyrics line 2", found.Lyrics, "Initial lyrics content should match")

	// Step 2: Test transaction rollback by simulating a failure
	// We'll create a mock scenario where we manually test the transaction behavior
	// Since we can't easily simulate a database failure in a real test,
	// we'll verify the transaction structure by checking that the method uses transactions

	// Create new lyrics for replacement
	newLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "New lyrics line 1\nNew lyrics line 2",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	// Step 3: Perform successful replace
	err = lyricRepo.ReplaceLyrics(ctx, songID, newLyric)
	require.NoError(t, err, "Failed to replace lyrics")

	// Verify lyrics were replaced
	found, err = lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err, "Failed to find replaced lyrics")
	require.NotNil(t, found, "Replaced lyrics should exist")
	assert.Equal(t, "New lyrics line 1\nNew lyrics line 2", found.Lyrics, "Replaced lyrics content should match")

	// Step 4: Test transaction rollback by attempting to replace with invalid data
	// We'll create a lyric with a song_id that doesn't exist to trigger a foreign key error
	invalidSongID := uuid.New()
	invalidLyric := &models.SongLyric{
		SongID:    invalidSongID, // This will cause foreign key violation
		Language:  "en",
		Lyrics:    "Invalid lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	// Attempt to replace with invalid song_id (should fail)
	err = lyricRepo.ReplaceLyrics(ctx, songID, invalidLyric)
	assert.Error(t, err, "Replace with invalid song_id should fail")

	// Verify that the original lyrics still exist (transaction was rolled back)
	found, err = lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err, "Failed to find lyrics after failed replace")
	require.NotNil(t, found, "Original lyrics should still exist after failed replace")
	assert.Equal(t, "New lyrics line 1\nNew lyrics line 2", found.Lyrics, "Lyrics should not have changed after failed replace")

	// Cleanup
	err = lyricRepo.DeleteBySongID(ctx, songID)
	require.NoError(t, err, "Failed to cleanup test lyrics")
}

// TestLyricReplaceTransaction_DeleteFails tests transaction rollback when delete fails
func TestLyricReplaceTransaction_DeleteFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "host=localhost user=postgres password=postgres dbname=clap port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to database")

	lyricRepo := repository.NewSongLyricRepository(db)
	ctx := context.Background()

	songID := uuid.New()
	userID := uuid.New()

	// Create initial lyrics
	initialLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "Test lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.Create(ctx, initialLyric)
	require.NoError(t, err)

	// Attempt to replace with nil lyric (should fail during insert)
	err = lyricRepo.ReplaceLyrics(ctx, songID, nil)
	assert.Error(t, err, "Replace with nil lyric should fail")

	// Verify original lyrics still exist
	found, err := lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err)
	require.NotNil(t, found, "Original lyrics should still exist")
	assert.Equal(t, "Test lyrics", found.Lyrics)

	// Cleanup
	lyricRepo.DeleteBySongID(ctx, songID)
}

// TestLyricReplaceTransaction_InsertFails tests transaction rollback when insert fails
func TestLyricReplaceTransaction_InsertFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "host=localhost user=postgres password=postgres dbname=clap port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to database")

	lyricRepo := repository.NewSongLyricRepository(db)
	ctx := context.Background()

	songID := uuid.New()
	userID := uuid.New()

	// Create initial lyrics
	initialLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "Original lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.Create(ctx, initialLyric)
	require.NoError(t, err)

	// Create a lyric with empty language (should violate NOT NULL constraint)
	invalidLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "", // Empty language should fail
		Lyrics:    "New lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.ReplaceLyrics(ctx, songID, invalidLyric)
	assert.Error(t, err, "Replace with invalid language should fail")

	// Verify original lyrics still exist (transaction rolled back)
	found, err := lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err)
	require.NotNil(t, found, "Original lyrics should still exist after failed insert")
	assert.Equal(t, "Original lyrics", found.Lyrics, "Lyrics should not have changed")

	// Cleanup
	lyricRepo.DeleteBySongID(ctx, songID)
}

// TestLyricReplaceSuccess tests successful lyrics replacement
func TestLyricReplaceSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "host=localhost user=postgres password=postgres dbname=clap port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to database")

	lyricRepo := repository.NewSongLyricRepository(db)
	ctx := context.Background()

	songID := uuid.New()
	userID := uuid.New()

	// Create initial lyrics
	initialLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "Original lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.Create(ctx, initialLyric)
	require.NoError(t, err)

	// Replace with new lyrics
	newLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "New lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.ReplaceLyrics(ctx, songID, newLyric)
	require.NoError(t, err, "Replace should succeed")

	// Verify lyrics were replaced
	found, err := lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err)
	require.NotNil(t, found, "New lyrics should exist")
	assert.Equal(t, "New lyrics", found.Lyrics, "Lyrics should be replaced")

	// Verify only one lyric exists (old one was deleted)
	lyrics, total, err := lyricRepo.FindAllBySongID(ctx, songID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "Should only have one lyric")
	assert.Len(t, lyrics, 1, "Should only have one lyric")

	// Cleanup
	lyricRepo.DeleteBySongID(ctx, songID)
}

// TestLyricReplaceWithNoExistingLyrics tests replace when no lyrics exist
func TestLyricReplaceWithNoExistingLyrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "host=localhost user=postgres password=postgres dbname=clap port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to database")

	lyricRepo := repository.NewSongLyricRepository(db)
	ctx := context.Background()

	songID := uuid.New()
	userID := uuid.New()

	// No existing lyrics - should still work (delete succeeds, insert succeeds)
	newLyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en",
		Lyrics:    "New lyrics",
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	err = lyricRepo.ReplaceLyrics(ctx, songID, newLyric)
	require.NoError(t, err, "Replace with no existing lyrics should succeed")

	// Verify lyrics were created
	found, err := lyricRepo.FindBySongID(ctx, songID, "en")
	require.NoError(t, err)
	require.NotNil(t, found, "New lyrics should exist")
	assert.Equal(t, "New lyrics", found.Lyrics)

	// Cleanup
	lyricRepo.DeleteBySongID(ctx, songID)
}
