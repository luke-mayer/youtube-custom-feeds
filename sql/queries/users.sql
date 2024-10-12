-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUserName :one
SELECT * FROM users
WHERE name = $1;

-- name: GetUserId :one
SELECT * FROM users
WHERE id = $1;

-- name: ContainsUser :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE name = $1
);

-- name: ContainsUserById :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE id = $1
);

-- name: DeleteUserName :one
DELETE FROM users WHERE name = $1
RETURNING *;

-- name: DeleteUserID :one
DELETE FROM users WHERE id = $1
RETURNING *;

-- name: UpdateUserName :one
UPDATE users
SET name = $2, updated_at = $3
WHERE id = $1
RETURNING *;

-- name: GetAllUsers :many
SELECT * FROM users;