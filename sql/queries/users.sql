-- name: CreateUser :one
INSERT INTO users (google_id, created_at, updated_at)
VALUES (
    $1,
    $2,
    $3
)
RETURNING id;

-- name: GetUserIdByGoogleId :one
SELECT id FROM users
WHERE google_id = $1;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1;

-- name: ContainsUserByGoogleId :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE google_id = $1
);

-- name: ContainsUserById :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE id = $1
);

-- name: DeleteUserById :one
DELETE FROM users WHERE id = $1
RETURNING *;

-- name: GetAllUsers :many
SELECT * FROM users;