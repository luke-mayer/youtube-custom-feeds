package main
/*
import (
	"context"
	"errors"
	"fmt"
	"log"
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

// Retrieves unique user id
func (s *state) getUserId() (uuid.UUID, error) {
	user, err := s.db.GetUserName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("error retireving info for user with name: %s",
			s.cfg.CurrentUserName)
	}

	return user.ID, nil
}

//************************************/
//         Pipeline Functions         //
//************************************//

/*
// Creates a custom feed for a user
func createFeed(s *state, userId uuid.UUID, feedName string) (database.Feed, error) {
	feed := database.Feed{}
	context := context.Background()

	exists, err := s.db.ContainsUserById(context, userId)
	if err != nil {
		return feed, fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return feed, fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	params := database.CreateFeedParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		UserID:    userId,
	}

	feed, err = s.db.CreateFeed(context, params)
	if err != nil {
		return feed, fmt.Errorf("error creating feed \"%s\" for user with id %v", feedName, userId)
	}

	log.Printf("Successfully created feed with - feed_id: %v, feedName: %v, for user with userId: %v",
		feed.ID, feed.Name, feed.UserID)
	return feed, nil
}

// Retrieves all feeds belonging to the specified user
func getAllUserFeeds(s *state, userId uuid.UUID) ([]database.GetAllUserFeedsRow, error) {
	feeds := []database.GetAllUserFeedsRow{}
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return feeds, fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	feeds, err = s.db.GetAllUserFeeds(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("error retrieving feeds for user with id %v", userId)
	}

	return feeds, nil
}

// Retrieves feed id for the feed withe the provided name, belonging to the specified user
func getUserFeedId(s *state, userId uuid.UUID, feedName string) (int32, error) {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return 0, fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return 0, fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	params := database.GetFeedIdParams{
		UserID: userId,
		Name:   feedName,
	}

	feedId, err := s.db.GetFeedId(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error retrieving feed \"%s\" for user with id %v", feedName, userId)
	}

	return feedId, nil
}

// Deletes all feeds belonging to the specified user
func deleteAllFeeds(s *state, userId uuid.UUID) error {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	err = s.db.DeleteAllFeeds(ctx, userId)
	if err != nil {
		return fmt.Errorf("error deleting all feeds: %s", err)
	}

	return nil
}

// Deletes feed with given name belonging to the specified user
func deleteFeed(s *state, userId uuid.UUID, feedName string) error {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	params := database.DeleteFeedParams{
		UserID: userId,
		Name:   feedName,
	}

	err = s.db.DeleteFeed(ctx, params)
	if err != nil {
		return fmt.Errorf("error deleting feed: %s", err)
	}

	return nil
}

// Creates channel
func createChannel(s *state, channelId, channelName string) error {
	channelUrl := youtube.GetChannelURL(channelId)

	params := database.InsertChannelParams{
		ChannelID:  channelId,
		ChannelUrl: channelUrl,
		Name:       channelName,
	}

	channel, err := s.db.InsertChannel(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error inserting channel \"%s\" into database: %s", channelName, err)
	}

	log.Printf("Successfully inserted channel \"%s\" in database", channel.Name)
	return nil
}

// Creates feed channel
func createFeedChannel(s *state, feedId int32, channelId, channelName string) error {
	containsParams := database.ContainsFeedChannelParams{
		FeedID:    feedId,
		ChannelID: channelId,
	}

	exists, err := s.db.ContainsFeedChannel(context.Background(), containsParams)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	exists, err = s.db.ContainsChannel(context.Background(), channelId)
	if err != nil {
		return err
	}
	if !exists {
		err = createChannel(s, channelId, channelName)
		if err != nil {
			return err
		}
	}

	params := database.InsertFeedChannelParams{
		FeedID:    feedId,
		ChannelID: channelId,
	}

	err = s.db.InsertFeedChannel(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error inserting feedId: %v, and channelId %s, : %s", feedId, channelId, err)
	}

	return nil
}

// Deletes channel
func deleteChannel(s *state, channelId string) error {
	ctx := context.Background()

	err := s.db.DeleteChannel(ctx, channelId)
	if err != nil {
		return fmt.Errorf("error deleting channel with id: %v, :%s", channelId, err)
	}

	return nil
}

// Deletes feed channel and deletes channel if references in feeds_channels db
func deleteFeedChannel(s *state, feedId int32, channelId string) error {
	ctx := context.Background()

	params := database.DeleteFeedChannelParams{
		FeedID:    feedId,
		ChannelID: channelId,
	}

	err := s.db.DeleteFeedChannel(ctx, params)
	if err != nil {
		return fmt.Errorf("error deleting feed channel: %s", err)
	}

	exists, err := s.db.ContainsChannel(ctx, channelId)
	if err != nil {
		return err
	}
	if !exists { // deleting channel from channels if no more references in feeds_channels
		return deleteChannel(s, channelId)
	}

	return nil
}

// Gets all the channelIds for channels in feed
func getAllFeedChannels(s *state, feedId int32) ([]string, error) {

	channels, err := s.db.GetAllFeedChannels(context.Background(), feedId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllFeedChannels(): error getting channel ids for feed with id: %v, :%s", feedId, err)
	}

	return channels, nil
}

// Adds the channel to feed, calling createFeedChannel
func addChannelToFeed(s *state, feedId int32, channelName string) error {
	var channelId string
	var exists bool
	ctx := context.Background()

	userId, err := s.getUserId()
	if err != nil {
		return fmt.Errorf("in addChannelToFeed(): error retrieving userId: %s", userId)
	}

	channelId, err = s.db.GetChannelByName(ctx, channelName)
	if err != nil {
		log.Printf("in addChannelToFeed(): error getting channelId, channel name may not exist in db yet: %s", err)
		exists, channelId, err = youtube.GetChannelId(channelName)
		if err != nil {
			return fmt.Errorf("in addChannelToFeed(): error retrieving channelId: %s", err)
		} else if !exists {
			return fmt.Errorf("in addChannelToFeed(): channelName did not match any youtube channel")
		}
	}

	err = createFeedChannel(s, feedId, channelId, channelName)
	if err != nil {
		return fmt.Errorf("in addChannelToFeed(): error creating feed channel: %s", err)
	}

	return nil
}

// ***********************************/
//
//	Handler Functions          //
//
// ************************************//
/*
func handlerCreateFeed(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) != 1 {
		fmt.Println("Usage: add_feed \"feed name\" - quotes are necessary if name contains spaces")
		return errors.New("error: invalid arguments")
	}

	userId, err := s.getUserId()
	if err != nil {
		return err
	}

	feed, err := createFeed(s, userId, cmd.args[0])
	if err != nil {
		return fmt.Errorf("error creating feed: %s", err)
	}

	fmt.Printf("Successfully created feed with - feed_id: %v, feedName: %v, for user with userId: %v",
		feed.ID, feed.Name, feed.UserID)

	return nil
}

func handlerGetAllUserFeeds(s *state, cmd command) error {
	userId, err := s.getUserId()
	if err != nil {
		return err
	}

	feeds, err := getAllUserFeeds(s, userId)
	if err != nil {
		return fmt.Errorf("error retrieving feeds: %s", err)
	}

	fmt.Println("All Feeds:")
	for _, feed := range feeds {
		fmt.Printf(" - %s\n", feed.Name)
	}

	return nil
}

func handlerDeleteAllFeeds(s *state, cmd command) error {
	userId, err := s.getUserId()
	if err != nil {
		return err
	}

	err = deleteAllFeeds(s, userId)
	if err != nil {
		return err
	}

	return nil
}

func handlerDeleteFeed(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error: no arguments provided")
	}

	if len(cmd.args) != 1 {
		fmt.Println("Usage: delete_feed \"feed name\" - quotes are necessary if name contains spaces")
		return errors.New("error: invalid arguments")
	}

	userId, err := s.getUserId()
	if err != nil {
		return err
	}

	feedName := cmd.args[0]

	err = deleteFeed(s, userId, feedName)
	if err != nil {
		return err
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

	cmds.register("create_feed", handlerCreateFeed)
	cmds.register("get_all_feeds", handlerGetAllUserFeeds)
	cmds.register("delete_all_feeds", handlerDeleteAllFeeds)
	cmds.register("delete_feed", handlerDeleteFeed)

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
