-- name: InsertChannel :one
INSERT INTO channels (channel_id, channel_url, name)
VALUES(
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetChannelNameUrl :one
SELECT channel_url, name FROM channels
WHERE channel_id = $1;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE channel_id = $1
RETURNING *;