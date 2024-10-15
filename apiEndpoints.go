package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/api/idtoken"
)

const PORT = ":8080"
const PREFIX = "/api/v1"
const CLIENT_ID = "CLIENT_ID_NEED_TO_DO"

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
	idTokenParams | feedParams
	getIdToken() string
}

type idTokenParams struct {
	IdToken string `json:"idToken"`
}

type feedParams struct {
	IdToken  string `json:"idToken"`
	FeedName string `json:"feedName"`
}

func (p idTokenParams) getIdToken() string {
	return p.IdToken
}

func (p feedParams) getIdToken() string {
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
	payload, err := idtoken.Validate(context.Background(), token, CLIENT_ID)
	if err != nil {
		return "", fmt.Errorf("in validateIdToken(): error validating token: %s", err)
	}

	googleId, ok := payload.Claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("in validateToken(): error extracting googleId from token: %s", err)
	}

	return googleId, nil
}

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

	if !exists {
		err := registerUser(s, googleId)
		if err != nil {
			errMessage := fmt.Sprintf("in login(): %s: %s", statusCodeMessages[statusCodes.ErrServer], err)
			log.Println(errMessage)
			writeResponse(w, statusCodeMessages[statusCodes.ErrServer], statusCodes.ErrServer)
			return
		}
	}
}

// POST
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
	writeResponse(w, message, 200)
}

func addChannelPOST(w http.ResponseWriter, req *http.Request) {

}

func getVideosGET(w http.ResponseWriter, req *http.Request) {

}

func renameFeedPATCH(w http.ResponseWriter, req *http.Request) {

}

func deleteFeedDELETE(w http.ResponseWriter, req *http.Request) {

}

func deleteChannelDELETE(w http.ResponseWriter, req *http.Request) {

}

func main() {
	router := mux.NewRouter()
	api := router.PathPrefix(PREFIX).Subrouter()

	api.HandleFunc("", createFeedPOST).Methods(http.MethodPost)
	api.HandleFunc("", addChannelPOST).Methods(http.MethodPost)
	api.HandleFunc("", getVideosGET).Methods(http.MethodGet)
	api.HandleFunc("", renameFeedPATCH).Methods(http.MethodPatch)
	api.HandleFunc("", deleteFeedDELETE).Methods(http.MethodDelete)
	api.HandleFunc("", deleteChannelDELETE).Methods(http.MethodDelete)

	log.Fatal(http.ListenAndServe(PORT, router))
}
