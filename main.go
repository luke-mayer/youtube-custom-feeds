package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/luke-mayer/youtube-custom-feeds/internal/logic"
	"github.com/luke-mayer/youtube-custom-feeds/internal/routes"
)

/*
func main() {
	s, err := logic.GetState()
	if err != nil {
		newErr := fmt.Sprintf("Error initializing state: %s", err)
		log.Fatal(newErr)
	}

	router := mux.NewRouter()
	api := router.PathPrefix(PREFIX).Subrouter()
	api.HandleFunc("/", getHelloWorld).Methods(http.MethodGet)
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
	log.Printf("YCF server running on port %s", PORT)

	log.Fatal(http.ListenAndServe(PORT, router))
}
*/

func main() {
	// TODO make an ENV variable
	port := ":8000"

	dockerized, err := strconv.ParseBool(os.Getenv("DOCKERIZED"))
	if !dockerized || err != nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error injecting environment variables: %s\n", err)
		}
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	e.Logger.SetLevel(2)

	e.Logger.Info("Starting Echo server...")
	log.Println("this is log")

	s, err := logic.GetState()
	if err != nil {
		log.Fatalf("Error initializing state: %s", err)
	}

	routes.AttachRoutes(e, s)

	e.Logger.Fatal(e.Start(port))
}
