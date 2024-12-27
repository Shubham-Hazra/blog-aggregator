package handler

import (
	"github.com/Shubham-Hazra/blog-aggregator/internal/config"
	"github.com/Shubham-Hazra/blog-aggregator/internal/database"
)

type State struct {
	Config    *config.Config
	DBQueries *database.Queries
}

func NewState(config *config.Config, queries *database.Queries) *State {
	return &State{
		Config:    config,
		DBQueries: queries,
	}
}