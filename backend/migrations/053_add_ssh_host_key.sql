-- Add SSH host key column for Trust-On-First-Use (TOFU) host key verification.
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS ssh_host_key TEXT DEFAULT '';
