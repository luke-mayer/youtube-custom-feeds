package youtube

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type video struct {
	channelName  string
	title        string
	videoId      string
	thumbnailURL string
	published_at string
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

	return false, "", nil
}

func GetChannelURL(channelId string) string {
	channelURL := fmt.Sprintf("https://www.youtube.com/channel/%s", channelId)
	return channelURL
}

func GetRecentVideos(service *youtube.Service, num int64, channelId string) ([]video, error) {
	call := service.Search.List([]string{"snippet"}).
		ChannelId(channelId).
		MaxResults(num).
		Order("date").
		Type("video")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting videos: %s", err)
	}

	recentVideos := []video{}
	for _, item := range response.Items {
		id := item.Id.VideoId
		url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)
		youtubeVideo := video{
			channelName:  item.Snippet.ChannelTitle,
			title:        item.Snippet.Title,
			videoId:      id,
			thumbnailURL: item.Snippet.Thumbnails.High.Url,
			published_at: item.Snippet.PublishedAt,
			videoURL:     url,
		}
		recentVideos = append(recentVideos, youtubeVideo)
	}

	return recentVideos, nil
}

func PrintVideos(videos []video) {
	fmt.Println("----------------------")
	for _, v := range videos {
		fmt.Printf("Channel: %s\n", v.channelName)
		fmt.Printf("Video Title: %s\n", v.title)
		fmt.Printf("URL: %s\n", v.videoURL)
		fmt.Printf("Published: %s\n", v.published_at)
		fmt.Printf("Thumbnail URL: %s\n", v.thumbnailURL)
		fmt.Println("----------------------")
	}
}
