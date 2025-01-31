package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/luke-mayer/youtube-custom-feeds/internal/config"
	"github.com/luke-mayer/youtube-custom-feeds/internal/database"
	"github.com/luke-mayer/youtube-custom-feeds/internal/youtube"
)

type State struct {
	Db  *database.Queries
	Cfg *config.Config
}

// retrieves the current state with sql database connection and current userName
func GetState() (*State, error) {
	var s State

	tempCfg, err := config.Read() // Gets db info
	if err != nil {
		return &State{}, fmt.Errorf("in getState(): error retireving config json: %s", err)
	}

	s.Cfg = &tempCfg

	db, err := sql.Open("postgres", s.Cfg.DBUrl)
	if err != nil {
		return &State{}, fmt.Errorf("in getState(): error connecting to database: %s", err)
	}

	err = db.Ping()
	if err != nil {
		log.Printf("Error pinging database: %v", err)
		return &State{}, fmt.Errorf("in getState(): error pinging database: %v", err)
	}

	s.Db = database.New(db)

	return &s, nil
}

// Creates a new user in the database
func registerUser(s *State, firebaseId string) error {
	params := database.CreateUserParams{
		FbUserID:  firebaseId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.Db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error in registerUser(): error creating user in database: %s", err)
	}

	return nil
}

//************************************//
//         Pipeline Functions         //
//************************************//

// Creates a custom feed for a user
func createFeed(s *State, userId string, feedName string) (bool, database.Feed, error) {
	feed := database.Feed{}
	ctx := context.Background()

	containsParams := database.ContainsFeedParams{
		UserID: userId,
		Name:   feedName,
	}

	contains, err := s.Db.ContainsFeed(ctx, containsParams)
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

	feed, err = s.Db.CreateFeed(ctx, params)
	if err != nil {
		return false, feed, fmt.Errorf("error creating feed \"%s\" for user with id %v", feedName, userId)
	}

	log.Printf("Successfully created feed with - feed_id: %v, feedName: %v, for user with userId: %v",
		feed.ID, feed.Name, feed.UserID)
	return false, feed, nil
}

// Retrieves all feeds belonging to the specified user
func getAllUserFeeds(s *State, userId string) ([]database.GetAllUserFeedsRow, error) {
	feeds := []database.GetAllUserFeedsRow{}
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error checking if userId exists: %s", err)
	}
	if !exists {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error user with firebase id %v does not exist in database", userId)
	}

	feeds, err = s.Db.GetAllUserFeeds(ctx, userId)
	if err != nil {
		return feeds, fmt.Errorf("in getAllUserFeeds(): error retrieving feeds for user with id %v", userId)
	}

	return feeds, nil
}

// Retrieves all feedNames belonging to the specified user
func getAllUserFeedNames(s *State, userId string) ([]string, error) {
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error checking if userId exists: %s", err)
	}
	if !exists {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error user with id %v does not exist in database", userId)
	}

	feedNames, err := s.Db.GetAllUserFeedNames(ctx, userId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllUserFeedNames(): error retrieving feedNames for user with id %v", userId)
	}

	return feedNames, nil
}

// Retrieves feed id for the feed withe the provided name, belonging to the specified user
func getUserFeedId(s *State, userId string, feedName string) (int32, error) {
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
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

	feedId, err := s.Db.GetFeedId(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error retrieving feed \"%s\" for user with id %v", feedName, userId)
	}

	return feedId, nil
}

// Deletes the user, including all of their feeds and subsequent channels
func deleteUser(s *State, userId string) error {
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
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

	err = s.Db.DeleteUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("in deleteUser(): error deleting user from database: %s", err)
	}

	return nil
}

