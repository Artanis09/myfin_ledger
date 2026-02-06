-- name: GetSMSMappingRules :many
SELECT * FROM sms_mapping_rules WHERE is_active = 1 ORDER BY priority DESC;

-- name: GetSMSMappingRulesByType :many
SELECT * FROM sms_mapping_rules WHERE rule_type = ? AND is_active = 1 ORDER BY priority DESC;

-- name: CreateSMSMappingRule :one
INSERT INTO sms_mapping_rules (rule_type, keyword, asset_type_id, tx_type, description, priority)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateSMSMappingRule :exec
UPDATE sms_mapping_rules SET keyword = ?, asset_type_id = ?, tx_type = ?, description = ?, priority = ? WHERE id = ?;

-- name: DeleteSMSMappingRule :exec
DELETE FROM sms_mapping_rules WHERE id = ?;

-- name: GetAllCardsWithKeywords :many
SELECT c.id, c.name, at.id as asset_type_id FROM cards c
LEFT JOIN asset_types at ON at.card_id = c.id AND at.is_card = 1;
