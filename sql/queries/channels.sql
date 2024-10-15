-- name: InsertChannel :one
INSERT INTO channels (channel_id, channel_upload_id, channel_handle, channel_url)
VALUES(
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetChannelHandleUploadId :one
SELECT channel_handle, channel_upload_id FROM channels
WHERE channel_id = $1;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE channel_id = $1;

-- name: GetChannelIdUploadIdByHandle :one
SELECT channel_id, channel_upload_id FROM channels
WHERE channel_handle = $1;

-- name: ContainsChannelInDB :one
SELECT EXISTS (
    SELECT 1 FROM channels
    WHERE channel_handle = $1
);
