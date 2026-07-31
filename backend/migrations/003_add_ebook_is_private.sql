ALTER TABLE ebooks ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT true;
CREATE INDEX IF NOT EXISTS idx_ebooks_visibility ON ebooks(is_private, uploaded_by);
