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

type UserAction int

const (
	Scraping UserAction = iota
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
	filesProcessed int

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

	return &User{
		name:          name,
		lastPostID:    0,
		highestPostID: 0,
	}, nil
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
			case <-u.done:
				break
			}
		}
	}()
}

// Queue does stuff.
func (u *User) Queue(p Post) {
	var counter int

	files := parseDataForFiles(p)

	if len(files) == 0 {
		return
	}

	timestamp := p.UnixTimestamp

	for _, f := range files {

		pathname := path.Join(cfg.downloadDirectory, u.name, f.Filename)

		// If there is a file that exists, we skip adding it and move on to the next one.
		// Or, if update mode is enabled, then we can simply stop searching.
		_, err := os.Stat(pathname)
		if err == nil {
			atomic.AddUint64(&alreadyExists, 1)
			continue
		}

		f.User = u.name
		f.UnixTimestamp = timestamp

		atomic.AddInt64(&pBar.Total, 1)

		showProgress()

		atomic.AddUint64(&totalFound, 1)
		u.fileChannel <- f
	} // Done adding URLs from a single post

	u.incrementFilesFound(counter)
}

// updateHighestPost sends an integer representing a post ID to the
// user's helper goroutine. It will replace the highest post ID if
// the value sent is higher than the current highest post. Otherwise,
// it does nothing.
func (u *User) updateHighestPost(i int64) {
	go func() {
		u.idProcessChan <- i
	}()
}

func (u *User) incrementFilesFound(i int) {
	go func() {
		u.fileProcessChan <- i
	}()
}

func (u *User) finishScraping(i int) {
	fmt.Println("Done scraping for", u.name, "(", i, "pages )")
	u.scrapeWg.Wait()
	u.status = Downloading
	close(u.fileChannel)
}
