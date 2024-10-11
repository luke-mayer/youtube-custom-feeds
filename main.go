package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/luke-mayer/youtube-custom-feeds/internal/config"
	"github.com/luke-mayer/youtube-custom-feeds/internal/database"
	"github.com/luke-mayer/youtube-custom-feeds/internal/youtube"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

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

type commands struct {
	allCommands map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.allCommands[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	cmdFunc, ok := c.allCommands[cmd.name]
	if !ok {
		err := fmt.Sprintf("error in run() - command %s not found in commands", cmd.name)
		return errors.New(err)
	}

	if err := cmdFunc(s, cmd); err != nil {
		newErr := fmt.Sprintf("error in run() - command %s function call returned error: %s", cmd.name, err)
		return errors.New(newErr)
	}
	return nil
}

func handlerLogin(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error in handlerLogin() - login requires a username argument")
	}

	if len(cmd.args) < 1 {
		return errors.New("error in handlerLogin() - a user name is required")
	}

	userName := cmd.args[0]

	contains, err := s.db.ContainsUser(context.Background(), userName)
	if err != nil {
		newErr := fmt.Sprintf("error: %s", err)
		return errors.New(newErr)
	}

	if !contains {
		newErr := fmt.Sprintf("error: \"%s\" is not a registered user.", userName)
		return errors.New(newErr)
	}

	s.cfg.SetUser(userName)
	fmt.Printf("User has been set to: %s\n", userName)

	return nil
}

// Used to add a user to the sql database
func handlerRegister(s *state, cmd command) error {
	if cmd.args == nil {
		fmt.Println("Usage: register \"user\"")
		return errors.New("error in handlerRegister() - no arguments")
	}

	if len(cmd.args) < 1 {
		fmt.Println("Usage: register \"user\"")
		return errors.New("error in handlerRegister() - not enough arguments provided")
	}

	params := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	}

	user, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		newErr := fmt.Sprintf("error in handlerRegister() - failed to register user. %s", err)
		return errors.New(newErr)
	}

	fmt.Printf("User \"%s\" with id \"%v\" successfully registered.\n", user.Name, user.ID)
	s.cfg.SetUser(user.Name)

	return nil
}

func handlerGetUserName(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) < 1 {
		fmt.Println("Usage: get_user_name \"user_name\"")
		return errors.New("error: not enough arguments provided")
	}

	user, err := s.db.GetUserName(context.Background(), cmd.args[0])
	if err != nil {
		newErr := fmt.Sprintf("error: failed to retrieve user: %s", err)
		return errors.New(newErr)
	}

	fmt.Println("name, id(uuid), created_at, updated_at")
	fmt.Printf("%s, %v, %v, %v\n", user.Name, user.ID, user.CreatedAt, user.UpdatedAt)
	return nil
}

func handlerGetUserId(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) < 1 {
		fmt.Println("Usage: get_user_id \"user_id\"")
		return errors.New("error: not enough arguments provided")
	}

	userId, err := uuid.Parse(cmd.args[0])
	if err != nil {
		newErr := fmt.Sprintf("error: failed to parse UUID from argument: %s", err)
		return errors.New(newErr)
	}

	user, err := s.db.GetUserId(context.Background(), userId)
	if err != nil {
		newErr := fmt.Sprintf("error: failed to retrieve user: %s", err)
		return errors.New(newErr)
	}

	fmt.Println("name, id(uuid), created_at, updated_at")
	fmt.Printf("%s, %v, %v, %v\n", user.Name, user.ID, user.CreatedAt, user.UpdatedAt)
	return nil
}

func handlerDeleteUserName(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) < 1 {
		fmt.Println("Usage: delete_user_name \"user_name\"")
		return errors.New("error: not enough arguments provided")
	}

	user, err := s.db.DeleteUserName(context.Background(), cmd.args[0])
	if err != nil {
		newErr := fmt.Sprintf("error: failed to delete user: %s", err)
		return errors.New(newErr)
	}

	fmt.Printf("Deleted user \"%s\" with id \"%v\" successfully.\n", user.Name, user.ID)
	return nil
}

