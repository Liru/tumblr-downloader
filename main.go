package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/cheggaaa/pb"
)

// VERSION is the current version of the program. It is used for
// checking if blogs need to be force-updated.
//
// This can be changed during the build phase like so:
//     go build -ldflags "-X main.VERSION=1.4.1"
var VERSION = "1.4.0"

var (
	pBar = pb.New(0)
)

func init() {
	flag.BoolVar(&cfg.IgnorePhotos, "ignore-photos", false, "Ignore any photos found in the selected tumblrs.")
	flag.BoolVar(&cfg.IgnoreVideos, "ignore-videos", false, "Ignore any videos found in the selected tumblrs.")
	flag.BoolVar(&cfg.IgnoreAudio, "ignore-audio", false, "Ignore any audio files found in the selected tumblrs.")
	flag.BoolVar(&cfg.UseProgressBar, "p", false, "Use a progress bar to show download status.")

	cfg.version = semver.MustParse(VERSION)
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
	if cfg.UpdateMode {
		log.Println("NOTE: Update mode is now the default mode. The -u flag is not needed and may cause problems in future versions.")
	}

	if cfg.NumDownloaders < 1 {
		log.Println("Invalid number of downloaders, setting to default")
		cfg.NumDownloaders = 10
	}
}

func main() {
	loadConfig()
	flag.Parse()
	verifyFlags()

	walkblock := make(chan struct{})
	go func() {
		fmt.Println("Scanning directory")
		filepath.Walk(cfg.DownloadDirectory, DirectoryScanner)
		fmt.Println("Done scanning.")
		close(walkblock)
	}()

	userBlogs := getUsersToDownload()
	setupDatabase(userBlogs)
	defer database.Close()

	// Here, we're done parsing flags.
	setupSignalInfo()
	<-walkblock
	fileChannels := make([]<-chan File, len(userBlogs)) // FIXME: Seems dirty.

	for {

		limiter := make(chan time.Time, 10*cfg.RequestRate)
		ticker := time.NewTicker(time.Second / time.Duration(cfg.RequestRate))

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

		if cfg.UseProgressBar {
			pBar.Start()
		}

		// Set up downloaders.

		var downloaderWg sync.WaitGroup
		downloaderWg.Add(cfg.NumDownloaders)

		for i := 0; i < cfg.NumDownloaders; i++ {
			go func(j int) {
				downloader(j, limiter, mergedFiles) // mergedFiles will close when scrapers are all done
				downloaderWg.Done()
			}(i)
		}

		downloaderWg.Wait() // Waits for all downloads to complete.

		if cfg.UseProgressBar {
			pBar.Finish()
		}

		updateDatabaseVersion()

		fmt.Println("Downloading complete.")
		gStats.PrintStatus()

		if !cfg.ServerMode {
			break
		}

		fmt.Println("Sleeping for", cfg.ServerSleep)
		time.Sleep(cfg.ServerSleep)
		cfg.UpdateMode = true
		cfg.ForceCheck = false
		ticker.Stop()
	}
}

func showProgress(s ...interface{}) {
	if cfg.UseProgressBar {
		pBar.Update()
	} else if len(s) > 0 {
		fmt.Println(s...)
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
				database.Close()
				gStats.PrintStatus()
				os.Exit(1)
			case syscall.SIGQUIT:
				gStats.PrintStatus()
			}
		}
	}()
}
