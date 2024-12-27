package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shubham-Hazra/blog-aggregator/internal/database"
	"github.com/Shubham-Hazra/blog-aggregator/pkg/rss"
	"github.com/Shubham-Hazra/blog-aggregator/pkg/types"
	"github.com/google/uuid"
)

type Handler struct {
	state       *State
	commandMap  map[string]func(*State, types.Command) error
}

func NewHandler(state *State) (h *Handler){
	h = &Handler{}
	h.state = state
	h.commandMap = map[string]func(*State, types.Command) error{
			"login":     HandleLogin,
			"register":  HandleRegister,
			"reset":     HandleReset,
			"users":     HandleUsers,
			"agg":       HandleAgg,
			"addfeed":   middlewareLoggedIn(HandleAddFeed),
			"feeds":     HandleFeeds,
			"follow":    middlewareLoggedIn(HandleFollow),
			"unfollow":    middlewareLoggedIn(HandleUnfollow),
			"following": middlewareLoggedIn(HandleFollowing),
			"browse": middlewareLoggedIn(HandleBrowse),
		}
	return h
}

func (h *Handler) Execute(cmd types.Command) error {
	handler, exists := h.commandMap[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	return handler(h.state, cmd)
}

// HandleAgg handles the aggregation of RSS feeds
func HandleAgg(s *State, cmd types.Command) error {

	time_between_reqs := cmd.Args[0]
	timeBetweenRequests, err := time.ParseDuration(time_between_reqs)
	if err != nil {
		return err
	}

	fmt.Println("Collecting feeds every " + timeBetweenRequests.String())
	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			return err
		}
	}
}

func scrapeFeeds(s *State) error {
	nextFeed, err := s.DBQueries.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.DBQueries.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		return err
	}

	feed, err := rss.FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}

	for _, item := range feed.Channel.Items{
		err = savePostToDB(s, &item, &nextFeed)
		if err != nil {
			if isDuplicateURLError(err) {
				continue
			}
			log.Printf("Error saving post with URL %s: %v\n", item.Link, err)
		}
	}

	return nil
}

func savePostToDB(s *State, item *rss.Item, feed *database.Feed) error {
	publishedAt, err := time.Parse(time.RFC1123, item.PubDate)
	if err != nil {
		return err
	}
	err = s.DBQueries.CreatePost(context.Background(), database.CreatePostParams{
		Title:       item.Title,
		Url:         item.Link,
		Description: sql.NullString{String: item.Description, Valid: true},
		PublishedAt: sql.NullTime{Time: publishedAt, Valid: true},
		FeedID:      feed.ID,
	})
	return err
}

func isDuplicateURLError(err error) bool {
	return err != nil && err.Error() == "pq: duplicate key value violates unique constraint \"posts_url_key\""
}

func HandleBrowse(s *State, cmd types.Command, user database.User) error {
	var limit int = 2
	if len(cmd.Args) != 0 {
		var err error
		limit, err = strconv.Atoi(cmd.Args[0])
		if err != nil {
			return err 
		}
	}

	posts , err:= s.DBQueries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit: int32(limit),
	} )
	if err != nil {
		return err 
	}

	for _, post := range posts {
		printPostInfo(&post)
	}

	return nil
}

// HandleFeeds displays all feeds in the system
func HandleFeeds(s *State, cmd types.Command) error {
	feeds, err := s.DBQueries.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, item := range feeds {
		printDivider()
		fmt.Printf("Feed Name: %s\nFeed URL: %s\nUser Name: %s\n",
			item.FeedName, item.FeedUrl, item.UserName)
		printDivider()
	}
	return nil
}

// HandleLogin manages user login
func HandleLogin(s *State, cmd types.Command) error {
	if err := validateArgCount(cmd.Args, 1, "login"); err != nil {
		return err
	}

	userName := cmd.Args[0]
	if _, err := s.DBQueries.GetUser(context.Background(), userName); err != nil {
		fmt.Printf("Error: User %s is not registered\n", userName)
		os.Exit(1)
	}

	if err := s.Config.SetUser(userName); err != nil {
		return fmt.Errorf("error encountered while login: %w", err)
	}

	fmt.Println("User logged in successfully")
	return nil
}

// HandleRegister manages user registration
func HandleRegister(s *State, cmd types.Command) error {
	if err := validateArgCount(cmd.Args, 1, "register"); err != nil {
		return err
	}

	userName := cmd.Args[0]
	if err := validateNewUser(s, userName); err != nil {
		return err
	}

	user, err := createUser(s, userName)
	if err != nil {
		return err
	}

	s.Config.SetUser(userName)
	fmt.Printf("User %s has been successfully registered\n", userName)
	logUserDetails(user)
	return nil
}

// HandleAddFeed adds a new feed to the system
func HandleAddFeed(s *State, cmd types.Command, user database.User) error {
	if err := validateArgCount(cmd.Args, 2, "addfeed"); err != nil {
		return err
	}

	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	feed, err := createFeed(s, feedName, feedURL, user.ID)
	if err != nil {
		return err
	}

	_, err = createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		return err
	}

	logFeedDetails(feed)
	return nil
}

