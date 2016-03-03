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

var PostParseMap = map[string]func(Post) []string{
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
	return c[22 : len(c)-2]
}

func parsePhotoPost(post Post) (URLs []string) {
	if !cfg.ignorePhotos {
		if len(post.Photos) == 0 {
			URLs = append(URLs, post.PhotoURL)
		} else {
			for _, photo := range post.Photos {
				URLs = append(URLs, photo.PhotoURL)
			}
		}
	}

	if !cfg.ignoreVideos {
		regexResult := gfycatSearch.FindStringSubmatch(post.PhotoCaption)
		if regexResult != nil {
			for _, v := range regexResult[1:] {
				URLs = append(URLs, GetGfycatURL(v))
			}
		}
	}
	return
}

func parseAnswerPost(post Post) (URLs []string) {
	if !cfg.ignorePhotos {
		URLs = inlineSearch.FindAllString(post.Answer, -1)
	}
	return
}

func parseRegularPost(post Post) (URLs []string) {
	if !cfg.ignorePhotos {
		URLs = inlineSearch.FindAllString(post.RegularBody, -1)
	}
	return
}

func parseVideoPost(post Post) (URLs []string) {
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

		URLs = append(URLs, videoURL)

		// Here, we get the GfyCat urls from the post.
		regextest = gfycatSearch.FindStringSubmatch(post.VideoCaption)
		if regextest != nil {
			for _, v := range regextest[1:] {
				URLs = append(URLs, GetGfycatURL(v))
			}
		}
	}
	return
}

func parseDataForFiles(post Post) (URLs []string) {
	fn, ok := PostParseMap[post.Type]
	if ok {
		URLs = fn(post)
	}
	return
}

func makeTumblrURL(user *blog, i int) *url.URL {

	base := fmt.Sprintf("http://%s.tumblr.com/api/read/json", user.name)

	tumblrURL, err := url.Parse(base)
	checkFatalError(err, "tumblrURL: ")

	vals := url.Values{}
	vals.Set("num", "50")
	vals.Add("start", strconv.Itoa((i-1)*50))
	// vals.Add("type", "photo")

	if user.tag != "" {
		vals.Add("tagged", user.tag)
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

// strIntLess calculates `strOld < strNew` as a comparison
// without converting to int.
//
// strIntLess assumes that the strings being compared are positive integers.
func strIntLess(strOld, strNew string) bool {
	lenOld, lenNew := len(strOld), len(strNew)

	if lenOld > lenNew {
		return false
	}
	if lenOld < lenNew {
		return true
	}
	return strOld < strNew
}

func scrape(user *blog, limiter <-chan time.Time) <-chan File {
	var wg sync.WaitGroup
	var IDMutex sync.RWMutex

	var once sync.Once
	fileChannel := make(chan File, 1000)

	go func() {

		done := make(chan struct{})
		closeDone := func() { close(done) }
		var i int

		// We need to put all of the following into a function because
		// Go evaluates params at defer instead of at execution.
		// That, and it beats writing `defer` multiple times.
		defer func() {
			fmt.Println("Done scraping for", user.name, "(", i-1, "pages )")
			wg.Wait()
			close(fileChannel)
		}()

		for i = 1; ; i++ {
			if shouldFinishScraping(limiter, done) {
				return
			}

			tumblrURL := makeTumblrURL(user, i)

			// fmt.Println(user.name, "is on page", i)
			showProgress(user.name, "is on page", i)
			resp, err := http.Get(tumblrURL.String())

			// XXX: Ugly as shit. This could probably be done better.
			if err != nil {
				i--
				log.Println(user, err)
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
					if strIntLess(user.highestPostID, post.ID) {
						IDMutex.RUnlock()
						IDMutex.Lock()

						// We need to check again because atomicity isn't guaranteed.
						if strIntLess(user.highestPostID, post.ID) {
							user.highestPostID = post.ID
						}

						IDMutex.Unlock()
					} else {
						IDMutex.RUnlock()
					}

					if !cfg.forceCheck && strIntLess(post.ID, user.lastPostID) {
						once.Do(closeDone)
						break
					}

					URLs := parseDataForFiles(post)

					if len(URLs) == 0 {
						continue
					}

					// fmt.Println(URLs)

					for _, URL := range URLs {
						f := File{
							User:          user.name,
							URL:           URL,
							UnixTimestamp: post.UnixTimestamp,
							ProgressBar:   user.progressBar,
						}

						filename := path.Base(f.URL)
						pathname := path.Join(cfg.downloadDirectory, user.name, filename)

						// If there is a file that exists, we skip adding it and move on to the next one.
						// Or, if update mode is enabled, then we can simply stop searching.
						_, err := os.Stat(pathname)
						if err == nil {
							atomic.AddUint64(&alreadyExists, 1)
							// user.progressBar.Increment()
							continue
						}

						atomic.AddInt64(&user.progressBar.Total, 1)

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
