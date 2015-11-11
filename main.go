package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	totalDownloaded, totalFound uint64
	alreadyExists, totalErrors  uint64 // Used for atomic operation

	numDownloaders, requestRate int // used for flags
)

func init() {
	flag.IntVar(&numDownloaders, "d", 3, "Number of downloader workers to run at once")
	flag.IntVar(&requestRate, "r", 2, "Maximum number of requests to make per second")
}

func main() {

	flag.Parse()

	users := flag.Args()

	if numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		numDownloaders = 3
	}

	limiter := make(chan time.Time, 20) // TODO: Investigate whether this is fine

	go func() {
		for t := range time.Tick(time.Second / time.Duration(requestRate)) {
			select {
			case limiter <- t:
			default:
			}
		}
	}()

	imageChannels := make([]<-chan Image, len(users)) // FIXME: Seems dirty.

	// Set up the scraping process.

	for i, user := range users {

		imgChan := scrape(user, limiter) // TODO: Scrape returns a channel of images. Use merge to combine.
		imageChannels[i] = imgChan

	}

	done := make(chan struct{})
	defer close(done)
	images := merge(done, imageChannels)

	// Set up downloaders.

	var downloader_wg sync.WaitGroup
	downloader_wg.Add(numDownloaders)

	for i := 0; i < numDownloaders; i++ {
		go func(j int) {
			downloader(j, limiter, images) // images will close when scrapers are all done
			downloader_wg.Done()
		}(i)
	}

	fmt.Println("Waiting...")

	downloader_wg.Wait()

	fmt.Println("Downloading complete.")
	fmt.Println(totalDownloaded, "/", totalFound, "images downloaded.")
	if alreadyExists != 0 {
		fmt.Println(alreadyExists, "previously downloaded.")
	}
	if totalErrors != 0 {
		fmt.Println(totalErrors, "errors while downloading. You may want to rerun the program to attempt to fix that.")
	}
}
