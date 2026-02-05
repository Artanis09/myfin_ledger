-- name: GetAllCards :many
SELECT * FROM cards ORDER BY name;

-- name: GetCard :one
SELECT * FROM cards WHERE id = ?;

-- name: CreateCard :one
INSERT INTO cards (name, billing_start_day, billing_end_day) 
VALUES (?, ?, ?)
RETURNING *;

-- name: UpdateCard :exec
UPDATE cards SET name = ?, billing_start_day = ?, billing_end_day = ? WHERE id = ?;

-- name: DeleteCard :exec
DELETE FROM cards WHERE id = ?;

-- name: GetAllCategories :many
SELECT * FROM categories ORDER BY name;

-- name: GetCategory :one
SELECT * FROM categories WHERE id = ?;

-- name: CreateCategory :one
INSERT INTO categories (name, keywords) VALUES (?, ?)
RETURNING *;

-- name: UpdateCategory :exec
UPDATE categories SET name = ?, keywords = ? WHERE id = ?;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: CreateTransaction :one
INSERT INTO transactions (
    card_id, category_id, transaction_date, description, amount,
    is_installment, installment_months, installment_current, original_amount, is_cancelled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = ?;

-- name: UpdateTransaction :exec
UPDATE transactions SET 
    card_id = ?, category_id = ?, transaction_date = ?, description = ?,
    amount = ?, is_installment = ?, installment_months = ?, 
    installment_current = ?, original_amount = ?, is_cancelled = ?
WHERE id = ?;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = ?;

-- name: GetTransactionsByDateRange :many
SELECT t.*, c.name as card_name, cat.name as category_name
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE t.transaction_date >= ? AND t.transaction_date < ?
ORDER BY t.transaction_date DESC;

-- name: GetTransactionsByCardAndDateRange :many
SELECT t.*, c.name as card_name, cat.name as category_name
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE t.card_id = ? AND t.transaction_date >= ? AND t.transaction_date < ?
ORDER BY t.transaction_date DESC;

-- name: GetAllTransactions :many
SELECT t.*, c.name as card_name, cat.name as category_name
FROM transactions t
JOIN cards c ON t.card_id = c.id
LEFT JOIN categories cat ON t.category_id = cat.id
ORDER BY t.transaction_date DESC;

-- name: GetSumByCategory :many
SELECT 
    COALESCE(cat.name, '미분류') as category_name,
    SUM(t.amount) as total_amount,
    COUNT(*) as count
FROM transactions t
LEFT JOIN categories cat ON t.category_id = cat.id
WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) < ?
GROUP BY t.category_id
ORDER BY total_amount DESC;

-- name: GetSumByCard :many
SELECT 
    c.name as card_name,
    c.id as card_id,
    SUM(t.amount) as total_amount,
    COUNT(*) as count
FROM transactions t
JOIN cards c ON t.card_id = c.id
WHERE substr(t.transaction_date, 1, 10) >= ? AND substr(t.transaction_date, 1, 10) < ?
GROUP BY t.card_id
ORDER BY total_amount DESC;