func handlerDeleteUserId(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) < 1 {
		fmt.Println("Usage: delete_user_id \"user_id\"")
		return errors.New("error: not enough arguments provided")
	}

	userId, err := uuid.Parse(cmd.args[0])
	if err != nil {
		newErr := fmt.Sprintf("error: failed to parse UUID from argument: %s", err)
		return errors.New(newErr)
	}

	user, err := s.db.DeleteUserID(context.Background(), userId)
	if err != nil {
		newErr := fmt.Sprintf("error: failed to delete user: %s", err)
		return errors.New(newErr)
	}

	fmt.Printf("Deleted user \"%s\" with id \"%v\" successfully.\n", user.Name, user.ID)
	return nil
}

func handlerUpdateUserName(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) < 2 {
		fmt.Println("Usage: update_user_name \"user_id\" \"new user_name\"")
		return errors.New("error: not enough arguments provided")
	}

	userId, err := uuid.Parse(cmd.args[0])
	if err != nil {
		newErr := fmt.Sprintf("error: failed to parse UUID from argument: %s", err)
		return errors.New(newErr)
	}

	oldUser, err := s.db.GetUserId(context.Background(), userId)
	if err != nil {
		newErr := fmt.Sprintf("error retrieving current user name: %s", err)
		return errors.New(newErr)
	}

	params := database.UpdateUserNameParams{
		ID:        userId,
		Name:      cmd.args[1],
		UpdatedAt: time.Now(),
	}

	user, err := s.db.UpdateUserName(context.Background(), params)
	if err != nil {
		newErr := fmt.Sprintf("error: failed to update user: %s", err)
		return errors.New(newErr)
	}

	if oldUser.Name == s.cfg.CurrentUserName {
		s.cfg.SetUser(user.Name)
	}

	fmt.Printf("Successfully updated user with id \"%v\" with new name \"%s\"\n", user.ID, user.Name)
	return nil
}

func handlerGetUsers(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	users, err := s.db.GetAllUsers(context.Background())
	if err != nil {
		newErr := fmt.Sprintf("error: failed to retrieve users: %s", err)
		return errors.New(newErr)
	}

	fmt.Println("All users: ")
	fmt.Println("name, id(uuid), created_at, updated_at")
	for _, user := range users {
		if s.cfg.CurrentUserName == user.Name {
			fmt.Printf("(current) %s, %v, %v, %v\n", user.Name, user.ID, user.CreatedAt, user.UpdatedAt)
		} else {
			fmt.Printf("%s, %v, %v, %v\n", user.Name, user.ID, user.CreatedAt, user.UpdatedAt)
		}
	}

	return nil
}

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatalln("fatal error - less than 2 arguments provided")
	}

	channelName := args[1]

	service, err := youtube.GetService()
	if err != nil {
		log.Fatalln(err)
	}

	exists, channelId, err := youtube.GetChannelId(service, channelName)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("exists: %v, channelId: %s\n", exists, channelId)
	channelURL := youtube.GetChannelURL(channelId)
	fmt.Println(channelURL)

	recentVideos, err := youtube.GetRecentVideos(service, 5, channelId)
	if err != nil {
		log.Println(err)
	}

	youtube.PrintVideos(recentVideos)
}

/*
func main() {
	var s state

	tempCfg := config.Read() // retrieves state from application-tracker-config.json

	s.cfg = &tempCfg

	db, err := sql.Open("postgres", s.cfg.DBUrl)
	if err != nil {
		log.Fatalf("fatal error - issue opening connection to database: %s\n", err)
	}
	s.db = database.New(db)

	cmds := commands{
		allCommands: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("get_users", handlerGetUsers)
	cmds.register("get_user_name", handlerGetUserName)
	cmds.register("get_user_id", handlerGetUserId)
	cmds.register("delete_user_name", handlerDeleteUserName)
	cmds.register("delete_user_id", handlerDeleteUserId)
	cmds.register("update_user_name", handlerUpdateUserName)

	args := os.Args
	if len(args) < 2 {
		log.Fatalln("fatal error - less than 2 arguments provided")
	}

	cmd := command{
		name: args[1],
		args: args[2:],
	}

	err = cmds.run(&s, cmd) //runs the called command
	if err != nil {
		log.Fatalln(err)
	}
}
*/
