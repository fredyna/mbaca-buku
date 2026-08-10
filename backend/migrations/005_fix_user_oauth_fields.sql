-- Migration 004 added the OAuth columns as plain nullable text, so every row
-- that predates it holds NULL. UserRepository scans those columns into plain
-- Go strings, and lib/pq cannot convert NULL to string — which made GetByEmail
-- and GetByID fail for every pre-existing user, breaking password login and
-- /auth/me alike. Backfill the NULLs and forbid them from here on.
UPDATE users SET provider = '' WHERE provider IS NULL;
UPDATE users SET provider_id = '' WHERE provider_id IS NULL;
UPDATE users SET avatar_url = '' WHERE avatar_url IS NULL;

ALTER TABLE users
    ALTER COLUMN provider SET DEFAULT '',
    ALTER COLUMN provider SET NOT NULL,
    ALTER COLUMN provider_id SET DEFAULT '',
    ALTER COLUMN provider_id SET NOT NULL,
    ALTER COLUMN avatar_url SET DEFAULT '',
    ALTER COLUMN avatar_url SET NOT NULL;

-- One account per provider identity. Partial so the empty string shared by all
-- password accounts does not collide.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_identity
    ON users (provider, provider_id)
    WHERE provider <> '' AND provider_id <> '';
