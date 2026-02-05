-- 카테고리 자동 매핑 테이블
-- 사용자가 특정 사용내역(description)의 카테고리를 변경하면
-- 해당 description에 대해 카테고리를 기억하여 다음에 자동 적용
CREATE TABLE IF NOT EXISTS category_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT NOT NULL,
    normalized_description TEXT NOT NULL,  -- 정규화된 description (공백/특수문자 제거, 소문자)
    category_id INTEGER NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 1,  -- 사용 횟수
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id),
    UNIQUE(normalized_description)
);

-- 빠른 조회를 위한 인덱스
CREATE INDEX IF NOT EXISTS idx_category_mappings_normalized ON category_mappings(normalized_description);
