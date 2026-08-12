-- Every statement here is IF NOT EXISTS because RunMigrations re-executes every
-- file in this directory on each server start; there is no version table.
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event      VARCHAR(20) NOT NULL,              -- 'login' | 'active'
    -- Text columns are NOT NULL DEFAULT '' so lib/pq can scan them into plain
    -- Go strings; see migration 005 for what NULL here costs.
    os         VARCHAR(50)  NOT NULL DEFAULT '',
    browser    VARCHAR(50)  NOT NULL DEFAULT '',
    device     VARCHAR(20)  NOT NULL DEFAULT '',  -- desktop | mobile | tablet
    ip_address VARCHAR(45)  NOT NULL DEFAULT '',  -- 45 = longest possible IPv6 text form
    user_agent TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Serves "latest activity for this user" in the admin list.
CREATE INDEX IF NOT EXISTS idx_activity_user_created
    ON user_activity_logs(user_id, created_at DESC);

-- Serves "latest login for this user" in the same query.
CREATE INDEX IF NOT EXISTS idx_activity_user_event
    ON user_activity_logs(user_id, event, created_at DESC);

-- Serves the retention sweep.
CREATE INDEX IF NOT EXISTS idx_activity_created
    ON user_activity_logs(created_at);
