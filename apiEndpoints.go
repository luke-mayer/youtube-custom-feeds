package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/luke-mayer/youtube-custom-feeds/internal/config"
	"github.com/luke-mayer/youtube-custom-feeds/internal/youtube"
	"google.golang.org/api/idtoken"
)

const PORT = ":8080"
const PREFIX = "/api/v1"
const VIDEO_LIMIT = 10

type StatusCodes struct {
	Success       int
	ErrRequest    int
	ErrDecoding   int
	ErrIdToken    int
	ErrServer     int
	ErrState      int
	ErrUserId     int
	ErrMarshaling int
	ErrFeed       int
}

var statusCodes = StatusCodes{
	Success:       200,
	ErrRequest:    400,
	ErrDecoding:   401,
	ErrIdToken:    402,
	ErrServer:     500,
	ErrState:      501,
	ErrUserId:     502,
	ErrMarshaling: 503,
	ErrFeed:       504,
}

var statusCodeMessages = map[int]string{
	statusCodes.Success:       "successful completion",
	statusCodes.ErrRequest:    "error: invalid request",
	statusCodes.ErrDecoding:   "error: decoding parameters",
	statusCodes.ErrIdToken:    "error: idToken issue",
	statusCodes.ErrServer:     "error: server issue",
	statusCodes.ErrState:      "error: initializing state",
	statusCodes.ErrUserId:     "error: retrieving user id",
	statusCodes.ErrMarshaling: "error: marshaling JSON",
	statusCodes.ErrFeed:       "error: creating feed",
}

type parameters interface {
	idTokenParams | feedParams | addChannelParams
	getIdToken() string
}

type idTokenParams struct {
	IdToken string `json:"idToken"`
}

type feedParams struct {
	IdToken  string `json:"idToken"`
	FeedName string `json:"feedName"`
}

type addChannelParams struct {
	IdToken       string `json:"idToken"`
	FeedName      string `json:"feedName"`
	ChannelHandle string `json:"channelHandle"`
}

func (p idTokenParams) getIdToken() string {
	return p.IdToken
}

func (p feedParams) getIdToken() string {
	return p.IdToken
}

func (p addChannelParams) getIdToken() string {
	return p.IdToken
}

// Used to unpack parameters from request and initialize the state and userId, returns statusCode if error
func unpackRequest[T parameters](params *T, r *http.Request) (*state, int32, int, error) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(params)
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error decoding parameters: %s", err)
		return &state{}, 0, statusCodes.ErrDecoding, newErr
	}

	s, err := getState()
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error retreiving state: %s", err)
		return &state{}, 0, statusCodes.ErrState, newErr
	}

	googleId, err := validateIdToken((*params).getIdToken())
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error validating idToken: %s", err)
		return &state{}, 0, statusCodes.ErrIdToken, newErr
	}

	userId, err := getUserId(s, googleId)
	if err != nil {
		newErr := fmt.Errorf("in unpackRequest(): error retrieving userId: %s", err)
		return &state{}, 0, statusCodes.ErrUserId, newErr
	}

	return s, userId, statusCodes.Success, nil
}

// Used to write error messages to response
func writeResponse(w http.ResponseWriter, message string, statusCode int) {
	type returnVals struct {
		Message string `json:"message"`
	}
	resBody := returnVals{
		Message: message,
	}

	data, err := json.Marshal(resBody)
	if err != nil {
		log.Printf("in writeResponse(): error marshaling JSON: %s", err)
		w.WriteHeader(statusCodes.ErrMarshaling)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// validates OAuth2 ID token and returns googleId (sum field)
func validateIdToken(token string) (string, error) {
	clientId, err := config.GetClientId()
	if err != nil {
		return "", fmt.Errorf("in validateIdToken(): error retrieiving client id: %s", err)
	}

	payload, err := idtoken.Validate(context.Background(), token, clientId)
	if err != nil {
		return "", fmt.Errorf("in validateIdToken(): error validating token: %s", err)
	}

	googleId, ok := payload.Claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("in validateToken(): error extracting googleId from token: %s", err)
	}

	return googleId, nil
}

// ------------------------ //
//		API ENDPOINTS		//
// ------------------------ //

// POST - Checks if user is in the database. If not, creates a new user
func login(w http.ResponseWriter, r *http.Request) {
	params := idTokenParams{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrDecoding], err)
		log.Println(errMessage)
		writeResponse(w, statusCodeMessages[statusCodes.ErrDecoding], statusCodes.ErrDecoding)
		return
	}

	s, err := getState()
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrState], err)
		log.Println(errMessage)
		writeResponse(w, statusCodeMessages[statusCodes.ErrState], statusCodes.ErrState)
		return
	}

	googleId, err := validateIdToken(params.getIdToken())
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrIdToken], err)
		log.Println(errMessage)
		writeResponse(w, statusCodeMessages[statusCodes.ErrIdToken], statusCodes.ErrIdToken)
		return
	}

	exists, err := s.db.ContainsUserByGoogleId(context.Background(), googleId)
	if err != nil {
		errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrServer], err)
		log.Println(errMessage)
		writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := statusCodeMessages[statusCodes.Success]

	if !exists {
		err := registerUser(s, googleId)
		if err != nil {
			errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrServer], err)
			log.Println(errMessage)
			writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
			return
		}
		message = "user did not exist in database - created new user"
	}

	writeResponse(w, message, statusCodes.Success)
}

