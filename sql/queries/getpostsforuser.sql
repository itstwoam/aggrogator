-- name: GetPostsForUser :many
SELECT * FROM posts
ORDER BY published_at DESC
LIMIT $1;
