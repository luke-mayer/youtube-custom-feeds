package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const PORT = ":8080"
const PREFIX = "/api/v1"

// POST
func createFeedPOST(w http.ResponseWriter, req *http.Request) {

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
