-- 취소 거래 지원을 위한 is_cancelled 필드 추가
ALTER TABLE transactions ADD COLUMN is_cancelled INTEGER NOT NULL DEFAULT 0;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (005, '005-cancel');