// Deletes all feeds belonging to the specified user
func deleteAllFeeds(s *State, userId string) error {
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
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
func deleteFeed(s *State, userId string, feedName string) error {
	ctx := context.Background()

	exists, err := s.Db.ContainsUserByFirebaseId(ctx, userId)
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

	err = s.Db.DeleteFeed(ctx, params)
	if err != nil {
		return fmt.Errorf("in deleteFeed(): error deleting feed: %s", err)
	}

	return nil
}

// Creates channel
func createChannel(s *State, channelId, uploadId, channelHandle string) error {
	channelUrl := youtube.GetChannelURL(channelId)

	params := database.InsertChannelParams{
		ChannelID:       channelId,
		ChannelUploadID: uploadId,
		ChannelUrl:      channelUrl,
		ChannelHandle:   channelHandle,
	}

	channel, err := s.Db.InsertChannel(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error inserting channel \"%s\" into database: %s", channelHandle, err)
	}

	log.Printf("Successfully inserted channel \"%s\" in database", channel.ChannelHandle)
	return nil
}

// Creates feed channel
func createFeedChannel(s *State, feedId int32, channelId, uploadId, channelHandle string) error {
	containsParams := database.ContainsFeedChannelParams{
		FeedID:    feedId,
		ChannelID: channelId,
	}

	exists, err := s.Db.ContainsFeedChannel(context.Background(), containsParams)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	exists, err = s.Db.ContainsChannel(context.Background(), channelId)
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

	err = s.Db.InsertFeedChannel(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error inserting feedId: %v, and channelId %s, : %s", feedId, channelId, err)
	}

	return nil
}

// Deletes channel
func deleteChannel(s *State, channelId string) error {
	ctx := context.Background()

	err := s.Db.DeleteChannel(ctx, channelId)
	if err != nil {
		return fmt.Errorf("error deleting channel with id: %v, :%s", channelId, err)
	}

	return nil
}

// Deletes feed channel and deletes channel if no remaining references in feeds_channels db
func deleteFeedChannel(s *State, feedId int32, channelId string) error {
	ctx := context.Background()

	params := database.DeleteFeedChannelParams{
		FeedID:    feedId,
		ChannelID: channelId,
	}

	err := s.Db.DeleteFeedChannel(ctx, params)
	if err != nil {
		return fmt.Errorf("error deleting feed channel: %s", err)
	}

	exists, err := s.Db.ContainsChannel(ctx, channelId)
	if err != nil {
		return err
	}
	if !exists { // deleting channel from channels if no more references in feeds_channels
		return deleteChannel(s, channelId)
	}

	return nil
}

