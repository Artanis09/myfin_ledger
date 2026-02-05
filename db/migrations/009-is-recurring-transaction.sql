-- Add is_recurring flag to transactions for fixed expenses/income
ALTER TABLE transactions ADD COLUMN is_recurring INTEGER NOT NULL DEFAULT 0;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (009, '009-is-recurring-transaction');
