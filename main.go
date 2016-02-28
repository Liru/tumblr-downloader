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
	"github.com/cheggaaa/pb"
)

var (
	totalDownloaded, totalFound uint64
	alreadyExists, totalErrors  uint64 // Used for atomic operation

	totalSizeDownloaded uint64

	numDownloaders    int
	requestRate       int
	updateMode        bool
	serverMode        bool
	serverSleep       time.Duration
	downloadDirectory string

	ignorePhotos   bool
	ignoreVideos   bool
	ignoreAudio    bool
	useProgressBar bool

	database *bolt.DB
	pBar     = pb.New(0)
)

type blog struct {
	name, tag     string
	lastPostID    string
	highestPostID string
	progressBar   *pb.ProgressBar
}

func init() {
	flag.IntVar(&numDownloaders, "d", 10, "Number of downloaders to run at once.")
	flag.IntVar(&requestRate, "r", 4, "Maximum number of requests per second to make.")
	flag.BoolVar(&updateMode, "u", false, "Update mode: Stop searching a tumblr when old files are encountered.")
	flag.BoolVar(&serverMode, "server", false, "Reruns the downloader regularly after a short pause.")
	flag.DurationVar(&serverSleep, "sleep", time.Hour, "Amount of time between download sessions. Used only if server mode is enabled.")

	flag.BoolVar(&ignorePhotos, "ignore-photos", false, "Ignore any photos found in the selected tumblrs.")
	flag.BoolVar(&ignoreVideos, "ignore-videos", false, "Ignore any videos found in the selected tumblrs.")
	flag.BoolVar(&ignoreAudio, "ignore-audio", false, "Ignore any audio files found in the selected tumblrs.")
	flag.BoolVar(&useProgressBar, "p", false, "Use a progress bar to show download status.")
	flag.StringVar(&downloadDirectory, "dir", "", "The `directory` where the files are saved. Default is the directory the program is run from.")
}

func newBlog(name string) *blog {
	return &blog{
		name:          name,
		lastPostID:    "0",
		highestPostID: "0",
		progressBar:   pBar,
	}
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

		b := newBlog(split[0])

		if len(split) > 1 {
			b.tag = split[1]
		}

		blogs = append(blogs, b)
	}
	// fmt.Println(blogs)
	return blogs, scanner.Err()
}

func getBlogsToDownload() []*blog {
	users := flag.Args()

	fileResults, err := readUserFile()
	if err != nil {
		log.Fatal(err)
	}

	userBlogs := make([]*blog, len(users))
	for _, user := range users {
		userBlogs = append(userBlogs, newBlog(user))
	}

	userBlogs = append(userBlogs, fileResults...)

	if len(userBlogs) == 0 {
		fmt.Fprintln(os.Stderr, "No users detected.")
		os.Exit(1)
	}

	return userBlogs
}

func setupDatabase(userBlogs []*blog) {
	db, err := bolt.Open("tumblr-update.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	database = db

	err = db.Update(func(tx *bolt.Tx) error {
		b, boltErr := tx.CreateBucketIfNotExists([]byte("tumblr"))
		if boltErr != nil {
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
		log.Fatal("database: ", err)
	}
}

func main() {
	flag.Parse()

	userBlogs := getBlogsToDownload()
	setupDatabase(userBlogs)
	defer database.Close()

	if numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		numDownloaders = 10
	}

	// Here, we're done parsing flags.

	imageChannels := make([]<-chan Image, len(userBlogs)) // FIXME: Seems dirty.

	for {

		limiter := make(chan time.Time, 10*requestRate)
		ticker := time.NewTicker(time.Second / time.Duration(requestRate))

		go func() {
			for t := range ticker.C {
				select {
				case limiter <- t:
				default:
				}
			}
		}()

		// Set up the scraping process.

		for i, user := range userBlogs {
			imgChan := scrape(user, limiter)
			imageChannels[i] = imgChan
		}

		done := make(chan struct{})
		defer close(done)
		images := merge(done, imageChannels)

		// Set up progress bars.

		if useProgressBar {
			pBar.Start()
		}

		// Set up downloaders.

		var downloaderWg sync.WaitGroup
		downloaderWg.Add(numDownloaders)

		for i := 0; i < numDownloaders; i++ {
			go func(j int) {
				downloader(j, limiter, images) // images will close when scrapers are all done
				downloaderWg.Done()
			}(i)
		}

		downloaderWg.Wait() // Waits for all downloads to complete.

		for _, user := range userBlogs {
			updateDatabase(user.name, user.highestPostID)
		}

		fmt.Println("Downloading complete.")
		printSummary()

		if !serverMode {
			break
		}

		fmt.Println("Sleeping for", serverSleep)
		time.Sleep(serverSleep)
		updateMode = true
		ticker.Stop()
	}
}

func showProgress(s ...interface{}) {
	if useProgressBar {
		pBar.Update()
	} else {
		fmt.Println(s...)
	}
}

func printSummary() {
	fmt.Println(totalDownloaded, "/", totalFound, "images downloaded.")
	fmt.Println(byteSize(totalSizeDownloaded), "downloaded during this session.")
	if alreadyExists != 0 {
		fmt.Println(alreadyExists, "previously downloaded.")
	}
	if totalErrors != 0 {
		fmt.Println(totalErrors, "errors while downloading. You may want to rerun the program to attempt to fix that.")
	}
}

func checkError(err error, args ...interface{}) {
	if err != nil {
		if len(args) != 0 {
			log.Println(args, err)
		} else {
			log.Println(err)
		}
	}
}

func checkFatalError(err error, args ...interface{}) {
	if err != nil {
		if len(args) != 0 {
			log.Fatal(args, err)
		} else {
			log.Fatal(err)
		}
	}
}
