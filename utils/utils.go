package utils

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

func RequestFeed(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("filed to fetch feed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}

	return string(body), nil
}

func RequestAndParseFeed(url string) (*gofeed.Feed, error) {
	content, err := RequestFeed(url)
	if err != nil {
		return nil, err
	}
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(content)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func FilterByDate(items []*gofeed.Item, lastCheckTime time.Time) []*gofeed.Item {
	newItems := make([]*gofeed.Item, 0)
	for _, item := range items {
		// rss and atom feeds have different date fields
		if item.UpdatedParsed == nil {
			item.UpdatedParsed = item.PublishedParsed
		}

		// only keep new items
		if item.UpdatedParsed.After(lastCheckTime) {
			newItems = append(newItems, item)
		}
	}
	return newItems
}
