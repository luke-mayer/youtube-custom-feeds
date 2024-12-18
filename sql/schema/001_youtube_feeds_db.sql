-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    fb_user_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE feeds (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES users(id)
                ON DELETE CASCADE,
    CONSTRAINT unique_name_user
        UNIQUE(name, user_id)
);

CREATE TABLE channels (
    channel_id VARCHAR(255) PRIMARY KEY,
    channel_upload_id VARCHAR(255) UNIQUE NOT NULL,
    channel_handle VARCHAR(255) UNIQUE NOT NULL,
    channel_url TEXT UNIQUE NOT NULL
);

CREATE TABLE feeds_channels (
    feed_id INTEGER NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (feed_id, channel_id),
    FOREIGN KEY (feed_id) REFERENCES feeds(id),
    FOREIGN KEY (channel_id) REFERENCES channels(channel_id)
);

-- +goose Down
DROP TABLE feeds_channels;

DROP TABLE channels;

DROP TABLE feeds;

DROP TABLE users;