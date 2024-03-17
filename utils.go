package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func updateLastCheckTimeFile(now *time.Time) {
	timeFile, err := os.Create(lastCheckTimeFilePath)
	if err != nil {
		log.Fatal("Error creating file:", err)
	}
	defer timeFile.Close()
	_, err = timeFile.WriteString(now.Format(time.RFC3339))
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}
}

func getLastCheckTime() time.Time {
	var lastCheckTime time.Time

	if _, err := os.Stat(lastCheckTimeFilePath); os.IsNotExist(err) {
		lastCheckTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	} else {
		bytes, err := os.ReadFile(lastCheckTimeFilePath)
		if err != nil {
			log.Fatal("Error reading file:", err)
		}
		parsed, err := time.Parse(time.RFC3339, string(bytes))
		if err != nil {
			log.Fatal("Error parsing time:", err)
		}
		lastCheckTime = parsed
	}

	return lastCheckTime
}

func readFeedUrls() map[string]string {
	file, err := os.Open(feedsFilePath)
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var data map[string]string
	err = decoder.Decode(&data)
	if err != nil {
		log.Fatal("Error decoding JSON:", err)
	}
	return data
}

func requestFeed(url string) string {
	res, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting feed:", err)
		return ""
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading body:", err)
		return ""
	}

	return string(body)
}
