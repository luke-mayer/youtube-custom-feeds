package main

/*

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
)

type rssFeed struct {
	Title string     `xml:"title"`
	Entry []rssEntry `xml:"entry"`
}

type rssEntry struct {
	Title   string `xml:"title"`
	VideoId string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Published  string     `xml:"published"`
	MediaGroup mediaGroup `xml:"http://search.yahoo.com/mrss/ group"`
}

type mediaGroup struct {
	MediaThumbnail mediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

type mediaThumbnail struct {
	Url string `xml:"url,attr"`
}

func fetchRSSFeed(ctx context.Context, rssUrl string) (*rssFeed, error) {
	feed := &rssFeed{}
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", rssUrl, nil)
	if err != nil {
		return feed, fmt.Errorf("in fetchRSSFeed(): error creating Request: %s", err)
	}

	req.Header.Add("User-Agent", "youtube-custom-feeds")

	res, err := client.Do(req)
	if err != nil {
		return feed, fmt.Errorf("in fetchRSSFeed(): error retrieving response: %s", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Println("in fetchRSSFeed(): error closing response body: ", err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return feed, fmt.Errorf("in fetchRSSFeed(): error reading response body: %s", err)
	}

	err = xml.Unmarshal(body, feed)
	if err != nil {
		return &rssFeed{}, fmt.Errorf("in fetchRSSFeed(): error unmarshaling xml body: %s", err)
	}

	return feed, nil
}
*/
