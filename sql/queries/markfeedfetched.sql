-- name: MarkFeedFetched :execrows
UPDATE feeds
SET updated_at = $2,
last_fetched_at = $2
WHERE id = $1;