// Deletes all channels in the provided feed
func deleteAllFeedChannels(s *State, feedId int32) error {
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
func getAllFeedChannels(s *State, feedId int32) ([]string, error) {

	channels, err := s.Db.GetAllFeedChannels(context.Background(), feedId)
	if err != nil {
		return []string{}, fmt.Errorf("in getAllFeedChannels(): error getting channel ids for feed with id: %v, :%s", feedId, err)
	}

	return channels, nil
}

// Retrieves all uploadIds associated with the provided channelIds
func getAllUploadIds(s *State, channelIds []string) ([]string, error) {
	uploadIds := []string{}

	for _, channelId := range channelIds {
		uploadId, err := s.Db.GetUploadId(context.Background(), channelId)
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
func getAllChannelHandles(s *State, channelIds []string) ([]string, error) {
	uploadIds := []string{}

	for _, channelId := range channelIds {
		uploadId, err := s.Db.GetChannelHandle(context.Background(), channelId)
		if err != nil {
			log.Println(fmt.Errorf("in getAllChannelHandles(): error retrieiving handle: %s", err))
			continue
		}
		uploadIds = append(uploadIds, uploadId)
	}

	return uploadIds, nil
}

// Retrieves the channelId associated with the given handle
func getChannelId(s *State, channelHandle string) (string, error) {
	ctx := context.Background()

	channelId, err := s.Db.GetChannelIdByHandle(ctx, channelHandle)
	if err != nil {
		return "", fmt.Errorf("in getChannelId(): error retrieving channelId for channelHandle<%s>: %s", channelHandle, err)
	}

	return channelId, nil
}

// Adds the channel to feed, calling createFeedChannel
func addChannelToFeed(s *State, feedId int32, channelHandle string) error {
	var channelId, uploadId string
	var exists bool
	ctx := context.Background()

	contains, err := s.Db.ContainsChannelInDB(ctx, channelHandle)
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
		channelIdUploadId, err := s.Db.GetChannelIdUploadIdByHandle(ctx, channelHandle)
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
func updateFeedName(s *State, feedId int32, newFeedName string) error {
	params := database.UpdateFeedNameQueryParams{
		ID:        feedId,
		Name:      newFeedName,
		UpdatedAt: time.Now(),
	}

	err := s.Db.UpdateFeedNameQuery(context.Background(), params)
	if err != nil {
		return fmt.Errorf("in updateFeedName(): error updating the feed name: %s", err)
	}

	return nil
}

const PORT = ":8080"
const PREFIX = "/api/v1"
const VIDEO_LIMIT = 10

/*
// Used to unpack parameters from request and initialize the state and userId, returns statusCode if error
func unpackRequest[T parameters](params *T, r *http.Request, s *State) (int32, int, error) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(params)
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error decoding parameters: %s", err)
		return 0, statusCodes.ErrDecoding, newErr
	}

	firebaseId := r.Header.Get("Firebase-ID")
	if firebaseId == "" {
		newErr := fmt.Errorf("in unpackRequest(): error retrieving firebaseId")
		return 0, statusCodes.ErrFirebaseId, newErr
	}

	userId, err := getUserId(s, firebaseId)
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error retrieving userId: %s", err)
		return 0, statusCodes.ErrUserId, newErr
	}

	return userId, statusCodes.Success, nil
}

func unpackGetRequest(r *http.Request, s *State) (int32, int, error) {

	firebaseId := r.Header.Get("Firebase-ID")
	if firebaseId == "" {
		newErr := fmt.Errorf("in unpackGetRequest(): error retrieving firebaseId")
		return 0, statusCodes.ErrFirebaseId, newErr
	}

	userId, err := getUserId(s, firebaseId)
	if err != nil {
		newErr := fmt.Errorf("in unpackGetRequest(): error retrieving userId: %s", err)
		return 0, statusCodes.ErrUserId, newErr
	}

	return userId, statusCodes.Success, nil
}

// Used to write messages (such as errors) to response
func writeResponseMessage(w http.ResponseWriter, message string, statusCode int) {
	type returnVals struct {
		Message string `json:"message"`
	}
	resBody := returnVals{
		Message: message,
	}

	writeResponse(w, resBody, statusCode)
}

// Used to write to response body
func writeResponse[T any](w http.ResponseWriter, resBody T, statusCode int) {
	data, err := json.Marshal(resBody)
	if err != nil {
		log.Printf("in writeResponseData(): error marshaling JSON: %s", err)
		w.WriteHeader(statusCodes.ErrMarshaling)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(statusCode)
	w.Write(data)
}
*/

/*
// validates OAuth2 ID token and returns firebaseId (sum field)
func validateFirebaseId(token string) (string, error) {
	clientId, err := config.GetClientId()
	if err != nil {
		return "", fmt.Errorf("in validateFirebaseId(): error retrieiving client id: %s", err)
	}

	payload, err := idtoken.Validate(context.Background(), token, clientId)
	if err != nil {
		return "", fmt.Errorf("in validateFirebaseId(): error validating token: %s", err)
	}

	firebaseId, ok := payload.Claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("in validateToken(): error extracting firebaseId from token: %s", err)
	}

	return firebaseId, nil
}
*/

// ------------------------ //
//		API ENDPOINTS		//
// ------------------------ //

// POST - Checks if user is in the database. If not, creates a new user
/*
func (s *State) login(w http.ResponseWriter, r *http.Request) {

	firebaseId := r.Header.Get("Firebase-ID")
	if firebaseId == "" {
		log.Println("in login(): error retireving firebaseId")
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFirebaseId], statusCodes.ErrFirebaseId)
		return
	}

	exists, err := s.Db.ContainsUserByFirebaseId(context.Background(), firebaseId)
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrServer], err)
		log.Println(errMessage)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := statusCodeMessages[statusCodes.Success]

	if !exists {
		err := registerUser(s, firebaseId)
		if err != nil {
			errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrServer], err)
			log.Println(errMessage)
			writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
			return
		}
		message = "user did not exist in database - created new user"
	}

	writeResponseMessage(w, message, statusCodes.Success)
}
*/

type Message struct {
	Message string `json:"message"`
}

// POST - Checks if user is in the database. If not, creates a new user
func (s *State) Login(c echo.Context) error {
	firebaseId := c.Request().Header.Get("Firebase-ID")
	if firebaseId == "" {
		log.Println("in login(): error retireving firebaseId")
		return echo.NewHTTPError(http.StatusUnauthorized, "Firebase-ID is not present")
	}

	exists, err := s.Db.ContainsUserByFirebaseId(context.Background(), firebaseId)
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", "Error checking if user exists in database", err)
		log.Println(errMessage)
		return echo.NewHTTPError(http.StatusInternalServerError, "")
	}

	if !exists {
		err := registerUser(s, firebaseId)
		if err != nil {
			errMessage := fmt.Sprintf("in login(): %s: %s", "Issue registering new user", err)
			log.Println(errMessage)
			return echo.NewHTTPError(http.StatusInternalServerError, "Issue registering new user")
		}
	}

	message := Message{
		Message: "Success",
	}

	return c.JSON(http.StatusOK, message)
}

// POST - Creates a new feed
func (s *State) CreateFeedHandler(c echo.Context) error {
	userId := c.Request().Header.Get("Firebase-Id")
	if userId == "" {
		log.Print("in createFeedHandler: Could not retreive Firebase-Id from request")
		return echo.NewHTTPError(http.StatusUnauthorized, "Firebase-ID is not present")
	}

	feedName := c.FormValue("feedName")
	if feedName == "" {
		log.Printf("in createFeedHandler: Could not retrieve feedName from request")
		return echo.NewHTTPError(http.StatusBadRequest, "feedName is not present")
	}

	contains, _, err := createFeed(s, userId, feedName)
	if err != nil {
		log.Printf("in createFeedHandler: error creating feed: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "feedName is not present")
	}
	if contains {
		message := fmt.Sprintf("Feed with name - %s - already exists for specified user", params.FeedName)
		writeResponseMessage(w, message, statusCodes.ErrFeedExists)
		return
	}

	message := fmt.Sprintf("Feed - %s - successfully created", params.FeedName)
	writeResponseMessage(w, message, statusCodes.Success)
}

// POST - adds the youtube channel to the user's indicated field
func (s *State) addChannelPOST(w http.ResponseWriter, r *http.Request) {
	params := feedChannelParams{}

	userId, statusCode, err := unpackRequest(&params, r, s)
	if err != nil {
		log.Printf("in addChannelPOST(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedId, err := getUserFeedId(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in addChannelPOST(): error retrieving feedId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	err = addChannelToFeed(s, feedId, params.ChannelHandle)
	if err != nil {
		log.Printf("in addChannelPOST(): error adding channel to feed: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := fmt.Sprintf("Channel - %s - successfully added to feed - %s", params.ChannelHandle, params.FeedName)
	writeResponseMessage(w, message, statusCodes.Success)
}

// GET - retrieves the user's feed names
func (s *State) getFeedsGET(w http.ResponseWriter, r *http.Request) {

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in getFeedsGET(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedNames, err := getAllUserFeedNames(s, userId)
	if err != nil {
		log.Printf("in getFeedsGET(): error retrieving feedNames: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := "Successfully retrieved feedNames"
	type returnVals struct {
		Message   string   `json:"message"`
		FeedNames []string `json:"feedNames"`
	}
	resBody := returnVals{
		Message:   message,
		FeedNames: feedNames,
	}

	writeResponse(w, resBody, statusCodes.Success)
}

// GET - retrieves the channel handles belonging to the user's specified feed
func (s *State) getChannelsGET(w http.ResponseWriter, r *http.Request) {

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in getChannelsGET(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedName := r.URL.Query().Get("feedName")

	feedId, err := getUserFeedId(s, userId, feedName)
	if err != nil {
		log.Printf("in getChannelsGET(): error retrieving feedId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	channelIds, err := getAllFeedChannels(s, feedId)
	if err != nil {
		log.Printf("in getChannelsGET(): error retrieving feed channel Ids: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	channelHandles, err := getAllChannelHandles(s, channelIds)
	if err != nil {
		log.Printf("in getChannelsGET(): error retrieving handles: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := "Successfully retrieved channel handles"
	type returnVals struct {
		Message        string   `json:"message"`
		ChannelHandles []string `json:"channelHandles"`
	}
	resBody := returnVals{
		Message:        message,
		ChannelHandles: channelHandles,
	}

	writeResponse(w, resBody, statusCodes.Success)
}

// GET - retrieves youtube videos for the provided feed
func (s *State) getVideosGET(w http.ResponseWriter, r *http.Request) {

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in getVideosGET(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedName := r.URL.Query().Get("feedName")

	feedId, err := getUserFeedId(s, userId, feedName)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving feedId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	channelIds, err := getAllFeedChannels(s, feedId)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving feed channel Ids: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	uploadIds, err := getAllUploadIds(s, channelIds)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving feed upload Ids: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	videos, err := youtube.GetFeedVideosJSON(VIDEO_LIMIT, uploadIds)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving videos as JSON: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(statusCodes.Success)
	w.Write(videos)
}

// PATCH - updates the provided feedName with the the provided newFeedName
func (s *State) renameFeedPATCH(w http.ResponseWriter, r *http.Request) {
	params := updateFeedParams{}

	userId, statusCode, err := unpackRequest(&params, r, s)
	if err != nil {
		log.Printf("in renameFeedPATCH(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedId, err := getUserFeedId(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in renameFeedPATCH(): error retrieving feedId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	err = updateFeedName(s, feedId, params.NewFeedName)
	if err != nil {
		log.Printf("in renameFeedPATCH(): error updating feed name: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	message := fmt.Sprintf("Feed name successfully updated from - %s - to new name - %s", params.FeedName, params.NewFeedName)
	writeResponseMessage(w, message, statusCodes.Success)
}

// DELETE - deletes the provided feed for the specific user
//
//	deletes all related feed-channels as a side effect
func (s *State) deleteFeedDELETE(w http.ResponseWriter, r *http.Request) {
	feedName := r.URL.Query().Get("feedName")

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in deleteFeedDELETE(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	err = deleteFeed(s, userId, feedName)
	if err != nil {
		log.Printf("in deleteFeedDELETE(): error deleting feed<%s>: %s", feedName, err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := fmt.Sprintf("Successfully deleted feed with name - %s", feedName)
	writeResponseMessage(w, message, statusCodes.Success)
}

// DELETE - deletes the provided channel for the specific user and feed
func (s *State) deleteChannelDELETE(w http.ResponseWriter, r *http.Request) {
	feedName := r.URL.Query().Get("feedName")
	channelHandle := r.URL.Query().Get("channelHandle")

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in deleteChannelDELETE(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	feedId, err := getUserFeedId(s, userId, feedName)
	if err != nil {
		log.Printf("in deleteChannelDELETE(): error retrieving feedId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	channelId, err := getChannelId(s, channelHandle)
	if err != nil {
		log.Printf("in deleteChannelDELETE(): error retrieving channelId: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	err = deleteFeedChannel(s, feedId, channelId)
	if err != nil {
		log.Printf("in deleteChannelDELETE(): error deleting channel: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := fmt.Sprintf("Successfully deleted channel with handle - %s", channelHandle)
	writeResponseMessage(w, message, statusCodes.Success)
}

// DELETE - deletes user from database including deleting all their feeds and channels
func (s *State) deleteUserDELETE(w http.ResponseWriter, r *http.Request) {

	userId, statusCode, err := unpackGetRequest(r, s)
	if err != nil {
		log.Printf("in deleteUserDELETE(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	err = deleteUser(s, userId)
	if err != nil {
		log.Printf("in deleteUserDELETE(): error deleting user from database: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := "Successfully deleted user from database"
	writeResponseMessage(w, message, statusCodes.Success)
}

// GET - hello world test function
func GetHelloWorld(c echo.Context) error {
	return c.String(http.StatusOK, "Hello World")
}

// OPTIONS - preflight for cors
func handleOPTIONS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Replace with specific domain later probably
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
}
