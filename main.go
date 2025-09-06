package main

import (
	"os"
	"errors"
	"github.com/itstwoam/aggrogator/internal/config"
	"fmt"
	"strings"
	_"github.com/lib/pq"
	"context"
	"time"
	"github.com/itstwoam/aggrogator/internal/database"
	"database/sql"
	"github.com/google/uuid"
	"io"
	"net/http"
	"github.com/itstwoam/aggrogator/internal/rssparser"
	"encoding/xml"
	"github.com/mmcdole/gofeed"
	"strconv"
)

type state struct {
	db *database.Queries `json:"db"`
	cfg *config.Config `json:"cfg"`
}

type command struct {
	cmd string `json:"cmd"`
	args []string `json:"args"`
}

type commands struct {
	command map[string]func(*state, command, map[string]string) error
	helpText map[string]string
}

func main(){
	cfg, err := config.Read()
	if err != nil{
		fmt.Println("could not retrieve config file")
		return
	}
	var curState state
	curState.cfg = cfg
	db, err := sql.Open("postgres", curState.cfg.DB_url)
	curState.db = database.New(db)
	valCommands := commands{}
	valCommands.command = make(map[string]func(*state,command, map[string]string) error)
	valCommands.helpText = make(map[string]string)
	valCommands.register("login", handlerLogins, "login <username> - Logs in with specified username")
	valCommands.register("register", handlerRegister, "register <username> - Registers and logs in the specified user")
	valCommands.register("reset", handlerReset, "reset - Removes all users thus all feeds from the database")
	valCommands.register("users", handlerListUsers, "users - Lists all the users registered and indicates the currently logged in user.")
	valCommands.register("agg", handlerAgg, "agg - Gets a test RSS feed")
	valCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed), "addfeed <name> <url> - Adds a feed with required name and url to the feeds table.")
	valCommands.register("feeds", handlerFeeds, "feeds - Displays any saved feeds")
	valCommands.register("help", handlerHelp, "Help - displays this information")
	valCommands.register("follow", middlewareLoggedIn(handlerFollow), "follow <url> - Adds a new feed follow from the provided url to the user")
	valCommands.register("following", middlewareLoggedIn(handlerFollowing), "following - Lists all the feeds being followed by the current user")
	valCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow), "unfollow <name> - Removes the followed feed by name")
	valCommands.register("browse", middlewareLoggedIn(handlerBrowse), "browse <count> - Displays count posts (2 by default)")
	args := os.Args
	if len(args) < 2{
		fmt.Println("no arguments given")
		os.Exit(1)
	}
	curCommand := command{}
	curCommand.cmd = cleanInput(args[1])[0]
	curCommand.args = args[2:]
	err = valCommands.run(&curState, curCommand, valCommands.helpText)
	if err != nil{
		fmt.Println("invalid command or missing username in main")
		os.Exit(1)
	}
	return
}

func handlerLogins(s *state, cmd command, _ map[string]string) error {
	err := checkArgs(cmd, 1)
	if err != nil {
		return errors.New("no username provided for login")
	}
	uName := cmd.args[0]
	_, err = s.db.GetUserByName(context.Background(), uName)
	if err != nil {
		fmt.Println("User name does not exist, please use the 'register <username>' command first")
		os.Exit(1)
	}
	s.cfg.Current_user_name = cmd.args[0]
	fmt.Println("Username "+cmd.args[0]+" is logged in.")
	err = config.SetUser(s.cfg)
	if err != nil {
		return err
	}
	return nil
}

