-- SMS 파싱 키워드 매핑 룰 테이블
CREATE TABLE IF NOT EXISTS sms_mapping_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_type TEXT NOT NULL CHECK(rule_type IN ('card', 'income', 'expense')),
    keyword TEXT NOT NULL,
    asset_type_id INTEGER DEFAULT NULL,
    tx_type TEXT DEFAULT NULL CHECK(tx_type IN ('expense', 'income')),
    description TEXT DEFAULT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (asset_type_id) REFERENCES asset_types(id)
);

-- 기본 키워드 매핑 룰
-- 카드 인식 키워드 (승인, 누적 등)
INSERT OR IGNORE INTO sms_mapping_rules (rule_type, keyword, tx_type, priority, description) VALUES 
    ('card', '승인', 'expense', 100, '카드 승인 키워드'),
    ('card', '누적', 'expense', 90, '카드 누적 키워드'),
    ('card', '결제', 'expense', 80, '카드 결제 키워드');

-- 입금 인식 키워드
INSERT OR IGNORE INTO sms_mapping_rules (rule_type, keyword, tx_type, priority, description) VALUES 
    ('income', '입금', 'income', 100, '입금 키워드');

-- 출금/이체 인식 키워드  
INSERT OR IGNORE INTO sms_mapping_rules (rule_type, keyword, tx_type, priority, description) VALUES 
    ('expense', '출금', 'expense', 100, '출금 키워드'),
    ('expense', '이체', 'expense', 90, '이체 키워드'),
    ('expense', '이용', 'expense', 80, '이용 키워드');

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (010, '010-sms-mapping-rules');
