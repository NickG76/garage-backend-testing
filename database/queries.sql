-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, phone, is_admin)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetAllUsers :many
SELECT * FROM users ORDER BY name;

-- name: CreateAppointment :one
INSERT INTO appointments (user_id, datetime, title, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UserCancelAppointment :exec
UPDATE appointments SET status = 'cancelled' WHERE id = $1 AND user_id = $2;

-- name: AdminDeleteAppointment :exec
DELETE FROM appointments WHERE id = $1;

-- name: GetAppointmentsForUser :many
SELECT * FROM appointments WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateAppointmentStatus :exec
UPDATE appointments SET status = $2 WHERE id = $1;

-- name: GetAllAppointments :many
SELECT
  a.id,
  a.user_id,
  a.datetime,
  a.title,
  a.description,
  a.status,
  a.created_at,
  u.name AS user_name,
  u.email AS user_email,
  u.phone AS user_phone
FROM appointments a
JOIN users u ON a.user_id = u.id
ORDER BY a.created_at DESC;

-- name: GetAppointmentsByID :one
SELECT * FROM appointments WHERE id = $1;

-- name: GetAllAppointmentsByMonth :many
SELECT * FROM appointments WHERE datetime >= $1 AND datetime < $2 ORDER BY datetime;


