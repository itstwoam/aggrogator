-- +goose UP
CREATE TABLE feed_follows(
	id UUID PRIMARY KEY NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	user_id UUID NOT NULL,
	feed_id UUID NOT NULL,
	CONSTRAINT fk_feed_id
	FOREIGN KEY (feed_id)
	REFERENCES feeds(id) ON DELETE CASCADE,
	CONSTRAINT fk_user_id
	FOREIGN KEY (user_id)
	REFERENCES users(id) ON DELETE CASCADE,
	UNIQUE (user_id, feed_id)
);

-- +goose Down
DROP TABLE feeds;
