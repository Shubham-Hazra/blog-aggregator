// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: tables.sql

package database

import (
	"context"
)

const resetTables = `-- name: ResetTables :exec
TRUNCATE TABLE users, feeds, feed_follows CASCADE
`

func (q *Queries) ResetTables(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, resetTables)
	return err
}
