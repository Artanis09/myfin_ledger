-- ====================================================================
-- card_id를 nullable로 변경 (현금지출 등 카드 없는 자산 유형 지원)
-- SQLite는 ALTER COLUMN을 지원하지 않으므로 테이블 재생성 필요
-- ====================================================================

-- 1. 임시 테이블 생성
CREATE TABLE transactions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER DEFAULT NULL,  -- NULL 허용으로 변경
    category_id INTEGER,
    transaction_date TIMESTAMP NOT NULL,
    description TEXT NOT NULL,
    amount INTEGER NOT NULL,
    is_installment INTEGER NOT NULL DEFAULT 0,
    installment_months INTEGER DEFAULT NULL,
    installment_current INTEGER DEFAULT NULL,
    original_amount INTEGER DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_cancelled INTEGER NOT NULL DEFAULT 0,
    memo TEXT DEFAULT NULL,
    tx_type TEXT NOT NULL DEFAULT 'expense' CHECK(tx_type IN ('expense', 'income')),
    asset_type_id INTEGER DEFAULT NULL,
    income_category_id INTEGER DEFAULT NULL,
    recurring_schedule_id INTEGER DEFAULT NULL,
    FOREIGN KEY (card_id) REFERENCES cards(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- 2. 기존 데이터 복사
INSERT INTO transactions_new 
SELECT * FROM transactions;

-- 3. 기존 테이블 삭제
DROP TABLE transactions;

-- 4. 새 테이블 이름 변경
ALTER TABLE transactions_new RENAME TO transactions;

-- 5. 인덱스 재생성 (필요 시)
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_card ON transactions(card_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(tx_type);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (008, '008-fix-card-id-nullable');
