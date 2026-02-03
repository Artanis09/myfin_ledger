-- 월별 소비 목표 테이블
CREATE TABLE IF NOT EXISTS monthly_goals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    target_amount INTEGER NOT NULL,  -- 목표 금액 (원)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(year, month)
);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (004, '004-goals');
