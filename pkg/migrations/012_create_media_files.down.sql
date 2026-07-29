DROP TRIGGER IF EXISTS update_media_files_updated_at ON media_files;
DROP INDEX IF EXISTS idx_media_files_deleted_at;
DROP INDEX IF EXISTS idx_media_files_uploaded_by;
DROP INDEX IF EXISTS idx_media_files_checksum;
DROP INDEX IF EXISTS idx_media_files_storage_key;
DROP TABLE IF EXISTS media_files;
