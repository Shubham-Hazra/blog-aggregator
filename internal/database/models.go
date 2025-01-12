// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Feed struct {
	ID            int32
	Name          string
	Url           string
	UserID        uuid.UUID
	CreatedAt     sql.NullTime
	UpdatedAt     sql.NullTime
	LastFetchedAt sql.NullTime
}

type FeedFollow struct {
	ID        int32
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
	UserID    uuid.UUID
	FeedID    int32
}

type Post struct {
	ID          int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Title       string
	Url         string
	Description sql.NullString
	PublishedAt sql.NullTime
	FeedID      int32
}

type User struct {
	ID        uuid.UUID
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
	Name      string
}
