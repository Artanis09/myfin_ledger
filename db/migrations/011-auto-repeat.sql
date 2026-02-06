-- Add auto repeat fields to transactions for recurring transactions
-- is_auto_repeat: 1이면 매달 자동으로 같은 내역이 생성됨
-- repeat_day: 매달 반복될 날짜 (1-31, NULL이면 원본 거래 날짜 사용)
-- parent_transaction_id: 자동 생성된 경우 원본 거래 ID
-- last_repeat_date: 마지막으로 반복 생성된 날짜

ALTER TABLE transactions ADD COLUMN is_auto_repeat INTEGER NOT NULL DEFAULT 0;
ALTER TABLE transactions ADD COLUMN repeat_day INTEGER DEFAULT NULL;
ALTER TABLE transactions ADD COLUMN parent_transaction_id INTEGER DEFAULT NULL;
ALTER TABLE transactions ADD COLUMN last_repeat_date TEXT DEFAULT NULL;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (011, '011-auto-repeat');
