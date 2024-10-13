package youtube

import (
	"log"
	"testing"
)

func TestGetFeedVideosByDate(t *testing.T) {
	var allVideos []video
	channelIDs := []string{
		"UC7s6t5KCNwRkb7U_3-E1Tpw", "UCd9TUql8V7J-Xy1RNgA7MlQ", "UCUyeluBRhGPCW4rPe_UvBZQ",
	}

	allVideos, errs := getFeedVideosByDate(5, 10, channelIDs)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("error getting videos in TestGetFeedVideosByDate: %s\n", e)
		}
		t.Fail()
	}

	if len(allVideos) < 1 {
		log.Println("Not a single video recieved, most likely an issue")
		t.Fail()
	}

	printVideos(allVideos)
}

func TestGetFeedVideosByAmount(t *testing.T) {
	var allVideos []video
	channelIDs := []string{
		"UC7s6t5KCNwRkb7U_3-E1Tpw", "UCd9TUql8V7J-Xy1RNgA7MlQ", "UCUyeluBRhGPCW4rPe_UvBZQ",
	}

	allVideos, errs := getFeedVideosByAmount(5, channelIDs)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("error getting videos in TestGetFeedVideosByAmount: %s\n", e)
		}
		t.Fail()
	}

	if len(allVideos) != 20 {
		log.Printf("recieved %v videos, expected 20\n", len(allVideos))
		t.Fail()
	}
}
