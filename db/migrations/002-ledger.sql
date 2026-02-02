-- 가계부 테이블

-- 카드사 테이블
CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    billing_start_day INTEGER NOT NULL DEFAULT 1,  -- 결산 시작일 (예: 22)
    billing_end_day INTEGER NOT NULL DEFAULT 31,   -- 결산 종료일 (예: 21, 다음달)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 카테고리 테이블
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    keywords TEXT NOT NULL DEFAULT '',  -- 쉼표로 구분된 키워드 목록
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 거래내역 테이블
CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    category_id INTEGER,
    transaction_date TIMESTAMP NOT NULL,
    description TEXT NOT NULL,
    amount INTEGER NOT NULL,  -- 원 단위
    is_installment INTEGER NOT NULL DEFAULT 0,
    installment_months INTEGER DEFAULT NULL,  -- 총 할부 개월수
    installment_current INTEGER DEFAULT NULL,  -- 현재 회차
    original_amount INTEGER DEFAULT NULL,  -- 할부 원금 총액
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES cards(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- 기본 카테고리 삽입
INSERT OR IGNORE INTO categories (name, keywords) VALUES 
    ('쇼핑', '쿠팡,11번가,G마켓,옥션,위메프,티몬'),
    ('식비', '배달의민족,요기요,쿠팡이츠,식당,레스토랑'),
    ('의료', '병원,약국,의원,클리닉'),
    ('교통', '주유소,주차장,택시,버스,지하철'),
    ('편의점', 'CU,GS25,세븐일레븐,이마트24,미니스톱'),
    ('카페', '스타벅스,이디야,투썸,커피빈,메가커피'),
    ('기타', '');

-- 기본 카드사 삽입
INSERT OR IGNORE INTO cards (name, billing_start_day, billing_end_day) VALUES
    ('삼성카드', 1, 31),
    ('신한카드', 1, 31),
    ('현대카드', 1, 31),
    ('KB국민카드', 1, 31),
    ('롯데카드', 1, 31),
    ('하나카드', 1, 31),
    ('우리카드', 1, 31),
    ('NH농협카드', 1, 31),
    ('BC카드', 1, 31);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (002, '002-ledger');
