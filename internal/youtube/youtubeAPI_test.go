package youtube

import (
	"log"
	"testing"
)

func TestGetFeedVideos(t *testing.T) {
	var allVideos []video
	channelIDs := []string{
		"UC7s6t5KCNwRkb7U_3-E1Tpw", "UCd9TUql8V7J-Xy1RNgA7MlQ", "UCUyeluBRhGPCW4rPe_UvBZQ",
	}

	allVideos, errs := getFeedVideos(5, channelIDs)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("error getting videos in TestGetFeedVideos: %s\n", e)
		}
		t.Fail()
	}

	if len(allVideos) < 1 {
		log.Println("Not a single video recieved, most likely an issue")
		t.Fail()
	}

	printVideos(allVideos)
}
