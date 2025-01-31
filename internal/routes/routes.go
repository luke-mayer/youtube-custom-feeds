package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/luke-mayer/youtube-custom-feeds/internal/logic"
)

func AttachRoutes(e *echo.Echo, s *logic.State) {
	e.GET("/", logic.GetHelloWorld)
	e.GET("/login", s.Login)
	e.POST("/feed", s.CreateFeedHandler)
}

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
