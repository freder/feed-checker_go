package main

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"freder.feed-checker/database"
	"freder.feed-checker/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
)

const dbFilePath = "./db.sqlite"
const maxConcurrency = 5

type feedFetchResult struct {
	name  string
	items []*gofeed.Item
}

func addFeed(feedUrl string) {
	_, err := url.Parse(feedUrl)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		os.Exit(1)
	}

	db := database.OpenDb(dbFilePath)
	defer db.Close()
	database.InsertFeed(db, feedUrl)
}

func removeFeed(feedUrl string) {
	db := database.OpenDb(dbFilePath)
	defer db.Close()
	database.RemoveFeed(db, feedUrl)
}

func listFeeds() {
	db := database.OpenDb(dbFilePath)
	defer db.Close()
	feedsMap := database.GetFeedUrls(db)
	for name, url := range feedsMap {
		fmt.Printf("%s: %s\n", name, url)
	}
}

func checkFeeds() {
	db := database.OpenDb(dbFilePath)
	defer db.Close()

	var wg sync.WaitGroup

	// create a buffered channel (which acts as a semaphone)
	// to control concurrency
	sem := make(
		chan struct{}, // empty struct takes up no memory
		maxConcurrency,
	)

	feeds := database.GetFeeds(db)
	results := make(chan *feedFetchResult, len(feeds))

	for _, feed := range feeds {
		wg.Add(1)
		go func(name string, url string, lastCheck string) {
			sem <- struct{}{} // blocks if channel is full
			defer wg.Done()
			defer (func() {
				<-sem // release
			})()

			fmt.Print(".") // progress indicator

			feed, err := utils.RequestAndParseFeed(url)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				results <- nil
				return
			}

			// update last check time
			database.UpdateFeedLastCheck(db, url, time.Now())

			var lastCheckTime time.Time
			if lastCheck == "" { // first time checking
				lastCheckTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
			} else {
				lastCheckTime, err = time.Parse(time.RFC3339, lastCheck)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					results <- nil
					return
				}
			}

			newItems := utils.FilterByDate(feed, lastCheckTime)

			results <- &feedFetchResult{name, newItems}
		}(feed.Title, feed.Url, feed.LastCheck)
	}

	wg.Wait()
	close(results)
	fmt.Println()

	newItemsCount := 0
	for result := range results {
		if result == nil {
			continue
		}

		items := result.items
		count := len(items)
		if count == 0 {
			continue
		}

		newItemsCount += count
		fmt.Println()
		fmt.Println(result.name + ": " + fmt.Sprint(count))

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
			fmt.Println(" ", item.Link)
		}
	}

	if newItemsCount == 0 {
		fmt.Println("No new items")
	}
}

func printUsageAndExit() {
	executable := os.Args[0]
	lines := []string{
		executable + " add <feed-url>",
		executable + " remove <feed-url>",
		executable + " list",
		executable + " check",
	}
	fmt.Println("Usage:\n" + strings.Join(lines, "\n"))
	os.Exit(1)
}

func main() {
	args := os.Args[1:] // skip program name
	if len(args) == 1 {
		switch args[0] {
		case "list":
			listFeeds()
		case "check":
			checkFeeds()
		default:
			printUsageAndExit()
		}
	} else if len(args) == 2 {
		switch args[0] {
		case "add":
			addFeed(args[1])
		case "remove":
			removeFeed(args[1])
		default:
			printUsageAndExit()
		}
	} else {
		printUsageAndExit()
	}
}
