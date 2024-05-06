package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"freder.rss-fetcher/utils"
	"github.com/mmcdole/gofeed"
)

const feedsFilePath = "../../feeds.json"
const lastCheckTimeFilePath = "./last-check.txt"
const maxConcurrency = 3

type result struct {
	name  string
	items []*gofeed.Item
}

func listFeeds() {
	feedsMap := utils.ReadFeedUrls(feedsFilePath)
	for name, url := range feedsMap {
		fmt.Printf("%s: %s\n", name, url)
	}
}

func checkFeeds() {
	now := time.Now()
	lastCheckTime := utils.GetLastCheckTime(lastCheckTimeFilePath)
	// TODO: remove — for testing only
	// lastCheckTime = time.Date(2023, 6, 1, 0, 0, 0, 0, time.Local)

	// write current time to file
	utils.UpdateLastCheckTimeFile(lastCheckTimeFilePath, &now)

	var wg sync.WaitGroup

	// create a buffered channel (which acts as a semaphone)
	// to control concurrency
	// struct{} is an empty struct, which takes up no memory
	sem := make(chan struct{}, maxConcurrency)

	feedsMap := utils.ReadFeedUrls(feedsFilePath)
	results := make(chan result, len(feedsMap))

	for name, url := range feedsMap {
		wg.Add(1)
		go func(name string, url string) {
			sem <- struct{}{} // blocks if channel is full
			defer wg.Done()

			content := utils.RequestFeed(url)
			fmt.Print(".") // progress indicator

			// parse feed
			parser := gofeed.NewParser()
			feed, err := parser.ParseString(content)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error parsing feed:", err)
				return
			}

			filtered := make([]*gofeed.Item, 0)
			for _, item := range feed.Items {
				// rss and atom feeds have different date fields
				if item.UpdatedParsed == nil {
					item.UpdatedParsed = item.PublishedParsed
				}

				if item.UpdatedParsed.After(lastCheckTime) {
					filtered = append(filtered, item)
				}
			}

			results <- result{name, filtered}
			<-sem // release
		}(name, url)
	}

	wg.Wait()
	close(results)
	fmt.Println()

	newItemsCount := 0
	for result := range results {
		name := result.name
		items := result.items

		c := len(items)
		if c == 0 {
			continue
		}

		newItemsCount += c
		fmt.Println()
		fmt.Println(name + ": " + fmt.Sprint(c))

		// reverse sort by date
		sort.SliceStable(items, func(i, j int) bool {
			a := *items[i].UpdatedParsed
			b := *items[j].UpdatedParsed
			return a.After(b)
		})

		for _, item := range items {
			timestamp := item.UpdatedParsed.Format(time.RFC3339)
			timestamp = strings.Split(timestamp, "T")[0]
			fmt.Println(
				"-",
				fmt.Sprintf("(%s)", timestamp),
				item.Title,
			)
		}
	}

	if newItemsCount == 0 {
		fmt.Println("No new items")
	}
}

func printUsageAndExit() {
	// name := os.Args[0]
	name := "<this>"
	fmt.Println("Usage: ./" + name + " list|check")
	os.Exit(1)
}

func main() {
	args := os.Args[1:] // skip program name
	if len(args) != 1 {
		printUsageAndExit()
	}

	switch args[0] {
	case "list":
		listFeeds()
	case "check":
		checkFeeds()
	default:
		printUsageAndExit()
	}
}
