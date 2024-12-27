package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Shubham-Hazra/blog-aggregator/internal/config"
	"github.com/Shubham-Hazra/blog-aggregator/internal/database"
	"github.com/Shubham-Hazra/blog-aggregator/internal/handler"
	"github.com/Shubham-Hazra/blog-aggregator/pkg/types"
	_ "github.com/lib/pq"
)

func main() {
    config := &config.Config{}
    config.Read()
    
    db, err := sql.Open("postgres", config.DB_URL)
    if err != nil {
        log.Fatal(err)
    }
    dbQueries := database.New(db)
    
    state := handler.NewState(config, dbQueries)
    cmdHandler := handler.NewHandler(state)

    if err := executeCommand(cmdHandler); err != nil {
        log.Fatal(err)
    }
}

func executeCommand(ch *handler.Handler) error {
    args := os.Args
    if len(args) < 2 {
        return fmt.Errorf("too few arguments")
    }
    
    cmd := types.Command{
        Name: args[1],
        Args: args[2:],
    }
    
    return ch.Execute(cmd)
}