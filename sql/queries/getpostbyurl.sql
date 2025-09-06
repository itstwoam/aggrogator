-- name: GetPostByURL :one
SELECT * FROM posts
WHERE url = $1;
