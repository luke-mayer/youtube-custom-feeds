-- name: CreateFeed :one
INSERT INTO feeds(created_at, updated_at, name, user_id)
VALUES(
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetAllUserFeeds :many
SELECT id, name FROM feeds
WHERE user_id = $1;

-- name: GetAllUserFeedNames :many
SELECT name FROM feeds
WHERE user_id = $1;

-- name: GetFeedId :one
SELECT id FROM feeds
WHERE user_id = $1 AND name = $2;

-- name: DeleteAllFeeds :exec
DELETE FROM feeds
WHERE user_id = $1
RETURNING *;

-- name: DeleteFeed :exec
DELETE FROM feeds
WHERE user_id = $1 AND name = $2
RETURNING *;

-- name: ContainsFeed :one
SELECT EXISTS (
    SELECT 1 FROM feeds
    WHERE user_id = $1 AND name = $2
);

-- name: UpdateName :one
UPDATE feeds
SET name = $2, updated_at = $3
WHERE user_id = $1
RETURNING *;