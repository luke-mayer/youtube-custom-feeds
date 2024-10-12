-- name: InsertFeedChannel :exec
INSERT INTO feeds_channels (feed_id, channel_id) 
VALUES(
    $1,
    $2
);

-- name: DeleteFeedChannel :exec
DELETE FROM feeds_channels 
WHERE feed_id = $1 AND channel_id = $2;

-- name: ContainsFeedChannel :one
SELECT EXISTS (
    SELECT 1 FROM feeds_channels
    WHERE feed_id = $1 AND channel_id = $2
);

-- name: ContainsChannel :one
SELECT EXISTS (
    SELECT 1 FROM feeds_channels
    WHERE channel_id = $1
);

-- name: GetAllFeedChannels :many
SELECT channel_id FROM feeds_channels
WHERE feed_id = $1;

-- name: DeleteAllFeedChannels :exec
DELETE FROM feeds_channels
WHERE feed_id = $1;