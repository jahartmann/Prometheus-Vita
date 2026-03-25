ALTER TABLE config_backup_files ADD COLUMN IF NOT EXISTS is_encrypted BOOLEAN DEFAULT false;
