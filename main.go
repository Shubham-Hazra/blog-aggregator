package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shubham-Hazra/blog-aggregator/internal/config"
	"github.com/Shubham-Hazra/blog-aggregator/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req , err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil , err
	}
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil , err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil , err
	}
	var rssFeed RSSFeed
	err = xml.Unmarshal(data, &rssFeed)
	if err != nil {
		return nil , err
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}

	return &rssFeed, nil
}

func handlerAgg(s *state, cmd command) error {
	url := "https://www.wagslane.dev/index.xml"
	ctx := context.Background()

	rssFeed, err := fetchFeed(ctx, url)
	if err != nil {
		return err
	}

	fmt.Println("Channel Title: " + rssFeed.Channel.Title)
	fmt.Println("Channel Description: " + rssFeed.Channel.Description)
	fmt.Println("Channel Link: " + rssFeed.Channel.Link)
	fmt.Println("Items:")
	for _, item := range rssFeed.Channel.Item {
		fmt.Println(strings.Repeat("#", 50))
		fmt.Println("Title: " + item.Title)
		fmt.Println("Description: " + item.Description)
		fmt.Println("Link: " + item.Link)
		fmt.Println("PubDate: " + item.PubDate)
		fmt.Println(strings.Repeat("#", 50))
	}
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.dbQueries.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, item := range feeds{
		fmt.Println(strings.Repeat("#", 50))
		fmt.Println("Feed Name: " + item.FeedName)
		fmt.Println("Feed URL: " + item.FeedUrl)
		fmt.Println("User Name: " + item.UserName)
		fmt.Println(strings.Repeat("#", 50))
	}
	return nil
}

type state struct {
	config *config.Config
	dbQueries *database.Queries
}

type command struct{
	name string 
	args []string
}

type commands struct{
	commandMap map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandMap[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	if _, ok := c.commandMap[cmd.name]; !ok {
		return errors.New("command " + cmd.name + " is not recognized")
	}
	err := c.commandMap[cmd.name](s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("login accepts only one argument")
	}

	_, err := s.dbQueries.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		fmt.Println("Error: User " + cmd.args[0] + " is not registered")
		os.Exit(1) 
	} 

	err = s.config.SetUser(cmd.args[0])
	if err != nil {
		return fmt.Errorf("error encountered while login: %w", err)
	}

	fmt.Println("User logged in successfully")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("register accepts only one argument")
	}

	nullTime := sql.NullTime{
		Time:  time.Now(),
		Valid: true, 
	}

	existingUser, err := s.dbQueries.GetUser(context.Background(), cmd.args[0])
	if err == nil && existingUser.Name == cmd.args[0] {
		fmt.Println("Error: A user with that name already exists.")
		log.Printf("Attempt to register an existing user: %v\n", cmd.args[0])
		os.Exit(1) // Exit with code 1
	} 

	user, err := s.dbQueries.CreateUser(context.Background(), database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
		Name: cmd.args[0],
	}) 
	if err != nil {
		return err
	}
	s.config.SetUser(cmd.args[0])
	fmt.Println("User " + cmd.args[0] + " has been successfully registered")
	log.Printf("id: %v, created_at: %v, updated_at: %v, name: %v\n",user.ID,user.CreatedAt, user.UpdatedAt, user.Name)
	
	return nil
}

func handlerAddfeed(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return errors.New("addfeed accepts exactly 2 arguments")
	}

	feedName := cmd.args[0]
	feedURL := cmd.args[1]

	existingUser, err := s.dbQueries.GetUser(context.Background(), s.config.CURRENT_USER_NAME)
	if err != nil {
		fmt.Println("Error: Could not retrieve user details from the database")
		os.Exit(1) // Exit with code 1
	} 

	nullTime := sql.NullTime{
		Time:  time.Now(),
		Valid: true, 
	}

	feed, err := s.dbQueries.CreateFeed(context.Background(), database.CreateFeedParams{
		Name: feedName,
		Url: feedURL,
		UserID: existingUser.ID,
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
	})
	if err != nil {
		fmt.Println("Error: Could not create feed")
		os.Exit(1) // Exit with code 1
	} 

	_, err = s.dbQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
		UserID: existingUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("Error: Could not create feed follow")
		os.Exit(1) // Exit with code 1
	} 

	log.Printf("id: %v, name: %v, url: %v, user_id: %v, created_at: %v, updated_at: %v", 
	feed.ID, 
		feed.Name, 
		feed.Url, 
		feed.UserID, 
		feed.CreatedAt, 
		feed.UpdatedAt)
	return nil
}

func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("follow accepts exactly 1 argument")
	}

	feedURL := cmd.args[0]

	existingUser, err := s.dbQueries.GetUser(context.Background(), s.config.CURRENT_USER_NAME)
	if err != nil {
		fmt.Println("Error: Could not retrieve user details from the database")
		os.Exit(1) 
	}

	nullTime := sql.NullTime{
		Time:  time.Now(),
		Valid: true, 
	}

	existingFeed, err := s.dbQueries.GetFeedFromUrl(context.Background(), feedURL)
	if err != nil {
		fmt.Println("Error: Could not retrieve feed details from the database")
		os.Exit(1) 
	}

	feedFollowRow, err := s.dbQueries.CreateFeedFollow(context.Background(),database.CreateFeedFollowParams{
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
		UserID: existingUser.ID,
		FeedID: existingFeed.ID,
	})
	if err != nil {
		fmt.Println("Error: Could not follow the feed")
		os.Exit(1) 
	}

	fmt.Println("Following feed: " + feedFollowRow.FeedName)
	fmt.Println("Current user: " + feedFollowRow.UserName)

	return nil
}


func handlerFollowing(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return errors.New("following accepts no arguments")
	}

	existingUser, err := s.dbQueries.GetUser(context.Background(), s.config.CURRENT_USER_NAME)
	if err != nil {
		fmt.Println("Error: Could not retrieve user details from the database")
		os.Exit(1) 
	}

	feeds, err := s.dbQueries.GetFeedFollowsForUser(context.Background(), existingUser.ID)
	if err != nil {
		fmt.Println("Error: Could not retrieve feed details from the database")
		return err
	}

	fmt.Println("Feeds followed by user: " + existingUser.Name)
	for _, name := range feeds {
		fmt.Println(name)
	}

	return nil
}
func handlerReset(s *state, cmd command) error {
	err := s.dbQueries.ResetTable(context.Background())
	if err != nil {
		return fmt.Errorf("unable to reset table users: %v", err)
	}

	fmt.Println("Successfully reset table users")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.dbQueries.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("unable to get users: %v", err)
	}

	for _, user := range users{
		if user.Name == s.config.CURRENT_USER_NAME {
			fmt.Println("* " + user.Name + " (current)")
		} else {
			fmt.Println("* " + user.Name)
		}
	}

	return nil
}

func main() {
	config := config.Config{}
	config.Read()
	programState := state{&config, nil}
	commands := commands{make(map[string]func(*state, command) error)}
		
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("reset", handlerReset)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAgg)
	commands.register("addfeed", handlerAddfeed)
	commands.register("feeds", handlerFeeds)
	commands.register("follow", handlerFollow)
	commands.register("following", handlerFollowing)

	db, err := sql.Open("postgres", config.DB_URL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	programState.dbQueries = dbQueries

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Too few arguments")
		os.Exit(1)
	}
	commandName := args[1]
	commandArgs := args[2:]
	command := command{commandName, commandArgs}
	err = commands.run(&programState, command)
	if err != nil {
		log.Fatal(err)
	}
}