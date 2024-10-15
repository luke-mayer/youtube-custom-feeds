package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"slices"
	"sync"
	"time"

	"github.com/luke-mayer/youtube-custom-feeds/internal/config"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type video struct {
	ChannelName  string    `json:"channel"`
	Title        string    `json:"title"`
	VideoId      string    `json:"id"`
	ThumbnailURL string    `json:"thumbnailURL"`
	PublishedAt  time.Time `json:"publishedAt"`
	VideoURL     string    `json:"videoURL"`
}

func getApiKey() (string, error) {
	//return os.Getenv("YOUTUBE_API_KEY")
	apiKey, err := config.GetSecret(config.ApiKeySecretName)
	if err != nil {
		return "", fmt.Errorf("in getApiKey(): error retrieving youtube api key: %s", err)
	}

	return apiKey, nil
}

func getService() (*youtube.Service, error) {
	ctx := context.Background()
	apiKey, err := getApiKey()
	if err != nil {
		return nil, fmt.Errorf("in getService(): error retrieving apiKey: %s", err)
	}

	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		newErr := fmt.Sprintf("in getService(): error creating YouTube client:\n%v", err)
		return nil, errors.New(newErr)
	}

	return service, nil
}

func GetChannelIdUploadId(channelHandle string) (exisits bool, channelId string, uploadId string, err error) {
	service, err := getService()
	if err != nil {
		newErr := fmt.Errorf("in GetChannelUploadId(): error getting youtube service: %s", err)
		return false, "", "", newErr
	}

	call := service.Channels.List([]string{"id", "contentDetails"}).ForHandle(channelHandle)
	response, err := call.Do()
	if err != nil {
		newErr := fmt.Sprintf("in GetChannelUploadId(): error retrieving channel details by handle:\n%v", err)
		return false, "", "", errors.New(newErr)
	}

	if len(response.Items) == 0 {
		log.Printf("in GetChannelUploadId(): no channel found with handle: %s", channelHandle)
		return false, "", "", nil
	}

	channel := response.Items[0]
	channelId = channel.Id
	uploadId = channel.ContentDetails.RelatedPlaylists.Uploads

	if len(uploadId) < 5 {
		log.Printf("ChannelHandle<%s> uploadId<%s>\n", channelHandle, uploadId)
	}

	return true, channelId, uploadId, nil
}

// Might be unecessary
func GetChannelURL(channelId string) string {
	channelURL := fmt.Sprintf("https://www.youtube.com/channel/%s", channelId)
	return channelURL
}

func responseToVideos(response *youtube.PlaylistItemListResponse) []video {
	recentVideos := []video{}
	for _, item := range response.Items {
		id := item.Snippet.ResourceId.VideoId
		url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)

		publishedAt, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		if err != nil {
			publishedAt = time.Time{} // time.Time zero value
			log.Printf("in responseToVideos(): error parsing publishedAt to time.Time for video with id: %s, error message: %s", id, err)
		}

		youtubeVideo := video{
			ChannelName:  item.Snippet.ChannelTitle,
			Title:        item.Snippet.Title,
			VideoId:      id,
			ThumbnailURL: item.Snippet.Thumbnails.High.Url,
			PublishedAt:  publishedAt,
			VideoURL:     url,
		}
		recentVideos = append(recentVideos, youtubeVideo)
	}

	return recentVideos
}

func getChannelVideos(limit int64, uploadId string) ([]video, error) {
	service, err := getService()
	if err != nil {
		return []video{}, fmt.Errorf("in getChannelVideos(): error retrieving youtube service: %v", err)
	}

	call := service.PlaylistItems.List([]string{"snippet"}).PlaylistId(uploadId).MaxResults(limit)
	response, err := call.Do()
	if err != nil {
		return []video{}, fmt.Errorf("in getChannelVideos(): error retrieving videos from youtube API: uploadId<%v>: %v", uploadId, err)
	}

	channelVideos := responseToVideos(response)

	return channelVideos, nil
}

func getFeedVideos(limit int64, uploadIds []string) ([]video, []error) {
	var waitGroupChannels, waitGroupFinished sync.WaitGroup
	videoSliceChannel := make(chan []video, len(uploadIds))
	errorsChannel := make(chan error, len(uploadIds))
	allVideos := []video{}
	allErrors := []error{}

	for _, uploadId := range uploadIds {
		waitGroupChannels.Add(1)

		go func(id string) {
			defer waitGroupChannels.Done()

			videos, err := getChannelVideos(limit, uploadId)
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideos(): error retrieving videos for channel with uploadId: %s, : %v", uploadId, err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videoSliceChannel <- videos
		}(uploadId)
	}

	// Closes channel after all the videos are retrieved
	go func() {
		waitGroupChannels.Wait()
		close(videoSliceChannel)
		close(errorsChannel)
	}()

	waitGroupFinished.Add(1)
	go func() {
		for videos := range videoSliceChannel {
			allVideos = append(allVideos, videos...)
		}
		waitGroupFinished.Done()
	}()

	waitGroupFinished.Add(1)
	go func() {
		for err := range errorsChannel {
			allErrors = append(allErrors, err)
		}
		waitGroupFinished.Done()
	}()

	waitGroupFinished.Wait()
	sortByDate(allVideos)

	return allVideos, allErrors
}

// Returns a slice of videos as strings
func videosAsStrings(videos []video) []string {
	videoStrings := []string{}

	for _, v := range videos {
		videoString := "----------------------\n"
		videoString += fmt.Sprintf("Channel: %s\n", v.ChannelName)
		videoString += fmt.Sprintf("Video Title: %s\n", v.Title)
		videoString += fmt.Sprintf("URL: %s\n", v.VideoURL)
		videoString += fmt.Sprintf("Published: %v\n", v.PublishedAt.Local().Format("2006-01-02 15:4:5"))
		videoString += fmt.Sprintf("Thumbnail URL: %s\n", v.ThumbnailURL) // Maybe remove

		videoStrings = append(videoStrings, videoString)
	}

	return videoStrings
}

// Returns slice of JSON representation of videos
func videosAsJSON(videos []video) ([]byte, error) {

	videosJSON, err := json.Marshal(videos)
	if err != nil {
		newErr := fmt.Errorf("in videosAsJSON(): error Marshaling videos: %s", err)
		return []byte{}, newErr
	}

	return videosJSON, nil
}

// Retrieves videos for the feed in JSON format
func GetFeedVideosJSON(limit int64, uploadIds []string) ([]byte, error) {
	videos, errs := getFeedVideos(limit, uploadIds)
	if len(errs) > 0 {
		log.Println("in getFeedVideosJSON(): errors:")
		for _, err := range errs {
			log.Println(err)
		}
	}
	if len(videos) < 1 {
		return []byte{}, fmt.Errorf("in GetFeedVideosJSON(): error, no videos retrieved")
	}

	videosJSON, err := videosAsJSON(videos)
	if err != nil {
		return []byte{}, fmt.Errorf("in GetFeedVideosJSON(): error marshaling videos as JSON: %v", err)
	}

	return videosJSON, nil
}

// Prints videos - mainly for testing purposes
func printVideos(videos []video) {
	fmt.Println()
	fmt.Print(videosAsStrings(videos))
}

// sorts a slice of videos in descending order by publication date and time
func sortByDate(videos []video) {
	slices.SortFunc(videos, func(a, b video) int {
		return a.PublishedAt.Compare(b.PublishedAt) * -1
	})
}
