package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/luke-mayer/youtube-custom-feeds/internal/youtube"
)

const PORT = ":8080"
const PREFIX = "/api/v1"
const VIDEO_LIMIT = 10

type StatusCodes struct {
	Success       int
	ErrRequest    int
	ErrDecoding   int
	ErrFirebaseId int
	ErrServer     int
	ErrState      int
	ErrUserId     int
	ErrMarshaling int
	ErrFeed       int
	ErrFeedExists int
}

var statusCodes = StatusCodes{
	Success:       200,
	ErrRequest:    400,
	ErrDecoding:   401,
	ErrFirebaseId: 402,
	ErrServer:     500,
	ErrState:      501,
	ErrUserId:     502,
	ErrMarshaling: 503,
	ErrFeed:       504,
	ErrFeedExists: 505,
}

var statusCodeMessages = map[int]string{
	statusCodes.Success:       "successful completion",
	statusCodes.ErrRequest:    "error: invalid request",
	statusCodes.ErrDecoding:   "error: decoding parameters",
	statusCodes.ErrFirebaseId: "error: firebase id issue",
	statusCodes.ErrServer:     "error: server issue",
	statusCodes.ErrState:      "error: issue initializing state",
	statusCodes.ErrUserId:     "error: retrieving user id",
	statusCodes.ErrMarshaling: "error: marshaling JSON",
	statusCodes.ErrFeed:       "error: creating feed",
	statusCodes.ErrFeedExists: "error: feed with provided name already exists for specified user",
}

type parameters interface {
	firebaseIdParams | feedParams | feedChannelParams | updateFeedParams
	getFirebaseId() string
}

type firebaseIdParams struct {
	FirebaseId string `json:"Firebase-Id"`
}

type feedParams struct {
	FirebaseId string `json:"firebaseId"`
	FeedName   string `json:"feedName"`
}

type feedChannelParams struct {
	FirebaseId    string `json:"firebaseId"`
	FeedName      string `json:"feedName"`
	ChannelHandle string `json:"channelHandle"`
}

type updateFeedParams struct {
	FirebaseId  string `json:"firebaseId"`
	FeedName    string `json:"feedName"`
	NewFeedName string `json:"newFeedName"`
}

func (p firebaseIdParams) getFirebaseId() string {
	return p.FirebaseId
}

func (p feedParams) getFirebaseId() string {
	return p.FirebaseId
}

func (p feedChannelParams) getFirebaseId() string {
	return p.FirebaseId
}

func (p updateFeedParams) getFirebaseId() string {
	return p.FirebaseId
}

// Used to unpack parameters from request and initialize the state and userId, returns statusCode if error
func unpackRequest[T parameters](params *T, r *http.Request, s *state) (int32, int, error) {
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

func unpackGetRequest(r *http.Request, s *state) (int32, int, error) {

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
func (s *state) login(w http.ResponseWriter, r *http.Request) {

	firebaseId := r.Header.Get("Firebase-ID")
	if firebaseId == "" {
		log.Println("in login(): error retireving firebaseId")
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFirebaseId], statusCodes.ErrFirebaseId)
		return
	}

	exists, err := s.db.ContainsUserByFirebaseId(context.Background(), firebaseId)
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

// POST - Creates a new feed
func (s *state) createFeedPOST(w http.ResponseWriter, r *http.Request) {
	params := feedParams{}

	userId, statusCode, err := unpackRequest(&params, r, s)
	if err != nil {
		log.Printf("in createFeedPOST(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponseMessage(w, statusCodeMessages[statusCode], statusCode)
		return
	}

	contains, _, err := createFeed(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in createFeedPOST(): error creating feed: %s", err)
		writeResponseMessage(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
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
func (s *state) addChannelPOST(w http.ResponseWriter, r *http.Request) {
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
func (s *state) getFeedsGET(w http.ResponseWriter, r *http.Request) {

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
func (s *state) getChannelsGET(w http.ResponseWriter, r *http.Request) {

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
func (s *state) getVideosGET(w http.ResponseWriter, r *http.Request) {

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
func (s *state) renameFeedPATCH(w http.ResponseWriter, r *http.Request) {
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
func (s *state) deleteFeedDELETE(w http.ResponseWriter, r *http.Request) {
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
func (s *state) deleteChannelDELETE(w http.ResponseWriter, r *http.Request) {
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
func (s *state) deleteUserDELETE(w http.ResponseWriter, r *http.Request) {

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

// OPTIONS - preflight for cors
func handleOPTIONS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Replace with specific domain later probably
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
}

func main() {
	s, err := getState()
	if err != nil {
		newErr := fmt.Sprintf("Error initializing state: %s", err)
		log.Fatal(newErr)
	}

	router := mux.NewRouter()
	api := router.PathPrefix(PREFIX).Subrouter()
	api.HandleFunc("/login", s.login).Methods(http.MethodPost)
	api.HandleFunc("/feed", s.createFeedPOST).Methods(http.MethodPost)
	api.HandleFunc("/channel", s.addChannelPOST).Methods(http.MethodPost)
	api.HandleFunc("/feeds", s.getFeedsGET).Methods(http.MethodGet)
	api.HandleFunc("/channels", s.getChannelsGET).Methods(http.MethodGet)
	api.HandleFunc("/videos", s.getVideosGET).Methods(http.MethodGet)
	api.HandleFunc("/feed", s.renameFeedPATCH).Methods(http.MethodPatch)
	api.HandleFunc("/feed", s.deleteFeedDELETE).Methods(http.MethodDelete)
	api.HandleFunc("/channel", s.deleteChannelDELETE).Methods(http.MethodDelete)
	api.HandleFunc("/user", s.deleteUserDELETE).Methods(http.MethodDelete)
	api.HandleFunc("/login", handleOPTIONS).Methods(http.MethodOptions)

	log.Fatal(http.ListenAndServe(PORT, router))
}
