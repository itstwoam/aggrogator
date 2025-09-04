-- name: DeleteFeedFollow :execrows
DELETE FROM feed_follows
Where feed_follows.user_id = $1
AND feed_follows.feed_id IN ( 
	SELECT id FROM feeds WHERE url = $2
);
