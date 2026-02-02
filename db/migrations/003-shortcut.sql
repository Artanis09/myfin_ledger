-- SMS 수신 로그 테이블 (아이폰 단축어로 전송된 메시지 저장)
CREATE TABLE IF NOT EXISTS sms_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    raw_text TEXT NOT NULL,
    parsed_card_name TEXT,
    parsed_amount INTEGER,
    parsed_description TEXT,
    parsed_date TEXT,
    transaction_id INTEGER,  -- 생성된 거래내역 ID (파싱 실패시 NULL)
    status TEXT NOT NULL DEFAULT 'received',  -- received, parsed, saved, error
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id)
);

-- API 키 테이블 (단축어 인증용)
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key_hash TEXT NOT NULL UNIQUE,  -- SHA256 해시
    name TEXT NOT NULL,  -- 키 이름 (예: "iPhone 단축어")
    is_active INTEGER NOT NULL DEFAULT 1,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (003, '003-shortcut');
