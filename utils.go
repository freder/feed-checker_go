package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func updateLastCheckTimeFile(now time.Time) {
	timeFile, err := os.Create(lastCheckTimeFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating file:", err)
		os.Exit(1)
	}
	defer timeFile.Close()
	_, err = timeFile.WriteString(now.Format(time.RFC3339))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to file:", err)
		os.Exit(1)
	}
}

func getLastCheckTime() time.Time {
	var lastCheckTime time.Time

	if _, err := os.Stat(lastCheckTimeFilePath); os.IsNotExist(err) {
		lastCheckTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	} else {
		bytes, err := os.ReadFile(lastCheckTimeFilePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading file:", err)
			os.Exit(1)
		}
		parsed, err := time.Parse(time.RFC3339, string(bytes))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			os.Exit(1)
		}
		lastCheckTime = parsed
	}

	return lastCheckTime
}

func readFeedUrls() map[string]string {
	file, err := os.Open(feedsFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var data map[string]string
	err = decoder.Decode(&data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error decoding JSON:", err)
		os.Exit(1)
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
