-- +goose UP
CREATE TABLE posts(
	id UUID PRIMARY KEY NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	seen_at TIMESTAMP,
	title TEXT,
	url TEXT UNIQUE NOT NULL,
	description TEXT,
	content TEXT,
	published_at TIMESTAMP NOT NULL,
	feed_id UUID NOT NULL,
	CONSTRAINT fk_feed_id
	FOREIGN KEY (feed_id)
	REFERENCES feeds(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE posts;
