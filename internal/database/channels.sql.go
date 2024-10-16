// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: channels.sql

package database

import (
	"context"
)

const containsChannelInDB = `-- name: ContainsChannelInDB :one
SELECT EXISTS (
    SELECT 1 FROM channels
    WHERE channel_handle = $1
)
`

func (q *Queries) ContainsChannelInDB(ctx context.Context, channelHandle string) (bool, error) {
	row := q.db.QueryRowContext(ctx, containsChannelInDB, channelHandle)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const deleteChannel = `-- name: DeleteChannel :exec
DELETE FROM channels
WHERE channel_id = $1
`

func (q *Queries) DeleteChannel(ctx context.Context, channelID string) error {
	_, err := q.db.ExecContext(ctx, deleteChannel, channelID)
	return err
}

const getChannelHandle = `-- name: GetChannelHandle :one
SELECT channel_handle FROM channels
WHERE channel_id = $1
`

func (q *Queries) GetChannelHandle(ctx context.Context, channelID string) (string, error) {
	row := q.db.QueryRowContext(ctx, getChannelHandle, channelID)
	var channel_handle string
	err := row.Scan(&channel_handle)
	return channel_handle, err
}

const getChannelHandleUploadId = `-- name: GetChannelHandleUploadId :one
SELECT channel_handle, channel_upload_id FROM channels
WHERE channel_id = $1
`

type GetChannelHandleUploadIdRow struct {
	ChannelHandle   string
	ChannelUploadID string
}

func (q *Queries) GetChannelHandleUploadId(ctx context.Context, channelID string) (GetChannelHandleUploadIdRow, error) {
	row := q.db.QueryRowContext(ctx, getChannelHandleUploadId, channelID)
	var i GetChannelHandleUploadIdRow
	err := row.Scan(&i.ChannelHandle, &i.ChannelUploadID)
	return i, err
}

const getChannelIdByHandle = `-- name: GetChannelIdByHandle :one
SELECT channel_id FROM channels
WHERE channel_handle = $1
`

func (q *Queries) GetChannelIdByHandle(ctx context.Context, channelHandle string) (string, error) {
	row := q.db.QueryRowContext(ctx, getChannelIdByHandle, channelHandle)
	var channel_id string
	err := row.Scan(&channel_id)
	return channel_id, err
}

const getChannelIdUploadIdByHandle = `-- name: GetChannelIdUploadIdByHandle :one
SELECT channel_id, channel_upload_id FROM channels
WHERE channel_handle = $1
`

type GetChannelIdUploadIdByHandleRow struct {
	ChannelID       string
	ChannelUploadID string
}

func (q *Queries) GetChannelIdUploadIdByHandle(ctx context.Context, channelHandle string) (GetChannelIdUploadIdByHandleRow, error) {
	row := q.db.QueryRowContext(ctx, getChannelIdUploadIdByHandle, channelHandle)
	var i GetChannelIdUploadIdByHandleRow
	err := row.Scan(&i.ChannelID, &i.ChannelUploadID)
	return i, err
}

const getUploadId = `-- name: GetUploadId :one
SELECT channel_upload_id FROM channels
WHERE channel_id = $1
`

func (q *Queries) GetUploadId(ctx context.Context, channelID string) (string, error) {
	row := q.db.QueryRowContext(ctx, getUploadId, channelID)
	var channel_upload_id string
	err := row.Scan(&channel_upload_id)
	return channel_upload_id, err
}

const insertChannel = `-- name: InsertChannel :one
INSERT INTO channels (channel_id, channel_upload_id, channel_handle, channel_url)
VALUES(
    $1,
    $2,
    $3,
    $4
)
RETURNING channel_id, channel_upload_id, channel_handle, channel_url
`

type InsertChannelParams struct {
	ChannelID       string
	ChannelUploadID string
	ChannelHandle   string
	ChannelUrl      string
}

func (q *Queries) InsertChannel(ctx context.Context, arg InsertChannelParams) (Channel, error) {
	row := q.db.QueryRowContext(ctx, insertChannel,
		arg.ChannelID,
		arg.ChannelUploadID,
		arg.ChannelHandle,
		arg.ChannelUrl,
	)
	var i Channel
	err := row.Scan(
		&i.ChannelID,
		&i.ChannelUploadID,
		&i.ChannelHandle,
		&i.ChannelUrl,
	)
	return i, err
}
