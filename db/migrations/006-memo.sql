-- Add memo column to transactions
ALTER TABLE transactions ADD COLUMN memo TEXT DEFAULT NULL;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (006, '006-memo');
