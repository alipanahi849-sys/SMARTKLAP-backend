# Seed default roles into database

Write-Host "Seeding default roles..."

$sql = @"
INSERT INTO roles (name, description) VALUES
    ('admin', 'System administrator with full access'),
    ('club_admin', 'Club administrator with limited administrative access'),
    ('moderator', 'Content moderator with limited administrative access'),
    ('user', 'Regular user with standard access')
ON CONFLICT (name) DO NOTHING;
"@

# Write SQL to temp file
$sql | Out-File -Encoding UTF8 -FilePath "tmp_seed.sql"

# Copy to container
docker cp tmp_seed.sql clap_postgres:/tmp/seed.sql

# Run seed inside container
docker exec clap_postgres psql -U postgres -d clap -f /tmp/seed.sql

# Cleanup
Remove-Item tmp_seed.sql

if ($LASTEXITCODE -eq 0) {
    Write-Host "Default roles seeded successfully"
} else {
    Write-Host "Seeding failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}
