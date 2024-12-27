-- name: ResetTables :exec
TRUNCATE TABLE users, feeds, feed_follows CASCADE;