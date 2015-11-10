package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
	//_ "github.com/PuerkitoBio/goquery"
)

var totalDownloaded uint64 // Used for atomic operation

func merge(done <-chan struct{}, cs []chan image) <-chan image {
	var wg sync.WaitGroup
	out := make(chan image)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan image) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done. This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func scrape(user string, limiter <-chan time.Time) <-chan image {
	imagechannel := make(chan image)
	fmt.Println(user)
	return imagechannel
}

func main() {

	var numDownloaders int
	flag.IntVar(&numDownloaders, "d", 1, "Number of downloader workers to run at once")
	flag.Parse()

	users := flag.Args()

	if numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to 1	")
		numDownloaders = 1
	}

	fmt.Println("Users:", users)

	done := make(chan struct{})
	defer close(done)

	limiter := make(chan time.Time, 20) // TODO: Investigate whether this is fine

	go func() {
		for t := range time.Tick(time.Millisecond * 500) {
			select {
			case limiter <- t:
			default:
			}
		} // exits after tick.Stop()
	}()

	imageChannels := make([]chan image, len(users))
	images := merge(done, imageChannels)

	// Set up the scraping process.
	var scrape_wg sync.WaitGroup
	scrape_wg.Add(len(users))

	for _, user := range users {
		go func() {
			scrape(user, limiter) // TODO: Scrape returns a channel of images. Use merge to combine.
			scrape_wg.Done()
		}()
	}

	// Set up downloaders.

	for i := 0; i < numDownloaders; i++ {
		go downloader(i, limiter, images)
	}
}
