package handler

import (
	"github.com/Shubham-Hazra/blog-aggregator/internal/database"
	"github.com/Shubham-Hazra/blog-aggregator/pkg/types"
)

func middlewareLoggedIn(handler func(s *State, cmd types.Command, user database.User) error) func(*State, types.Command) error {
	return func (s *State, cmd types.Command) error {
		user, err := getCurrentUser(s)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}