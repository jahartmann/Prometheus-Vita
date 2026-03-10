ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;

-- Flag any user still using default password pattern
UPDATE users SET must_change_password = true WHERE username = 'admin' AND last_login IS NULL;
