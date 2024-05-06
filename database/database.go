package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"freder.rss-checker/utils"
	"github.com/mmcdole/gofeed"
)

const tableName = "feeds"

type FeedsTableRow struct {
	Id        int
	Url       string
	Title     string
	LastCheck string
}

func OpenDb(dbFilePath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	// create a table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS feeds (
		id         INTEGER PRIMARY KEY,
		url        TEXT NOT NULL UNIQUE,
		title      TEXT NOT NULL,
		last_check TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal("Error creating table: ", err)
	}

	return db
}

func InsertFeed(db *sql.DB, url string) {
	// get title
	content, err := utils.RequestFeed(url)
	if err != nil {
		log.Fatal("Error fetching feed:", err)
	}
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(content)
	if err != nil {
		log.Fatal("Error parsing feed:", err)
	}
	title := feed.Title

	_, err = db.Exec(
		"INSERT INTO "+tableName+" (url, title, last_check) VALUES (?, ?, ?)",
		url, title, "",
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			fmt.Printf("Feed with URL %s already exists in the database\n", url)
			return
		}
		log.Fatal("Error inserting feed into database: ", err)
	}
}

func RemoveFeed(db *sql.DB, url string) {
	_, err := db.Exec("DELETE FROM "+tableName+" WHERE url = ?", url)
	if err != nil {
		log.Fatal("Error deleting feed from database: ", err)
	}
}

func UpdateFeedLastCheck(db *sql.DB, url string, now time.Time) {
	_, err := db.Exec(
		"UPDATE "+tableName+" SET last_check = ? WHERE url = ?",
		now.Format(time.RFC3339), url,
	)
	if err != nil {
		fmt.Println("Error updating last check time: ", err)
	}
}

func GetFeeds(db *sql.DB) []FeedsTableRow {
	fieldNames := []string{
		"id",
		"url",
		"title",
		"last_check",
	}
	fields := strings.Join(fieldNames, ", ")
	query := fmt.Sprintf(
		"SELECT %s FROM %s;",
		fields, tableName,
	)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Error querying database:", err)
	}
	defer rows.Close()

	feedRows := make([]FeedsTableRow, 0)
	for rows.Next() {
		row := FeedsTableRow{}
		err = rows.Scan(
			&row.Id,
			&row.Url,
			&row.Title,
			&row.LastCheck,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error scanning row:", err)
			continue
		}
		feedRows = append(feedRows, row)
	}

	return feedRows
}

func GetFeedUrls(db *sql.DB) map[string]string {
	feedRows := GetFeeds(db)
	feedsMap := make(map[string]string)
	for _, row := range feedRows {
		feedsMap[row.Title] = row.Url
	}
	return feedsMap
}
