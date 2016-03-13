package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/boltdb/bolt"
	"github.com/cheggaaa/pb"
)

// VERSION is the current version of the program. It is used for
// checking if blogs need to be force-updated.
//
// This can be changed during the build phase like so:
//     go build -ldflags "-X main.VERSION=1.4.1"
var VERSION = "1.4.0"

var (
	cfg Config

	database *bolt.DB
	pBar     = pb.New(0)
)

func init() {
	flag.IntVar(&cfg.numDownloaders, "d", 10, "Number of downloaders to run at once.")
	flag.IntVar(&cfg.requestRate, "r", 4, "Maximum number of requests per second to make.")
	flag.BoolVar(&cfg.updateMode, "u", false, "Update mode. DEPRECATED: Update mode is now the default mode.")
	flag.BoolVar(&cfg.forceCheck, "f", false, "Bypasses update mode and rechecks all blog pages")
	flag.BoolVar(&cfg.serverMode, "server", false, "Reruns the downloader regularly after a short pause.")
	flag.DurationVar(&cfg.serverSleep, "sleep", time.Hour, "Amount of time between download sessions. Used only if server mode is enabled.")

	flag.BoolVar(&cfg.ignorePhotos, "ignore-photos", false, "Ignore any photos found in the selected tumblrs.")
	flag.BoolVar(&cfg.ignoreVideos, "ignore-videos", false, "Ignore any videos found in the selected tumblrs.")
	flag.BoolVar(&cfg.ignoreAudio, "ignore-audio", false, "Ignore any audio files found in the selected tumblrs.")
	flag.BoolVar(&cfg.useProgressBar, "p", false, "Use a progress bar to show download status.")
	flag.StringVar(&cfg.downloadDirectory, "dir", "", "The `directory` where the files are saved. Default is the directory the program is run from.")

	cfg.Version = semver.MustParse(VERSION)
}

func readUserFile() ([]*User, error) {
	path := "download.txt"
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var users []*User
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.Trim(scanner.Text(), " \n\r\t")
		split := strings.SplitN(text, " ", 2)

		b, err := newUser(split[0])
		if err != nil {
			log.Println(err)
			continue
		}

		if len(split) > 1 {
			b.tag = split[1]
		}

		users = append(users, b)
	}
	// fmt.Println(blogs)
	return users, scanner.Err()
}

func getUsersToDownload() []*User {
	users := flag.Args()

	fileResults, err := readUserFile()
	if err != nil {
		log.Fatal(err)
	}

	var userBlogs []*User
	for _, user := range users {
		u, err := newUser(user)
		if err != nil {
			log.Println(err)
			continue
		}
		userBlogs = append(userBlogs, u)
	}

	userBlogs = append(userBlogs, fileResults...)

	if len(userBlogs) == 0 {
		fmt.Fprintln(os.Stderr, "No users detected.")
		os.Exit(1)
	}

	return userBlogs
}

func verifyFlags() {
	if cfg.updateMode {
		log.Println("NOTE: Update mode is now the default mode. The -u flag is not needed and may cause problems in future versions.")
	}

	if cfg.numDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		cfg.numDownloaders = 10
	}
}

func main() {
	flag.Parse()
	verifyFlags()

	userBlogs := getUsersToDownload()
	setupDatabase(userBlogs)
	defer database.Close()

	// Here, we're done parsing flags.
	setupSignalInfo()

	fileChannels := make([]<-chan File, len(userBlogs)) // FIXME: Seems dirty.

	for {

		limiter := make(chan time.Time, 10*cfg.requestRate)
		ticker := time.NewTicker(time.Second / time.Duration(cfg.requestRate))

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
			fileChan := scrape(user, limiter)
			fileChannels[i] = fileChan
		}

		done := make(chan struct{})
		defer close(done)
		mergedFiles := merge(done, fileChannels)

		// Set up progress bars.

		if cfg.useProgressBar {
			pBar.Start()
		}

		// Set up downloaders.

		var downloaderWg sync.WaitGroup
		downloaderWg.Add(cfg.numDownloaders)

		for i := 0; i < cfg.numDownloaders; i++ {
			go func(j int) {
				downloader(j, limiter, mergedFiles) // mergedFiles will close when scrapers are all done
				downloaderWg.Done()
			}(i)
		}

		downloaderWg.Wait() // Waits for all downloads to complete.
		pBar.Finish()

		updateDatabaseVersion()
		for _, user := range userBlogs {
			updateDatabase(user.name, user.highestPostID)
		}

		fmt.Println("Downloading complete.")
		printSummary()

		if !cfg.serverMode {
			break
		}

		fmt.Println("Sleeping for", cfg.serverSleep)
		time.Sleep(cfg.serverSleep)
		cfg.updateMode = true
		cfg.forceCheck = false
		ticker.Stop()
	}
}

func showProgress(s ...interface{}) {
	if cfg.useProgressBar {
		pBar.Update()
	} else if len(s) > 0 {
		fmt.Println(s...)
	}
}

func printSummary() {
	fmt.Println(gStats.filesDownloaded, "/", gStats.filesFound, "files downloaded.")
	fmt.Println(byteSize(gStats.totalSizeDownloaded), "downloaded during this session.")
	if gStats.alreadyExists != 0 {
		fmt.Println(gStats.alreadyExists, "previously downloaded.")
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

func setupSignalInfo() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGINT:
				printSummary()
				os.Exit(1)
			case syscall.SIGQUIT:
				printSummary()
			}
		}
	}()
}
