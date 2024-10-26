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

	err = db.Ping()
	if err != nil {
		log.Printf("Error pinging database: %v", err)
		return &state{}, fmt.Errorf("in getState(): error pinging database: %v", err)
	}

	s.db = database.New(db)

	return &s, nil
}

// Retrieves user id using a firebase user id
func getUserId(s *state, firebaseId string) (int32, error) {
	userId, err := s.db.GetUserIdByFirebaseId(context.Background(), firebaseId)
	if err != nil {
		return 0, fmt.Errorf("in getUserId(): error retrieving userId: %s", err)
	}

	return userId, nil
}

// Creates a new user in the database
func registerUser(s *state, firebaseId string) error {
	params := database.CreateUserParams{
		FbUserID:  firebaseId,
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
func createFeed(s *state, userId int32, feedName string) (bool, database.Feed, error) {
	feed := database.Feed{}
	ctx := context.Background()

	containsParams := database.ContainsFeedParams{
		UserID: userId,
		Name:   feedName,
	}

	contains, err := s.db.ContainsFeed(ctx, containsParams)
	if err != nil {
		return false, feed, fmt.Errorf("in createFeed(): error checking if user already has a feed with provided name: %s", err)
	}
	if contains {
		return true, feed, nil
	}

	params := database.CreateFeedParams{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		UserID:    userId,
	}

	feed, err = s.db.CreateFeed(ctx, params)
	if err != nil {
		return false, feed, fmt.Errorf("error creating feed \"%s\" for user with id %v", feedName, userId)
	}

	log.Printf("Successfully created feed with - feed_id: %v, feedName: %v, for user with userId: %v",
		feed.ID, feed.Name, feed.UserID)
	return false, feed, nil
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

// Retrieves all feedNames belonging to the specified user
func getAllUserFeedNames(s *state, userId int32) ([]string, error) {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error checking if userId exists: %s", err)
	}
	if !exists {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error user with id %v does not exist in database", userId)
	}

	feedNames, err := s.db.GetAllUserFeedNames(ctx, userId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error retrieving feedNames for user with id %v", userId)
	}

	return feedNames, nil
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

// Deletes the user, including all of their feeds and subsequent channels
func deleteUser(s *state, userId int32) error {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("in deleteUser(): error checking if userId exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("in deleteUser(): error checking if userId exists: %s", err)
	}

	err = deleteAllFeeds(s, userId)
	if err != nil {
		return fmt.Errorf("in deleteUser(): error deleting all feeds: %s", err)
	}

	err = s.db.DeleteUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("in deleteUser(): error deleting user from database: %s", err)
	}

	return nil
}

// Deletes all feeds belonging to the specified user
func deleteAllFeeds(s *state, userId int32) error {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("in deleteAllFeeds(): error checking if userId exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("in deleteAllFeeds(): error user with id %v does not exist in database", userId)
	}

	feedNames, err := getAllUserFeedNames(s, userId)
	if err != nil {
		return fmt.Errorf("in deleteAllFeeds(): error retrieving all user feedNames: %s", err)
	}

	for _, feedName := range feedNames {
		err := deleteFeed(s, userId, feedName)
		if err != nil {
			return fmt.Errorf("in deleteAllFeeds(): Error deleing all feeds: %s", err)
		}
	}

	return nil
}

// Deletes feed with given name belonging to the specified user.
// Deletes all feed-channels as a consequence
func deleteFeed(s *state, userId int32, feedName string) error {
	ctx := context.Background()

	exists, err := s.db.ContainsUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("error checking if userId exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("error user with id %v does not exist in database", userId)
	}

	feedId, err := getUserFeedId(s, userId, feedName)
	if err != nil {
		return fmt.Errorf("in deleteFeed(): error retrieving feedId: %s", err)
	}

	err = deleteAllFeedChannels(s, feedId)
	if err != nil {
		return fmt.Errorf("in deleteFeed(): error deleting all feed-channels: %s", err)
	}

	params := database.DeleteFeedParams{
		UserID: userId,
		Name:   feedName,
	}

	err = s.db.DeleteFeed(ctx, params)
	if err != nil {
		return fmt.Errorf("in deleteFeed(): error deleting feed: %s", err)
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

// Deletes feed channel and deletes channel if no remaining references in feeds_channels db
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

// Deletes all channels in the provided feed
func deleteAllFeedChannels(s *state, feedId int32) error {
	channelIds, err := getAllFeedChannels(s, feedId)
	if err != nil {
		return fmt.Errorf("in deleteAllFeedChannels(): error retrieving all channelIds in feed: %s", err)
	}

	for _, channelId := range channelIds {
		err := deleteFeedChannel(s, feedId, channelId)
		if err != nil {
			return fmt.Errorf("in deleteAllFeedChannels(): error deleing channel-feed: %s", err)
		}
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
			log.Println(fmt.Errorf("in getAllUploadIds(): error retrieiving uploadId: %s", err))
			continue
		}
		uploadIds = append(uploadIds, uploadId)
	}

	/*
		if len(uploadIds) < 1 {
			return []string{}, fmt.Errorf("in getAllUploadIds(): error retrieving uploadIds, not a single Id retrieved")
		}
	*/

	return uploadIds, nil
}

// Retrieves all handles associated with the provided channelIds
func getAllChannelHandles(s *state, channelIds []string) ([]string, error) {
	uploadIds := []string{}

	for _, channelId := range channelIds {
		uploadId, err := s.db.GetChannelHandle(context.Background(), channelId)
		if err != nil {
			log.Println(fmt.Errorf("in getAllChannelHandles(): error retrieiving handle: %s", err))
			continue
		}
		uploadIds = append(uploadIds, uploadId)
	}

	return uploadIds, nil
}

// Retrieves the channelId associated with the given handle
func getChannelId(s *state, channelHandle string) (string, error) {
	ctx := context.Background()

	channelId, err := s.db.GetChannelIdByHandle(ctx, channelHandle)
	if err != nil {
		return "", fmt.Errorf("in getChannelId(): error retrieving channelId for channelHandle<%s>: %s", channelHandle, err)
	}

	return channelId, nil
}

// Adds the channel to feed, calling createFeedChannel
func addChannelToFeed(s *state, feedId int32, channelHandle string) error {
	var channelId, uploadId string
	var exists bool
	ctx := context.Background()

	contains, err := s.db.ContainsChannelInDB(ctx, channelHandle)
	if err != nil {
		return fmt.Errorf("in addChannelToFeed(): error checking if DB contains channel: %v", err)
	}
	if !contains {
		exists, channelId, uploadId, err = youtube.GetChannelIdUploadId(channelHandle)
		if err != nil {
			return fmt.Errorf("in addChannelToFeed(): error retrieving channelId: %s", err)
		} else if !exists {
			return fmt.Errorf("in addChannelToFeed(): channelHandle did not match any youtube channel")
		}
	} else {
		channelIdUploadId, err := s.db.GetChannelIdUploadIdByHandle(ctx, channelHandle)
		if err != nil {
			return fmt.Errorf("in addChannelToFeed(): error retrieving channelId and uploadId: %s", err)
		}
		channelId = channelIdUploadId.ChannelID
		uploadId = channelIdUploadId.ChannelUploadID
	}

	err = createFeedChannel(s, feedId, channelId, uploadId, channelHandle)
	if err != nil {
		return fmt.Errorf("in addChannelToFeed(): error creating feed channel: %s", err)
	}

	return nil
}

// Updates the name of the specified feed belonging to the specified user
func updateFeedName(s *state, feedId int32, newFeedName string) error {
	params := database.UpdateFeedNameQueryParams{
		ID:        feedId,
		Name:      newFeedName,
		UpdatedAt: time.Now(),
	}

	err := s.db.UpdateFeedNameQuery(context.Background(), params)
	if err != nil {
		return fmt.Errorf("in updateFeedName(): error updating the feed name: %s", err)
	}

	return nil
}
