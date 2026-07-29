# Run database migrations for Clap backend

Write-Host "Running database migrations..."

$migrations = @(
    "pkg/migrations/001_init_schema.up.sql",
    "pkg/migrations/002_create_leagues.up.sql",
    "pkg/migrations/003_create_seasons.up.sql",
    "pkg/migrations/004_create_clubs.up.sql",
    "pkg/migrations/005_create_club_seasons.up.sql",
    "pkg/migrations/006_create_matches.up.sql",
    "pkg/migrations/007_create_songs.up.sql",
    "pkg/migrations/008_create_song_lyrics.up.sql",
    "pkg/migrations/009_create_match_song_schedules.up.sql",
    "pkg/migrations/010_add_club_id_to_users.up.sql",
    "pkg/migrations/011_add_unique_active_season_constraint.up.sql",
    "pkg/migrations/012_create_media_files.up.sql",
    "pkg/migrations/013_add_media_fields_to_songs.up.sql",
    # Phase 4: Realtime Engine Foundation
    "pkg/migrations/014_create_realtime_sessions.up.sql",
    "pkg/migrations/015_create_realtime_events.up.sql",
    "pkg/migrations/016_create_client_heartbeats.up.sql",
    "pkg/migrations/017_create_match_runtime_states.up.sql",
    "pkg/migrations/018_create_playback_schedules.up.sql",
    "pkg/migrations/019_create_scheduler_events.up.sql",
    # Phase 4.1: Realtime Hardening
    "pkg/migrations/020_add_version_to_match_runtime_states.up.sql",
    "pkg/migrations/021_add_version_to_realtime_sessions.up.sql",
    "pkg/migrations/022_add_duration_version_to_playback_schedules.up.sql",
    "pkg/migrations/023_create_idempotency_keys.up.sql",
    "pkg/migrations/024_add_processing_failed_to_scheduler_events.up.sql",
    # Phase 4.2.1: Production Hardening
    "pkg/migrations/025_add_match_id_fk_to_realtime_sessions.up.sql",
    "pkg/migrations/026_add_session_id_fk_to_scheduler_events.up.sql",
    "pkg/migrations/027_extend_realtime_events_event_type_constraint.up.sql",
    "pkg/migrations/028_add_created_at_index_to_client_heartbeats.up.sql",
    "pkg/migrations/029_add_created_at_index_to_realtime_events.up.sql",
    # Phase 4.3: Mobile API Contract
    "pkg/migrations/030_add_points_to_users.up.sql",
    "pkg/migrations/031_create_profiles.up.sql",
    "pkg/migrations/032_create_chants.up.sql",
    "pkg/migrations/033_create_quizzes.up.sql",
    "pkg/migrations/034_create_shop.up.sql",
    "pkg/migrations/035_create_news.up.sql",
    "pkg/migrations/036_create_videos.up.sql",
    "pkg/migrations/037_create_match_statistics.up.sql"
)

foreach ($migration in $migrations) {
    Write-Host "Running $migration..."
    docker cp $migration clap_postgres:/tmp/migration.sql
    docker exec clap_postgres psql -U postgres -d clap -f /tmp/migration.sql
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Migration $migration failed with exit code $LASTEXITCODE"
        exit $LASTEXITCODE
    }
}

Write-Host "All migrations completed successfully"