// POST - Creates a new feed
func createFeedPOST(w http.ResponseWriter, r *http.Request) {
	params := feedParams{}

	s, userId, statusCode, err := unpackRequest(&params, r)
	if err != nil {
		log.Printf("in createFeedPOST(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponse(w, statusCodeMessages[statusCode], statusCode)
	}

	_, err = createFeed(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in createFeedPOST(): error creating feed: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	message := fmt.Sprintf("Feed - %s - successfully created", params.FeedName)
	writeResponse(w, message, statusCodes.Success)
}

// POST - adds the youtube channel to the user's indicated field
func addChannelPOST(w http.ResponseWriter, r *http.Request) {
	params := addChannelParams{}

	s, userId, statusCode, err := unpackRequest(&params, r)
	if err != nil {
		log.Printf("in addChannelPOST(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponse(w, statusCodeMessages[statusCode], statusCode)
	}

	feedId, err := getUserFeedId(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in addChannelPOST(): error retrieving feedId: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	err = addChannelToFeed(s, feedId, params.ChannelHandle)
	if err != nil {
		log.Printf("in addChannelPOST(): error adding channel to feed: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	message := fmt.Sprintf("Channel - %s - successfully added to feed - %s", params.ChannelHandle, params.FeedName)
	writeResponse(w, message, statusCodes.Success)
}

// GET - retrieves youtube videos for the provided
func getVideosGET(w http.ResponseWriter, r *http.Request) {
	params := feedParams{}

	s, userId, statusCode, err := unpackRequest(&params, r)
	if err != nil {
		log.Printf("in getVideosGET(): %s: %s", statusCodeMessages[statusCode], err)
		writeResponse(w, statusCodeMessages[statusCode], statusCode)
	}

	feedId, err := getUserFeedId(s, userId, params.FeedName)
	if err != nil {
		log.Printf("in addChannelPOST(): error retrieving feedId: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrFeed], statusCodes.ErrFeed)
		return
	}

	channelIds, err := getAllFeedChannels(s, feedId)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving feed channel Ids: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	uploadIds, err := getAllUploadIds(s, channelIds)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving feed upload Ids: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	videos, err := youtube.GetFeedVideosJSON(VIDEO_LIMIT, uploadIds)
	if err != nil {
		log.Printf("in getVideosGET(): error retrieving videos as JSON: %s", err)
		writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCodes.Success)
	w.Write(videos)
}

func renameFeedPATCH(w http.ResponseWriter, r *http.Request) {

}

func deleteFeedDELETE(w http.ResponseWriter, r *http.Request) {

}

func deleteChannelDELETE(w http.ResponseWriter, r *http.Request) {

}

func main() {
	router := mux.NewRouter()
	api := router.PathPrefix(PREFIX).Subrouter()
	api.HandleFunc("/login", login).Methods(http.MethodPost)
	api.HandleFunc("/feed", createFeedPOST).Methods(http.MethodPost)
	api.HandleFunc("/channel", addChannelPOST).Methods(http.MethodPost)
	api.HandleFunc("/videos", getVideosGET).Methods(http.MethodGet)
	api.HandleFunc("", renameFeedPATCH).Methods(http.MethodPatch)
	api.HandleFunc("", deleteFeedDELETE).Methods(http.MethodDelete)
	api.HandleFunc("", deleteChannelDELETE).Methods(http.MethodDelete)

	log.Fatal(http.ListenAndServe(PORT, router))
}
