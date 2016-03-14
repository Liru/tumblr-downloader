package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/cheggaaa/pb"
)

var userVerificationRegex = regexp.MustCompile(`[A-Za-z0-9]*`)

// UserAction represents what the user is currently doing.
type UserAction int

//go:generate stringer -type=UserAction
const (
	// Scraping is the default action of a user.
	Scraping UserAction = iota

	// Downloading represents a user that's done scraping,
	// but files are still queued up.
	Downloading
)

// User represents a tumblr user blog. It stores details that help
// to download files efficiently.
type User struct {
	name, tag     string
	lastPostID    int64
	highestPostID int64
	progressBar   *pb.ProgressBar
	status        UserAction

	filesFound     int
	filesProcessed int32

	done        chan struct{}
	fileChannel chan File

	idProcessChan   chan int64
	fileProcessChan chan int

	scrapeWg, downloadWg sync.WaitGroup
}

func newUser(name string) (*User, error) {

	if !userVerificationRegex.MatchString(name) {
		return nil, errors.New("newUser: Invalid username format: " + name)
	}

	query := fmt.Sprintf("https://api.tumblr.com/v2/blog/%s.tumblr.com/avatar/16", name)
	resp, err := http.Get(query)
	if err != nil {
		return nil, errors.New("newUser: Couldn't connect to tumblr to check user validity")
	}
	defer resp.Body.Close()

	var js map[string]interface{}
	contents, _ := ioutil.ReadAll(resp.Body)

	// Valid users return images from this call, even default ones.
	// If there is no error while unmarshaling this, then we have valid json.
	// Which means that this is an invalid user.
	if json.Unmarshal(contents, &js) == nil {
		return nil, errors.New("newUser: User not found: " + name)
	}

	// We have a valid user.

	u := &User{
		name:          name,
		lastPostID:    0,
		highestPostID: 0,

		done: make(chan struct{}),

		idProcessChan:   make(chan int64),
		fileProcessChan: make(chan int),
	}

	u.StartHelper()
	gStats.nowScraping.Blog[u] = true
	return u, nil
}

// StartHelper starts a helper goroutine that keeps track of things
// such as a user's highest post ID.
func (u *User) StartHelper() {
	go func() {
		for {
			select {
			case id := <-u.idProcessChan:
				if id > u.highestPostID {
					u.highestPostID = id
				}
			case f := <-u.fileProcessChan:
				u.filesFound += f
				atomic.AddUint64(&gStats.filesFound, uint64(f))
			case <-u.done:
				break
			}
		}
	}()
}

// Queue does stuff.
func (u *User) Queue(p Post) {
	files := parseDataForFiles(p)

	counter := len(files)
	if counter == 0 {
		return
	}
	u.incrementFilesFound(counter)

	timestamp := p.UnixTimestamp

	for _, f := range files {

		pathname := path.Join(cfg.downloadDirectory, u.name, f.Filename)

		// If there is a file that exists, we skip adding it and move on to the next one.
		// Or, if update mode is enabled, then we can simply stop searching.
		_, err := os.Stat(pathname)
		if err == nil {
			atomic.AddUint64(&gStats.alreadyExists, 1)
			atomic.AddInt32(&u.filesProcessed, 1)
			u.downloadWg.Done()
			continue
		}

		f.User = u
		f.UnixTimestamp = timestamp

		atomic.AddInt64(&pBar.Total, 1)

		showProgress()

		u.fileChannel <- f
	} // Done adding URLs from a single post
}

// updateHighestPost sends an integer representing a post ID to the
// user's helper goroutine. It will replace the highest post ID if
// the value sent is higher than the current highest post. Otherwise,
// it does nothing.
//
// TODO: Use updateHighestPost in appropriate area
func (u *User) updateHighestPost(i int64) {
	go func() {
		u.idProcessChan <- i
	}()
}

func (u *User) incrementFilesFound(i int) {
	u.downloadWg.Add(i)
	u.fileProcessChan <- i
}

// finishScraping declares that a user is done scraping, and all that's
// left to do is download the files that were scraped.
//
// finishScraping will wait until all of the scraping goroutines have
// sent their files to the download queue before closing that queue.
func (u *User) finishScraping(i int) {
	fmt.Println("Done scraping for", u.name, "(", i, "pages )")
	u.scrapeWg.Wait()
	u.status = Downloading

	close(u.fileChannel)
	go u.Done()
}

// Done indicates that the user is done everything it's supposed to do.
func (u *User) Done() {
	u.downloadWg.Wait()
	fmt.Println("Done downloading for", u.name)
	close(u.done) // Stop the helper function
	gStats.nowScraping.Blog[u] = false
	updateDatabase(u.name, u.highestPostID)
}

// String implements the Stringer interface.
func (u *User) String() string {
	return u.name
}

// GetStatus prints the status of the user.
//
// Used mostly with GlobalStats to show per-user download/scrape status.
func (u *User) GetStatus() string {
	return fmt.Sprintln(u.name, "-", u.status,
		"(", u.filesProcessed, "/", u.filesFound, ")")
}