func handlerRegister(s *state, cmd command, _ map[string]string) error {
	err := checkArgs(cmd, 1)
	if err != nil {
		return errors.New("no username found")
	}
	uName := cmd.args[0]
	_, err = s.db.GetUserByName(context.Background(), uName)
	if err == nil {
		fmt.Println("user already exists, exiting")
		os.Exit(1)
	}
	if !errors.Is(err, sql.ErrNoRows){
		fmt.Println("unexpected error checking user %v\n", err)
			return fmt.Errorf("failed to check user: %w", err)
	}
	curTime := time.Now()
	//fmt.Print("curTime = "+curTime)
	_, err = s.db.CreateUser(context.Background(), database.CreateUserParams{ ID:	uuid.New(), CreatedAt: curTime, UpdatedAt: curTime, Name: uName,})
	if err != nil{
		fmt.Printf("error registering new user: %v\n", err)
		return errors.New("error registering new user")
	}
	fmt.Println("User "+uName+" has been registered.")
	err = handlerLogins(s, cmd, nil)
	if err != nil{
		return errors.New("Failed to login newly registered user")
		}
	return nil
}

func handlerReset(s *state, cmd command, _ map[string]string) error {
	usersDeleted, err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("Deleted %v users\n", usersDeleted)
	os.Exit(0)
	return nil
}

func handlerListUsers(s *state, cmd command, _ map[string]string) error {
	cUser := s.cfg.Current_user_name
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("error in listing uers")
		return errors.New("error in listing users")
	}
	for i := range users {
		final := "* "+ users[i]
		if users[i] == cUser {
			final += " (current)"
		}
		fmt.Println(final)
	}
	return nil
}

func handlerAgg(s *state, cmd command, _ map[string]string) error {
	err := checkArgs(cmd, 1)
	if err != nil {
		fmt.Println("invalid or bad wait time")
		os.Exit(1)
	}
	waitTime, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		fmt.Println("error in frequency parse, exiting")
		os.Exit(1)
	}
	ticker := time.NewTicker(waitTime)
	for ; ; <-ticker.C {
		feed, err := s.db.GetNextFeedToFetch(context.Background())
		//fmt.Printf("feed.ID: %v\n", feed.ID)
		//fmt.Printf("error: %v\n", err)
		if err != nil {
			fmt.Println("error retrieving next feed to update")
		}	
		rows, err := s.db.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{ ID: feed.ID, UpdatedAt: time.Now()})
		if rows < 1 {
			fmt.Println("no times updated")
		}
		if err != nil {
			fmt.Println("error in marking fetched feeds"+feed.Url)
		}
		//feedText, err := fetchFeed(context.Background(), feed.Url)		
		fp := gofeed.NewParser()
		parsedFeed, err := fp.ParseURL(feed.Url)
		if err != nil {
			fmt.Printf("Could not get feed %s\n", feed.Url)
		}else{	
			fmt.Println(parsedFeed.Title)
			fmt.Println(parsedFeed.Description)
			_ = addPosts(s, parsedFeed.Items, feed.ID)
		}
	}
	return nil
}

func handlerAddFeed(s *state, cmd command, _ map[string]string, user database.User) error {
	nonameorfeed := "name or url missing from addfeed command"
	err := checkArgs(cmd, 2)
	if err != nil {
		fmt.Println(nonameorfeed)
		return errors.New(nonameorfeed)
	}
	var name, feed string
	name = cmd.args[0]
	feed = cmd.args[1]
	curTime := time.Now()
	_, err = s.db.CreateFeed(context.Background(), database.CreateFeedParams{ID: uuid.New(),  Name:	name, Url: feed, CreatedAt: curTime, UpdatedAt: curTime, UserID: user.ID})
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		fmt.Printf("error creating feed: %s with url: %s for user: %s with UUID: %v\n", feed, name, user.Name, user.ID)
		return err
	}
	fmt.Printf("Created feed: %s\nAt URL: %s\nfor user: %s\n", name, feed, s.cfg.Current_user_name)
	cmd.args[0] = feed
	err = handlerFollow(s, cmd, nil, user)
	if err != nil {
		fmt.Println("failed to add feed_follow for user")
		return err
	}
	return nil
}

