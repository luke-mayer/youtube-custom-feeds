package main

/*
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

/*

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

*/
