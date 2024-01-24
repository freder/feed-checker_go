package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

const feedsFilePath = "../../feeds.json"
const lastCheckTimeFilePath = "./last-check.txt"
const maxConcurrency = 3

func listFeeds() {
	feedsMap := readFeedUrls()
	for name, url := range feedsMap {
		fmt.Printf("%s: %s\n", name, url)
	}
}

func checkFeeds() {
	now := time.Now()
	lastCheckTime := getLastCheckTime()
	// TODO: remove — for testing only
	// lastCheckTime = time.Date(2023, 6, 1, 0, 0, 0, 0, time.Local)

	// write current time to file
	updateLastCheckTimeFile(&now)

	var wg sync.WaitGroup

	// create a buffered channel (which acts as a semaphone)
	// to control concurrency
	// struct{} is an empty struct, which takes up no memory
	sem := make(chan struct{}, maxConcurrency)

	feedsMap := readFeedUrls()
	var results = make(map[string][]*gofeed.Item)

	for name, url := range feedsMap {
		wg.Add(1)
		go func(name string, url string) {
			sem <- struct{}{} // blocks if channel is full
			defer wg.Done()

			content := requestFeed(url)
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
			results[name] = filtered

			<-sem // release
		}(name, url)
	}

	wg.Wait()
	fmt.Println()

	newItemsCount := 0
	for name, items := range results {
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

func printUsage() {
	// name := os.Args[0]
	name := "<this>"
	fmt.Println("Usage: ./" + name + " list|check")
}

func main() {
	args := os.Args[1:] // skip program name
	if len(args) != 1 {
		printUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		listFeeds()
	case "check":
		checkFeeds()
	default:
		printUsage()
		os.Exit(1)
	}
}