// HandleFollow allows a user to follow a feed
func HandleFollow(s *State, cmd types.Command, user database.User) error {
	if err := validateArgCount(cmd.Args, 1, "follow"); err != nil {
		return err
	}

	feedURL := cmd.Args[0]

	feed, err := s.DBQueries.GetFeedFromUrl(context.Background(), feedURL)
	if err != nil {
		fmt.Println("Error: Could not retrieve feed details from the database")
		os.Exit(1)
	}

	followRow, err := createFeedFollow(s, user.ID, feed.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Following feed: %s\nCurrent user: %s\n",
		followRow.FeedName, followRow.UserName)
	return nil
}

// HandleUnfollow allows a user to unfollow a feed
func HandleUnfollow(s *State, cmd types.Command, user database.User) error {
	if err := validateArgCount(cmd.Args, 1, "unfollow"); err != nil {
		return err
	}

	feedURL := cmd.Args[0]

	err := s.DBQueries.DeleteFeedFollowsForUser(context.Background(), database.DeleteFeedFollowsForUserParams{
		ID: user.ID,
		Url: feedURL,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Unfollowing feed: %s\nCurrent user: %s\n",
	feedURL, user.Name)
	return nil
}

// HandleFollowing shows feeds followed by current user
func HandleFollowing(s *State, cmd types.Command, user database.User) error {
	if err := validateArgCount(cmd.Args, 0, "following"); err != nil {
		return err
	}

	feeds, err := s.DBQueries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error retrieving feed details: %w", err)
	}

	fmt.Printf("Feeds followed by user: %s\n", user.Name)
	for _, name := range feeds {
		fmt.Println(name)
	}

	return nil
}

// HandleReset resets the database tables
func HandleReset(s *State, cmd types.Command) error {
	err := s.DBQueries.ResetTables(context.Background())
	if err != nil {
		return fmt.Errorf("unable to reset tables: %v", err)
	}

	fmt.Println("Successfully reset all the tables")
	return nil
}

// HandleUsers displays all users in the system
func HandleUsers(s *State, cmd types.Command) error {
	users, err := s.DBQueries.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("unable to get users: %v", err)
	}

	for _, user := range users {
		if user.Name == s.Config.CURRENT_USER_NAME {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

// Helper functions

func validateArgCount(args []string, expected int, commandName string) error {
	if len(args) != expected {
		return fmt.Errorf("%s accepts exactly %d argument(s)", commandName, expected)
	}
	return nil
}

func getCurrentUser(s *State) (database.User, error) {
	user, err := s.DBQueries.GetUser(context.Background(), s.Config.CURRENT_USER_NAME)
	if err != nil {
		fmt.Println("Error: Could not retrieve user details from the database")
		os.Exit(1)
	}
	return user, nil
}

func validateNewUser(s *State, userName string) error {
	existingUser, err := s.DBQueries.GetUser(context.Background(), userName)
	if err == nil && existingUser.Name == userName {
		fmt.Println("Error: A user with that name already exists.")
		log.Printf("Attempt to register an existing user: %v\n", userName)
		os.Exit(1)
	}
	return nil
}

func createUser(s *State, userName string) (database.User, error) {
	nullTime := getNullTime()
	return s.DBQueries.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
		Name:      userName,
	})
}

func createFeed(s *State, name, url string, userID uuid.UUID) (database.Feed, error) {
	nullTime := getNullTime()
	feed, err := s.DBQueries.CreateFeed(context.Background(), database.CreateFeedParams{
		Name:      name,
		Url:       url,
		UserID:    userID,
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
	})
	if err != nil {
		return feed, err
	}
	return feed, nil
}

func createFeedFollow(s *State, userID uuid.UUID, feedID int32) (database.CreateFeedFollowRow, error) {
	nullTime := getNullTime()
	followRow, err := s.DBQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		CreatedAt: nullTime,
		UpdatedAt: nullTime,
		UserID:    userID,
		FeedID:    feedID,
	})
	if err != nil {
		fmt.Println("Error: Could not create feed follow")
		os.Exit(1)
	}
	return followRow, nil
}

func getNullTime() sql.NullTime {
	return sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
}

func printDivider() {
	fmt.Println(strings.Repeat("#", 50))
}

func printPostInfo(post *database.GetPostsForUserRow) {
	printDivider()
	fmt.Printf("Feed Name: %v\nTitle: %v\nDescription: %v\nLink: %v\nPubDate: %v\n",
		post.FeedName, post.Title, post.Description.String, post.Url, post.PublishedAt.Time)
	printDivider()
}

func logUserDetails(user database.User) {
	log.Printf("id: %v, created_at: %v, updated_at: %v, name: %v\n",
		user.ID, user.CreatedAt, user.UpdatedAt, user.Name)
}

func logFeedDetails(feed database.Feed) {
	log.Printf("id: %v, name: %v, url: %v, user_id: %v, created_at: %v, updated_at: %v",
		feed.ID, feed.Name, feed.Url, feed.UserID, feed.CreatedAt, feed.UpdatedAt)
}