func handlerFeeds(s *state, cmd command, _ map[string]string) error {
	//Get all the feeds
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("failed to retrieve feeds")
		return err
	}
	if len(feeds) < 1 {
		fmt.Println("No feeds found.")
	}
	spacer := "---------------------------"
	for i := range feeds {
		feed := feeds[i]
		user, err := s.db.GetUser(context.Background(), feed.UserID)
		if err != nil {
			fmt.Println("failed to retrieve username")
			return err
		}
		fmt.Println(spacer)
		//fmt.Println("Name: "+ fromNullString(feed.Name))
		fmt.Println("Name: "+ feed.Name)
		fmt.Println("URL: " + feed.Url)
		fmt.Println("User: " + user.Name)
	}
	return nil
}

func handlerHelp(_ *state, _ command, help map[string]string) error {
	fmt.Println("Available commands: ")
	for _, desc := range help {
		fmt.Printf("%s\n", desc)
	}
	return nil
}

func handlerFollow(s *state, cmd command, _ map[string]string, user database.User) error{
	err := checkArgs(cmd, 1)
	if err != nil {
		fmt.Println("feed url not found")
		os.Exit(1)
	}
	//_, err = s.db.CreateFeed(context.Background(), database.CreateFeedParams{ID: uuid.New(),  Name:	name, Url: feed, CreatedAt: curTime, UpdatedAt: curTime, UserID: user.ID})
	curTime := time.Now()	
	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		fmt.Println("failed to retrieve feed record from url")
		fmt.Println(feed.Name)
		fmt.Println(feed.Url)
		return err
	}
	record, err := s.db.CreateFeedFollow(
			context.Background(),
			database.CreateFeedFollowParams{
				ID:  uuid.New(),
				CreatedAt: curTime,
				UpdatedAt: curTime,
				UserID:	user.ID,
				FeedID: feed.ID,
			},
		)
	if err != nil {
		fmt.Println("failed to create feed_follow record")
		os.Exit(1)
	}
	fmt.Printf("%s is now following the feed: %s\n", s.cfg.Current_user_name, record.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command, _ map[string]string, user database.User) error{
	uName := user.Name
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), uName)
	if err != nil {
		fmt.Println("failed to retrieve username from database")
		return err
	}
	if len(follows) < 1 {
		fmt.Println("User is not following any feeds.")
		return nil
	}
	spacer := "---------------------------"
	fmt.Println(spacer)
	fmt.Println("Here are the followed feeds for "+uName)
	for i := range follows {
		fmt.Println(follows[i].FeedName)	
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, _ map[string]string, user database.User) error{
	userUUID := user.ID
	err := checkArgs(cmd, 1)
	if err != nil {
		fmt.Println("Missing or invalid feed name")
	}
	//_, err = s.db.CreateFeed(context.Background(), database.CreateFeedParams{ID: uuid.New(),  Name:	name, Url: feed, CreatedAt: curTime, UpdatedAt: curTime, UserID: user.ID})
	deleted, err := s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{UserID: userUUID, Url: cmd.args[0]})
	if err != nil {
		fmt.Println("error deleting feed")
		return err
	}
	if deleted < 1 {
		fmt.Println("No feeds unfollowed.")
		return nil
	}
	fmt.Printf("Unfollowed %v feeds.\n", deleted)
	return nil	
	
}

func handlerBrowse(s *state, cmd command, _ map[string]string, user database.User) error {
	var limit int32 = 2
	if len(cmd.args) > 0 {
		parsed, err := strconv.Atoi(cmd.args[0])
		if err == nil {
			limit = int32(parsed)
		}
	}
	fmt.Printf("%v\n", limit)
	posts, err := s.db.GetRandomPosts(context.Background(), limit)
	if err != nil {
		fmt.Println("Failed to retrieve posts")
		os.Exit(1)
	}
	spacer := "---------------------"
	curTime := time.Now()
	for i := range posts {
		err = s.db.MarkPostRead(context.Background(), database.MarkPostReadParams{ ID: posts[i].ID,
			SeenAt: sql.NullTime{Time: curTime, Valid: true},
		})
		if err != nil {
			fmt.Println("Error in setting seen_at timestamp for "+posts[i].Url)
			os.Exit(1)
		}
		fmt.Println(spacer)
		if posts[i].Title.Valid {
			fmt.Printf(posts[i].Title.String)
		}
		if posts[i].Description.Valid {
			fmt.Println(posts[i].Description.String)
		}
		fmt.Println(posts[i].Url)
	}
	return nil
}

