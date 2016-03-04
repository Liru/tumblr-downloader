package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	inlineSearch   = regexp.MustCompile(`(http:\/\/\d{2}\.media\.tumblr\.com\/\w{32}\/tumblr_inline_\w+\.\w+)`) // FIXME: Possibly buggy/unoptimized.
	videoSearch    = regexp.MustCompile(`"hdUrl":"(.*\/tumblr_\w+)"`)                                           // fuck it
	altVideoSearch = regexp.MustCompile(`source src="(.*tumblr_\w+)(?:\/\d+)?" type`)
	gfycatSearch   = regexp.MustCompile(`href="https?:\/\/(?:www\.)?gfycat\.com\/(\w+)`)
)

// PostParseMap maps tumblr post types to functions that search those
// posts for content.
var PostParseMap = map[string]func(Post) []File{
	"photo":   parsePhotoPost,
	"answer":  parseAnswerPost,
	"regular": parseRegularPost,
	"video":   parseVideoPost,
}

// TrimJS trims the javascript response received from Tumblr.
// The response starts with "var tumblr_api_read = " and ends with ";".
// We need to remove these to parse the response as JSON.
func TrimJS(c []byte) []byte {
	// The length of "var tumblr_api_read = " is 22.
	return c[22 : len(c)-1]
}

func parsePhotoPost(post Post) (files []File) {
	var id string
	if !cfg.ignorePhotos {

		if len(post.Photos) == 0 {
			f := newFile(post.PhotoURL)
			files = append(files, f)
			id = f.Filename
		} else {
			for _, photo := range post.Photos {
				f := newFile(photo.PhotoURL)
				files = append(files, f)
				id = f.Filename
			}

		}
	}

	if !cfg.ignoreVideos {
		var slug string
		if len(id) > 26 {
			slug = id[:26]
		}
		files = append(files, getGfycatFiles(post.PhotoCaption, slug)...)
	}
	return
}

func parseAnswerPost(post Post) (files []File) {
	if !cfg.ignorePhotos {
		for _, f := range inlineSearch.FindAllString(post.Answer, -1) {
			files = append(files, newFile(f))
		}
	}
	return
}

func parseRegularPost(post Post) (files []File) {
	if !cfg.ignorePhotos {
		for _, f := range inlineSearch.FindAllString(post.RegularBody, -1) {
			files = append(files, newFile(f))
		}
	}
	return
}

func parseVideoPost(post Post) (files []File) {
	if !cfg.ignoreVideos {
		regextest := videoSearch.FindStringSubmatch(post.Video)
		if regextest == nil { // hdUrl is false. We have to get the other URL.
			regextest = altVideoSearch.FindStringSubmatch(post.Video)
		}

		// If it's still nil, it means it's another embedded video type, like Youtube, Vine or Pornhub.
		// In that case, ignore it and move on. Not my problem.
		if regextest == nil {
			return
		}
		videoURL := strings.Replace(regextest[1], `\`, ``, -1)

		// If there are problems with downloading video, the below part may be the cause.
		// videoURL = strings.Replace(videoURL, `/480`, ``, -1)
		videoURL += ".mp4"

		f := newFile(videoURL)
		files = append(files, f)

		// We slice from 0 to 24 because that's the length of the ID
		// portion of a tumblr video file.
		slug := f.Filename[:23]

		files = append(files, getGfycatFiles(post.VideoCaption, slug)...)
	}
	return
}

func parseDataForFiles(post Post) (files []File) {
	fn, ok := PostParseMap[post.Type]
	if ok {
		files = fn(post)
	}
	return
}

func makeTumblrURL(u *User, i int) *url.URL {

	base := fmt.Sprintf("http://%s.tumblr.com/api/read/json", u.name)

	tumblrURL, err := url.Parse(base)
	checkFatalError(err, "tumblrURL: ")

	vals := url.Values{}
	vals.Set("num", "50")
	vals.Add("start", strconv.Itoa((i-1)*50))
	// vals.Add("type", "photo")

	if u.tag != "" {
		vals.Add("tagged", u.tag)
	}

	tumblrURL.RawQuery = vals.Encode()
	return tumblrURL
}

func shouldFinishScraping(lim <-chan time.Time, done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		select {
		case <-done:
			return true
		case <-lim:
			// We get a value from limiter, and proceed to scrape a page.
			return false
		}
	}
}

func scrape(u *User, limiter <-chan time.Time) <-chan File {
	var wg sync.WaitGroup
	var IDMutex sync.RWMutex

	var once sync.Once
	fileChannel := make(chan File, 10000)

	go func() {

		done := make(chan struct{})
		closeDone := func() { close(done) }
		var i int

		// We need to put all of the following into a function because
		// Go evaluates params at defer instead of at execution.
		// That, and it beats writing `defer` multiple times.
		defer func() {
			fmt.Println("Done scraping for", u.name, "(", i-1, "pages )")
			wg.Wait()
			close(fileChannel)
		}()

		for i = 1; ; i++ {
			if shouldFinishScraping(limiter, done) {
				return
			}

			tumblrURL := makeTumblrURL(u, i)

			showProgress(u.name, "is on page", i)
			resp, err := http.Get(tumblrURL.String())

			// XXX: Ugly as shit. This could probably be done better.
			if err != nil {
				i--
				log.Println(u, err)
				continue
			}
			defer resp.Body.Close()

			contents, _ := ioutil.ReadAll(resp.Body)

			// This is returned as pure javascript. We need to filter out the variable and the ending semicolon.
			contents = TrimJS(contents)

			var blog TumbleLog
			err = json.Unmarshal(contents, &blog)
			if err != nil {
				// Goddamnit tumblr, make a consistent API that doesn't
				// fucking return strings AND booleans in the same field
			}

			if len(blog.Posts) == 0 {
				break
			}

			wg.Add(1)

			go func() {
				defer wg.Done()

				for _, post := range blog.Posts {

					IDMutex.RLock()
					if post.ID > u.highestPostID {
						IDMutex.RUnlock()
						IDMutex.Lock()

						// We need to check again because atomicity isn't guaranteed.
						if post.ID > u.highestPostID {
							u.highestPostID = post.ID
						}

						IDMutex.Unlock()
					} else {
						IDMutex.RUnlock()
					}

					if !cfg.forceCheck && post.ID < u.lastPostID {
						once.Do(closeDone)
						break
					}

					files := parseDataForFiles(post)

					if len(files) == 0 {
						continue
					}

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
						f.UnixTimestamp = post.UnixTimestamp

						atomic.AddInt64(&pBar.Total, 1)

						showProgress()

						atomic.AddUint64(&totalFound, 1)
						fileChannel <- f
					} // Done adding URLs from a single post

				} // Done searching all posts on a page

			}() // Function that asynchronously adds all URLs to download queue

		} // loop that searches blog, page by page

	}() // Function that asynchronously adds all downloadables from a blog to a queue
	return fileChannel
}
