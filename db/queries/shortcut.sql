-- name: CreateSMSLog :one
INSERT INTO sms_logs (
    raw_text, parsed_card_name, parsed_amount, parsed_description,
    parsed_date, transaction_id, status, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSMSLogs :many
SELECT * FROM sms_logs ORDER BY created_at DESC LIMIT ?;

-- name: UpdateSMSLogStatus :exec
UPDATE sms_logs SET status = ?, transaction_id = ?, error_message = ? WHERE id = ?;

-- name: CreateAPIKey :one
INSERT INTO api_keys (key_hash, name) VALUES (?, ?)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = ? AND is_active = 1;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: GetAllAPIKeys :many
SELECT id, name, is_active, last_used_at, created_at FROM api_keys ORDER BY created_at DESC;

-- name: DeactivateAPIKey :exec
UPDATE api_keys SET is_active = 0 WHERE id = ?;

-- name: GetCardByName :one
SELECT * FROM cards WHERE name = ?;
