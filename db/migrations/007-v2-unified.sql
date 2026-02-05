-- ====================================================================
-- 가계부 통합 관리 시스템 v2.0 스키마 마이그레이션
-- ====================================================================

-- 1. 설정 테이블 (가계부 관리 기간 등)
CREATE TABLE IF NOT EXISTS app_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 기본 설정: 가계부 관리 기간 (start_day, end_day)
-- start_day <= end_day: 같은 달 (예: 1~31)
-- start_day > end_day: 월 경계 (예: 5일~익월 4일)
INSERT OR IGNORE INTO app_settings (key, value) VALUES 
    ('ledger_period_start_day', '1'),
    ('ledger_period_end_day', '31');

-- 2. 자산 유형 테이블 (지출용/수입용 구분)
CREATE TABLE IF NOT EXISTS asset_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('expense', 'income')),  -- 'expense' or 'income'
    is_card INTEGER NOT NULL DEFAULT 0,  -- 카드인 경우 1 (기존 cards 테이블 연동)
    card_id INTEGER DEFAULT NULL,  -- is_card=1인 경우 cards.id 참조
    is_recurring INTEGER NOT NULL DEFAULT 0,  -- 고정지출/고정수입 여부
    display_order INTEGER NOT NULL DEFAULT 0,
    is_system INTEGER NOT NULL DEFAULT 0,  -- 시스템 기본값 여부
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES cards(id)
);

-- 3. 수입 카테고리 테이블 (지출 카테고리는 기존 categories 테이블 사용)
CREATE TABLE IF NOT EXISTS income_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 기본 수입 카테고리
INSERT OR IGNORE INTO income_categories (name, display_order) VALUES 
    ('급여', 1),
    ('용돈', 2),
    ('부수입', 3),
    ('금융소득', 4),
    ('인센티브', 5),
    ('기타', 99);

-- 4. 거래내역 테이블 확장 (기존 transactions 테이블에 컬럼 추가)
-- 수입/지출 구분
ALTER TABLE transactions ADD COLUMN tx_type TEXT NOT NULL DEFAULT 'expense' CHECK(tx_type IN ('expense', 'income'));

-- 자산 유형 ID (card_id 대신 사용, 하위 호환성 위해 card_id 유지)
ALTER TABLE transactions ADD COLUMN asset_type_id INTEGER DEFAULT NULL;

-- 수입 카테고리 ID (tx_type='income'인 경우 사용)
ALTER TABLE transactions ADD COLUMN income_category_id INTEGER DEFAULT NULL;

-- 5. 반복 거래 스케줄 테이블
CREATE TABLE IF NOT EXISTS recurring_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tx_type TEXT NOT NULL CHECK(tx_type IN ('expense', 'income')),
    asset_type_id INTEGER NOT NULL,
    category_id INTEGER DEFAULT NULL,       -- 지출 카테고리 (tx_type='expense')
    income_category_id INTEGER DEFAULT NULL, -- 수입 카테고리 (tx_type='income')
    description TEXT NOT NULL,
    amount INTEGER NOT NULL,
    day_of_month INTEGER NOT NULL,  -- 매월 반복 일자 (1-31)
    memo TEXT DEFAULT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,  -- 활성 상태
    last_generated_date TEXT DEFAULT NULL, -- 마지막 자동 생성 날짜
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (asset_type_id) REFERENCES asset_types(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (income_category_id) REFERENCES income_categories(id)
);

-- 반복 거래로 생성된 거래 추적
ALTER TABLE transactions ADD COLUMN recurring_schedule_id INTEGER DEFAULT NULL;

-- 6. 기존 카드를 자산 유형으로 마이그레이션
-- 기존 cards 테이블의 각 카드를 asset_types에 추가
INSERT OR IGNORE INTO asset_types (name, type, is_card, card_id, is_system, display_order)
SELECT name, 'expense', 1, id, 1, id FROM cards;

-- 지출용 기본 자산 유형 추가
INSERT OR IGNORE INTO asset_types (name, type, is_card, is_recurring, is_system, display_order) VALUES 
    ('이체', 'expense', 0, 0, 1, 100),
    ('현금지출', 'expense', 0, 0, 1, 101),
    ('고정지출', 'expense', 0, 1, 1, 102);

-- 수입용 기본 자산 유형 추가
INSERT OR IGNORE INTO asset_types (name, type, is_card, is_recurring, is_system, display_order) VALUES 
    ('고정수입', 'income', 0, 1, 1, 200),
    ('부수입', 'income', 0, 0, 1, 201),
    ('임시', 'income', 0, 0, 1, 202),
    ('이월', 'income', 0, 0, 1, 203),
    ('기타', 'income', 0, 0, 1, 204);

-- 7. 기존 거래에 asset_type_id 설정 (card_id 기반)
UPDATE transactions 
SET asset_type_id = (
    SELECT at.id FROM asset_types at 
    WHERE at.card_id = transactions.card_id AND at.is_card = 1
)
WHERE asset_type_id IS NULL AND card_id IS NOT NULL;

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (007, '007-v2-unified');
