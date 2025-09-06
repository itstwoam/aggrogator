-- name: GetRandomPosts :many
SELECT * FROM posts
ORDER BY (seen_at IS NOT NULL), seen_at ASC, RANDOM()
LIMIT $1;
