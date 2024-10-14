package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

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

func getApiKey() string {
	return os.Getenv("YOUTUBE_API_KEY")
}

func getService() (*youtube.Service, error) {
	ctx := context.Background()
	apiKey := getApiKey()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		newErr := fmt.Sprintf("in getService(): error creating YouTube client:\n%v", err)
		return nil, errors.New(newErr)
	}

	return service, nil
}

func GetChannelId(channelName string) (bool, string, error) {
	service, err := getService()
	if err != nil {
		newErr := fmt.Errorf("in getChannelId(): error getting youtube service: %s", err)
		return false, "", newErr
	}

	call := service.Search.List([]string{"snippet"}).
		Q(channelName).
		Type("channel").
		MaxResults(10)

	response, err := call.Do()
	if err != nil {
		newErr := fmt.Sprintf("in GetChannelId(): error searching for channel:\n%v", err)
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

// Might be unecessary
func GetChannelURL(channelId string) string {
	channelURL := fmt.Sprintf("https://www.youtube.com/channel/%s", channelId)
	return channelURL
}

// Returns url to get xml of recent videos from provided channel
func getChannelRssUrl(channelId string) string {
	channelURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelId)
	return channelURL
}

// Parses entries from an rssFeed and returns them as a video slice
func getFeedAsVideos(limit int, feed *rssFeed) ([]video, error) {
	videos := []video{}
	channelName := html.UnescapeString(feed.Title)

	for i, entry := range feed.Entry {
		if i >= limit {
			break
		}

		publishedAt, err := time.Parse(time.RFC3339, entry.Published)
		if err != nil {
			return []video{}, fmt.Errorf("in getFeedAsVideos(): error parsing publishedAt time: %s", err)
		}

		vid := video{
			ChannelName:  channelName,
			Title:        html.UnescapeString(entry.Title),
			VideoId:      entry.VideoId,
			ThumbnailURL: entry.MediaGroup.MediaThumbnail.Url,
			PublishedAt:  publishedAt,
			VideoURL:     entry.Link.Href,
		}

		videos = append(videos, vid)
	}

	return videos, nil
}

// Returns recent videos from a youtube with the provided channel Id
func getVideosRSS(limit int, channelId string) ([]video, error) {
	rssUrl := getChannelRssUrl(channelId)

	feed, err := fetchRSSFeed(context.Background(), rssUrl)
	if err != nil {
		return []video{}, fmt.Errorf("in getVideosRSS(): error fetching the rssFeed: %s", err)
	}

	videos, err := getFeedAsVideos(limit, feed)
	if err != nil {
		return []video{}, fmt.Errorf("in getVideosRSS(): error converting rssFeed to []video: %s", err)
	}

	return videos, nil
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

// ISSUE: live videos could result in less then num amount of videos returned for the channel
func getVideosByAmount(service *youtube.Service, num int64, channelId string) ([]video, error) {
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

func getVideosByUploadDate(service *youtube.Service, days, limit int64, channelId string) ([]video, error) {
	hours := -24 * days
	now := time.Now().UTC()
	publishedAfter := now.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339)

	call := service.Search.List([]string{"snippet"}).
		ChannelId(channelId).
		PublishedAfter(publishedAfter).
		MaxResults(limit).
		Order("date").
		Type("video")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting videos: %s", err)
	}

	recentVideos := responseToVideos(response)

	return recentVideos, nil
}

// Concurrently retrives videos by upload date for all channels
func getFeedVideosByDate(days, limit int64, channelIDs []string) ([]video, []error) {
	var waitGroupChannels, waitGroupFinished sync.WaitGroup
	videoSliceChannel := make(chan []video, len(channelIDs))
	errorsChannel := make(chan error, 2*len(channelIDs))
	allVideos := []video{}
	allErrors := []error{}

	for _, channelID := range channelIDs {
		waitGroupChannels.Add(1)

		go func(id string) {
			defer waitGroupChannels.Done()

			service, err := getService()
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideosByDate: error retrieving youtube service:\n%s", err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videos, err := getVideosByUploadDate(service, days, limit, channelID)
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideosByDate: error retrieving videos for channel with ID: %s, :\n%s", channelID, err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videoSliceChannel <- videos
		}(channelID)
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
	sortByDate(allVideos) // sorting into descending order by publication date and time

	return allVideos, allErrors
}

// Concurrently retrives videos by upload date for all channels
func getFeedVideosByAmount(num int64, channelIDs []string) ([]video, []error) {
	var waitGroupChannels, waitGroupFinished sync.WaitGroup
	videoSliceChannel := make(chan []video, len(channelIDs))
	errorsChannel := make(chan error, 2*len(channelIDs))
	allVideos := []video{}
	allErrors := []error{}

	for _, channelID := range channelIDs {
		waitGroupChannels.Add(1)

		go func(id string) {
			defer waitGroupChannels.Done()

			service, err := getService()
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideosByAmount: error retrieving youtube service:\n%s", err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videos, err := getVideosByAmount(service, num, channelID)
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideosByAmount: error retrieving videos for channel with ID: %s, :\n%s", channelID, err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videoSliceChannel <- videos
		}(channelID)
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
	return allVideos, allErrors
}

func getFeedVideos(limit int, channelIDs []string) ([]video, []error) {
	var waitGroupChannels, waitGroupFinished sync.WaitGroup
	videoSliceChannel := make(chan []video, len(channelIDs))
	errorsChannel := make(chan error, 2*len(channelIDs))
	allVideos := []video{}
	allErrors := []error{}

	for _, channelID := range channelIDs {
		waitGroupChannels.Add(1)

		go func(id string) {
			defer waitGroupChannels.Done()

			videos, err := getVideosRSS(limit, channelID)
			if err != nil {
				newErr := fmt.Errorf("in getFeedVideos: error retrieving videos for channel with ID: %s, :%s", channelID, err)
				log.Printf("%v\n", newErr)
				errorsChannel <- newErr
			}

			videoSliceChannel <- videos
		}(channelID)
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
	sortByDate(allVideos) // sorting into descending order by publication date and time

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
func videosAsJSON(videos []video) ([][]byte, error) {
	videosJSON := [][]byte{}

	for _, v := range videos {
		vJSON, err := json.Marshal(v)
		if err != nil {
			newErr := fmt.Errorf("in videosAsJSON(): error Marshaling video: \n%s", err)
			return videosJSON, newErr
		}

		videosJSON = append(videosJSON, vJSON)
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
