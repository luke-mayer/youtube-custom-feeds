package youtube

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type video struct {
	channelName  string
	title        string
	videoId      string
	thumbnailURL string
	publishedAt  time.Time
	videoURL     string
}

func getApiKey() string {
	return os.Getenv("YOUTUBE_API_KEY")
}

func GetService() (*youtube.Service, error) {
	ctx := context.Background()
	apiKey := getApiKey()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		newErr := fmt.Sprintf("Error creating YouTube client: %v", err)
		return nil, errors.New(newErr)
	}

	return service, nil
}

func GetChannelId(service *youtube.Service, channelName string) (bool, string, error) {
	call := service.Search.List([]string{"snippet"}).
		Q(channelName).
		Type("channel").
		MaxResults(10)

	response, err := call.Do()
	if err != nil {
		newErr := fmt.Sprintf("Error searching for channel - %v", err)
		return false, "", errors.New(newErr)
	}

	for _, item := range response.Items {
		if strings.EqualFold(item.Snippet.ChannelTitle, channelName) {
			return true, item.Snippet.ChannelId, nil
		}
	}

	returnString := fmt.Sprintf("Channel \"%s\" not found. Please make sure spelling is correct and exact.", channelName)
	return false, returnString, nil
}

func GetChannelURL(channelId string) string {
	channelURL := fmt.Sprintf("https://www.youtube.com/channel/%s", channelId)
	return channelURL
}

func responseToVideos(response *youtube.SearchListResponse) []video {
	recentVideos := []video{}
	for _, item := range response.Items {
		// Excludes live content
		if item.Snippet.LiveBroadcastContent == "live" || item.Snippet.LiveBroadcastContent == "upcoming" {
			continue
		}

		id := item.Id.VideoId
		url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)

		publishedAt, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		if err != nil {
			publishedAt = time.Time{} // time.Time zero value
			log.Printf("Issue parsting publishedAt to time.Time for videi with id: %s, error message: %s", id, err)
		}

		youtubeVideo := video{
			channelName:  item.Snippet.ChannelTitle,
			title:        item.Snippet.Title,
			videoId:      id,
			thumbnailURL: item.Snippet.Thumbnails.High.Url,
			publishedAt:  publishedAt,
			videoURL:     url,
		}
		recentVideos = append(recentVideos, youtubeVideo)
	}

	return recentVideos
}

func GetVideosByAmount(service *youtube.Service, num int64, channelId string) ([]video, error) {
	call := service.Search.List([]string{"snippet"}).
		ChannelId(channelId).
		MaxResults(num).
		Order("date").
		Type("video")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting videos: %s", err)
	}

	recentVideos := responseToVideos(response)

	return recentVideos, nil
}

func GetVideosByUploadDate(service *youtube.Service, days int64, channelId string) ([]video, error) {
	hours := -24 * days
	now := time.Now().UTC()
	publishedAfter := now.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)

	call := service.Search.List([]string{"snippet"}).
		ChannelId(channelId).
		PublishedAfter(publishedAfter).
		MaxResults(100).
		Order("date").
		Type("video")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting videos: %s", err)
	}

	recentVideos := responseToVideos(response)

	return recentVideos, nil
}

func PrintVideos(videos []video) {
	fmt.Println("----------------------")
	for _, v := range videos {
		fmt.Printf("Channel: %s\n", v.channelName)
		fmt.Printf("Video Title: %s\n", v.title)
		fmt.Printf("URL: %s\n", v.videoURL)
		fmt.Printf("Published: %v\n", v.publishedAt.Local().Format("2006-01-02 15:4:5"))
		fmt.Printf("Thumbnail URL: %s\n", v.thumbnailURL)
		fmt.Println("----------------------")
	}
}
