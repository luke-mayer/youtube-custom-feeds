package youtube

import (
	"log"
	"testing"
)

func TestGetFeedVideos(t *testing.T) {
	var allVideos []video
	uploadIds := []string{}
	channelHandles := []string{
		"@theonlyzanny", "@ThePrimeTimeagen", "@ColbertLateShow",
	}

	/*
		channelIDs := []string{
			"UC7s6t5KCNwRkb7U_3-E1Tpw", "UCd9TUql8V7J-Xy1RNgA7MlQ", "UCUyeluBRhGPCW4rPe_UvBZQ",
		}
	*/

	for _, handle := range channelHandles {
		exists, _, uploadId, err := GetChannelIdUploadId(handle)
		log.Println(uploadId)
		if err != nil {
			log.Printf("in TestGetFeedVideos: error getting channelId uploadId: %v", err)
			t.Fail()
			continue
		}
		if !exists {
			log.Printf("in TestGetFeedVideos: GetChannelIdUploadId came back false: handle<%s> uploadId<%s> %v", handle, uploadId, err)
		}
		uploadIds = append(uploadIds, uploadId)
	}

	allVideos, errs := getFeedVideos(3, uploadIds)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("error getting videos in TestGetFeedVideos: %v", e)
		}
		t.Fail()
	}

	if len(allVideos) < 1 {
		log.Println("Not a single video recieved, most likely an issue")
		t.Fail()
	}

	printVideos(allVideos)
}
