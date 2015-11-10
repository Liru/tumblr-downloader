package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	totalDownloaded, totalFound, alreadyExists uint64 // Used for atomic operation
)

func main() {

	var numDownloaders int
	flag.IntVar(&numDownloaders, "d", 3, "Number of downloader workers to run at once")
	flag.Parse()

	users := flag.Args()

	if numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		numDownloaders = 3
	}

	fmt.Println("Users:", users)

	limiter := make(chan time.Time, 20) // TODO: Investigate whether this is fine

	go func() {
		for t := range time.Tick(time.Millisecond * 500) {
			select {
			case limiter <- t:
				// fmt.Println("tick")
			default:
			}
		} // exits after tick.Stop()
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

}