func (c *commands) run(s *state, cmd command, help map[string]string) error{
	//find the command to run
	cmdToRun := cmd.cmd
	if cmdToRun == "" {
		fmt.Println("command "+ cmdToRun + " not found.")
		return errors.New("command " + cmdToRun + "not found.")
	}
	//Does it exist in the map?
	mapCmd, exists := c.command[cmdToRun]
	if !exists {
		return errors.New("command "+cmdToRun+" does not exist in map")
	}
	return mapCmd(s, cmd, help)
}

func (c *commands) register(name string, f func(*state, command, map[string]string) error, help string){
	c.command[name] = f
	c.helpText[name] = help
}

func cleanInput(text string) []string{
	parts := strings.Fields(text)
	for i := range parts {
		parts[i] = strings.Trim(parts[i], " ")
		parts[i] = strings.ToLower(parts[i])
	}
	return parts
}

func cleanArgs(parts []string) []string{
	for i := range parts {
		parts[i] = strings.Trim(parts[i], " ")
		parts[i] = strings.ToLower(parts[i])
	}
	return parts
}

func fetchFeed(ctx context.Context, feedURL string) (*rssparser.RSSFeed, error){
	client := &http.Client{}
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RSSBot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var xmlData  rssparser.RSSFeed
	err = xml.Unmarshal(body, &xmlData)
	if err != nil {
		return nil, err
	}
	rssparser.Unescaper(&xmlData)
	return &xmlData, nil
}

func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func fromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "(null)"
}

func checkArgs(cmd command, count int) error{
	if len(cmd.args) < count {
		return errors.New("arg count less than required")
	}
	for i := 0; i < count; i++ {
		cmd.args[i] = strings.Trim(cmd.args[i], `"`)
		if len(cmd.args[i]) < 1 || cmd.args[i] == "" {
			return errors.New("invalid argument")
		}
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, help map[string]string, user database.User) error) func(*state, command, map[string]string) error {
	return func(s *state, cmd command, help map[string]string) error {
		user, err := s.db.GetUserByName(context.Background(), s.cfg.Current_user_name)
		if err != nil {
			return err
		}
		return handler(s, cmd, help, user)
	}
}

func addPosts(s *state, items []*gofeed.Item, feedID uuid.UUID) error {
	for _, post := range items {
		_ , err := s.db.GetPostByURL(context.Background(), post.Link)
		if err == sql.ErrNoRows {
	//_, err = s.db.CreateFeed(context.Background(), database.CreateFeedParams{ID: uuid.New(),  Name:	name, Url: feed, CreatedAt: curTime, UpdatedAt: curTime, UserID: user.ID})
			curTime := time.Now()
			published := time.Time{}
			if post.PublishedParsed != nil {
				published = *post.PublishedParsed
			}
			cPost, err := s.db.CreatePost(context.Background(),
				database.CreatePostParams{ID: uuid.New(),
					CreatedAt: curTime,
					UpdatedAt: curTime,
					SeenAt: sql.NullTime{Valid: false},
					Title: toNullString(post.Title),
					Url: post.Link,
					Description: toNullString(post.Description),
					PublishedAt: published,
					FeedID: feedID},
				)
			if err == nil {
				fmt.Println("Added post "+cPost.Url)
			}else{
				fmt.Println("error adding post "+cPost.Url)
			}
		}
	}
	return nil
}
