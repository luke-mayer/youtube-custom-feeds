package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/luke-mayer/youtube-custom-feeds/internal/config"
	"github.com/luke-mayer/youtube-custom-feeds/internal/database"
	"github.com/luke-mayer/youtube-custom-feeds/internal/youtube"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

// retrieves the current state with sql database connection and current userName
func getState() (*state, error) {
	var s state

	tempCfg, err := config.Read() // retrieves state from youtube-custom-feeds.json
	if err != nil {
		return &state{}, fmt.Errorf("in getState(): error retireving config json: %s", err)
	}

	s.cfg = &tempCfg

	db, err := sql.Open("postgres", s.cfg.DBUrl)
	if err != nil {
		return &state{}, fmt.Errorf("in getState(): error connecting to database: %s", err)
	}
	s.db = database.New(db)

	return &s, nil
}

// Retrieves user id using a google oauth2 id
func getUserId(s *state, googleId string) (int32, error) {
	userId, err := s.db.GetUserIdByGoogleId(context.Background(), googleId)
	if err != nil {
		return 0, fmt.Errorf("in getUserId(): error retrieving userId: %s", err)
	}

	return userId, nil
}

// Creates a new user in the database
func registerUser(s *state, googleId string) error {
	params := database.CreateUserParams{
		GoogleID:  googleId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error in registerUser(): error creating user in database: %s", err)
	}

	return nil
}

//************************************//
//         Pipeline Functions         //
//************************************//

// Creates a custom feed for a user
func createFeed(s *state, userId int32, feedName string) (database.Feed, error) {
	feed := database.Feed{}
	ctx := context.Background()

	params := database.CreateFeedParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		UserID:    userId,
	}

	feed, err := s.db.CreateFeed(ctx, params)
	if err != nil {
		return feed, fmt.Errorf("error creating feed \"%s\" for user with id %v", feedName, userId)
	}

	log.Printf("Successfully created feed with - feed_id: %v, feedName: %v, for user with userId: %v",
		feed.ID, feed.Name, feed.UserID)
	return feed, nil
}

// Retrieves all feeds belonging to the specified user
func getAllUserFeeds(s *state, userId int32) ([]database.GetAllUserFeedsRow, error) {
	feeds := []database.GetAllUserFeedsRow{}
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error checking if userId exists: %s", err)
	}
	if !exists {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error user with id %v does not exist in database", userId)
	}

	feeds, err = s.db.GetAllUserFeeds(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error retrieving feeds for user with id %v", userId)
	}

	return feeds, nil
}

// Retrieves feed id for the feed withe the provided name, belonging to the specified user
func getUserFeedId(s *state, userId int32, feedName string) (int32, error) {
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
func deleteAllFeeds(s *state, userId int32) error {
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
func deleteFeed(s *state, userId int32, feedName string) error {
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
func createChannel(s *state, channelId, uploadId, channelHandle string) error {
	channelUrl := youtube.GetChannelURL(channelId)

	params := database.InsertChannelParams{
		ChannelID:       channelId,
		ChannelUploadID: uploadId,
		ChannelUrl:      channelUrl,
		ChannelHandle:   channelHandle,
	}

	channel, err := s.db.InsertChannel(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error inserting channel \"%s\" into database: %s", channelHandle, err)
	}

	log.Printf("Successfully inserted channel \"%s\" in database", channel.ChannelHandle)
	return nil
}

// Creates feed channel
func createFeedChannel(s *state, feedId int32, channelId, uploadId, channelHandle string) error {
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
		err = createChannel(s, channelId, uploadId, channelHandle)
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

// Retrieves all uploadIds associated with the provided channelIds
func getAllUploadIds(s *state, channelIds []string) ([]string, error) {
	uploadIds := []string{}

	for _, channelId := range channelIds {
		uploadId, err := s.db.GetUploadId(context.Background(), channelId)
		if err != nil {
			log.Println(fmt.Errorf("in getAllFeedChannels(): error retrieiving uploadId: %s", err))
			continue
		}
		uploadIds = append(uploadIds, uploadId)
	}

	if len(uploadIds) < 1 {
		return []string{}, fmt.Errorf("in getAllFeedChannels(): error retrieving uploadIds, not a single Id retrieved")
	}

	return uploadIds, nil
}

// Adds the channel to feed, calling createFeedChannel
func addChannelToFeed(s *state, feedId int32, channelHandle string) error {
	var channelId, uploadId string
	var exists bool
	ctx := context.Background()

	channelIdUploadId, err := s.db.GetChannelIdUploadIdByHandle(ctx, channelHandle)
	if err != nil {
		log.Printf("in addChannelToFeed(): error getting channelId, channel name may not exist in db yet: %s", err)
		exists, channelId, uploadId, err = youtube.GetChannelIdUploadId(channelHandle)
		if err != nil {
			return fmt.Errorf("in addChannelToFeed(): error retrieving channelId: %s", err)
		} else if !exists {
			return fmt.Errorf("in addChannelToFeed(): channelHandle did not match any youtube channel")
		}
	} else {
		channelId = channelIdUploadId.ChannelID
		uploadId = channelIdUploadId.ChannelUploadID
	}

	err = createFeedChannel(s, feedId, channelId, uploadId, channelHandle)
	if err != nil {
		return fmt.Errorf("in addChannelToFeed(): error creating feed channel: %s", err)
	}

	return nil
}
