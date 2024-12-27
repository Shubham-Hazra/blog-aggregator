-- name: CreateFeed :one
INSERT INTO feeds (name, url, user_id, created_at, updated_at)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetFeedFromUrl :one
SELECT * 
FROM feeds
WHERE url = $1;

-- name: GetFeeds :many
SELECT 
    feeds.name AS feed_name,
    feeds.url AS feed_url,
    users.name AS user_name
FROM 
    feeds
JOIN 
    users
ON 
    feeds.user_id = users.id;

