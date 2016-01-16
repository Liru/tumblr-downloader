package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

var (
	totalDownloaded, totalFound uint64
	alreadyExists, totalErrors  uint64 // Used for atomic operation

	numDownloaders int
	requestRate    int
	updateMode     bool

	database *bolt.DB
)

type blog struct {
	name, tag  string
	lastPostID string
}

func init() {
	flag.IntVar(&numDownloaders, "d", 10, "Number of downloaders to run at once")
	flag.IntVar(&requestRate, "r", 4, "Maximum number of requests to make per second")
	flag.BoolVar(&updateMode, "u", false, "Update mode: Stop searching a tumblr when old files are encountered")
}

func readUserFile() ([]*blog, error) {
	path := "download.txt"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var blogs []*blog
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " \n\r\t")
		split := strings.SplitN(text, " ", 2)

		b := &blog{
			name:       split[0],
			lastPostID: "0",
		}

		if len(split) > 1 {
			b.tag = split[1]
		}

		blogs = append(blogs, b)
	}
	// fmt.Println(blogs)
	return blogs, scanner.Err()
}

func main() {

	flag.Parse()

	users := flag.Args()

	fileResults, err := readUserFile()

	if (err != nil) && len(users) == 0 {
		fmt.Fprintln(os.Stderr, "No download.txt detected. Create one and add the blogs you want to download.")
		os.Exit(1)
	}
	userBlogs := make([]*blog, len(users))
	for _, user := range users {
		userBlogs = append(userBlogs, &blog{name: user, lastPostID: "0"})
	}

	userBlogs = append(userBlogs, fileResults...)

	if len(userBlogs) == 0 {
		fmt.Fprintln(os.Stderr, "No users detected.")
		os.Exit(1)
	}

	if numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		numDownloaders = 10
	}

	limiter := make(chan time.Time, 10*requestRate)

	db, err := bolt.Open("tumblr-update.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	database = db

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("tumblr"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		for _, blog := range userBlogs {
			v := b.Get([]byte(blog.name))
			if len(v) != 0 {
				blog.lastPostID = string(v) // TODO: Messy, probably.
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for t := range time.Tick(time.Second / time.Duration(requestRate)) {
			select {
			case limiter <- t:
			default:
			}
		}
	}()

	imageChannels := make([]<-chan Image, len(userBlogs)) // FIXME: Seems dirty.

	// Set up the scraping process.

	for i, user := range userBlogs {
		imgChan := scrape(user, limiter)
		imageChannels[i] = imgChan
	}

	done := make(chan struct{})
	defer close(done)
	images := merge(done, imageChannels)

	// Set up downloaders.

	var downloaderWg sync.WaitGroup
	downloaderWg.Add(numDownloaders)

	for i := 0; i < numDownloaders; i++ {
		go func(j int) {
			downloader(j, limiter, images) // images will close when scrapers are all done
			downloaderWg.Done()
		}(i)
	}

	downloaderWg.Wait()

	fmt.Println("Downloading complete.")
	fmt.Println(totalDownloaded, "/", totalFound, "images downloaded.")
	if alreadyExists != 0 {
		fmt.Println(alreadyExists, "previously downloaded.")
	}
	if totalErrors != 0 {
		fmt.Println(totalErrors, "errors while downloading. You may want to rerun the program to attempt to fix that.")
	}
}
