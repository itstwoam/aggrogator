-- name: MarkPostRead :exec
UPDATE posts
SET seen_at = $2
WHERE id = $1;
