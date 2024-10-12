-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE feeds (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES users(id)
);

CREATE TABLE channels (
    channel_id TEXT PRIMARY KEY,
    channel_url TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE feeds_channels (
    feed_id INTEGER NOT NULL,
    channel_id TEXT,
    PRIMARY KEY (feed_id, channel_id),
    FOREIGN KEY (feed_id) REFERENCES feeds(id),
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id)
);

-- +goose Down
DROP TABLE feeds_channels;

DROP TABLE channels;

DROP TABLE feeds;

DROP TABLE users;