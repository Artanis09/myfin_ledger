-- name: GetMonthlyGoal :one
SELECT * FROM monthly_goals WHERE year = ? AND month = ?;

-- name: SetMonthlyGoal :one
INSERT INTO monthly_goals (year, month, target_amount)
VALUES (?, ?, ?)
ON CONFLICT(year, month) DO UPDATE SET target_amount = excluded.target_amount
RETURNING *;

-- name: GetAllMonthlyGoals :many
SELECT * FROM monthly_goals ORDER BY year DESC, month DESC;
