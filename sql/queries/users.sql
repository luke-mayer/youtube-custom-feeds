-- name: CreateUser :exec
INSERT INTO users (fb_user_id, created_at, updated_at)
VALUES (
    $1,
    $2,
    $3
);

-- name: GetUserById :one
SELECT * FROM users
WHERE fb_user_id = $1;

-- name: ContainsUserByFirebaseId :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE fb_user_id = $1
);

-- name: DeleteUserById :exec
DELETE FROM users WHERE fb_user_id = $1;

-- name: GetAllUsers :many
SELECT * FROM users;
