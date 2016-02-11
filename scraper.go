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

// A Post is reflective of the JSON used in the tumblr API.
// It contains a PhotoURL, and, optionally, an array of photos.
// If Photos isn't empty, it typically contains at least one URL which matches PhotoURL.
type Post struct {
	ID            string
	Type          string
	PhotoURL      string `json:"photo-url-1280"`
	Photos        []Post `json:"photos,omitempty"`
	UnixTimestamp int64  `json:"unix-timestamp"`
	PhotoCaption  string `json:"photo-caption"`

	// for regular posts
	RegularBody string `json:"regular-body"`

	// for answer posts
	Answer string

	// for videos
	Video        string `json:"video-player"`
	VideoCaption string `json:"video-caption"` // For links to outside sites.
}

// A Blog is the outer container for Posts. It is necessary for easier JSON deserialization,
// even though it's useless in and of itself.
type Blog struct {
	Posts []Post `json:"posts"`
}

var (
	inlineSearch   = regexp.MustCompile(`(http:\/\/\d{2}\.media\.tumblr\.com\/\w{32}\/tumblr_inline_\w+\.\w+)`) // FIXME: Possibly buggy/unoptimized.
	videoSearch    = regexp.MustCompile(`"hdUrl":"(.*\/tumblr_\w+)"`)                                           // fuck it
	altVideoSearch = regexp.MustCompile(`source src="(.*tumblr_\w+)(?:\/\d+)?" type`)
	gfycatSearch   = regexp.MustCompile(`href="https?:\/\/(?:www\.)?gfycat\.com\/(\w+)`)
)

func scrape(user *blog, limiter <-chan time.Time) <-chan Image {
	var wg sync.WaitGroup
	highestID := "0"
	var IDMutex sync.Mutex

	var once sync.Once
	imageChannel := make(chan Image, 1000)

	go func() {

		done := make(chan struct{})
		closeDone := func() { close(done) }
		var i int

		defer updateDatabase(user.name, &highestID)
		defer close(imageChannel)
		defer wg.Wait()
		defer fmt.Println("Done scraping for", user.name, "(", i, "pages )")
		for i = 1; ; i++ {
			select {
			case <-done:
				return
			default:
				select {
				case <-done:
					return
				case <-limiter:
					// We get a value from limiter, and proceed to scrape a page.
				}
			}

			base := fmt.Sprintf("http://%s.tumblr.com/api/read/json", user.name)

			tumblrURL, err := url.Parse(base)
			if err != nil {
				log.Fatal("tumblrURL: ", err)
			}

			vals := url.Values{}
			vals.Set("num", "50")
			vals.Add("start", strconv.Itoa((i-1)*50))
			// vals.Add("type", "photo")

			if user.tag != "" {
				vals.Add("tagged", user.tag)
			}

			tumblrURL.RawQuery = vals.Encode()

			// fmt.Println(user.name, "is on page", i)
			Update(user.name, "is on page", i)
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
			contents = []byte(strings.Replace(string(contents), "var tumblr_api_read = ", "", 1))
			contents = []byte(strings.Replace(string(contents), ";", "", -1))

			var blog Blog
			json.Unmarshal(contents, &blog)

			if len(blog.Posts) == 0 {
				break
			}

			wg.Add(1)

			go func() {
				defer wg.Done()
				lastPostIDint, err := strconv.Atoi(user.lastPostID)
				if err != nil {
					log.Fatal("parse1", err)
				}
				for _, post := range blog.Posts {
					postIDint, _ := strconv.Atoi(post.ID)

					IDMutex.Lock()
					highestIDint, _ := strconv.Atoi(highestID)
					if postIDint >= highestIDint {
						highestID = post.ID
					}
					IDMutex.Unlock()

					if (postIDint <= lastPostIDint) && updateMode {
						once.Do(closeDone)
						break
					}

					var URLs []string

					switch post.Type { // TODO: Refactor and clean this up. This is messy and has repeated code.
					case "photo":
						if !ignorePhotos {
							if len(post.Photos) == 0 {
								URLs = append(URLs, post.PhotoURL)
							} else {
								for _, photo := range post.Photos {
									URLs = append(URLs, photo.PhotoURL)
								}
							}
						}

						if !ignoreVideos {
							regexResult := gfycatSearch.FindStringSubmatch(post.PhotoCaption)
							if regexResult != nil {
								for _, v := range regexResult[1:] {
									URLs = append(URLs, GetGfycatURL(v))
								}
							}
						}

					case "answer":
						if !ignorePhotos {
							URLs = inlineSearch.FindAllString(post.Answer, -1)
						}
					case "regular":
						if !ignorePhotos {
							URLs = inlineSearch.FindAllString(post.RegularBody, -1)
						}
					case "video":
						if !ignoreVideos {
							regextest := videoSearch.FindStringSubmatch(post.Video)
							if regextest == nil { // hdUrl is false. We have to get the other URL.
								regextest = altVideoSearch.FindStringSubmatch(post.Video)
							}

							// If it's still nil, it means it's another embedded video type, like Youtube, Vine or Pornhub.
							// In that case, ignore it and move on. Not my problem.
							if regextest == nil {
								continue
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

					default:
						continue
					} // Done switch statement

					// fmt.Println(URLs)

					for _, URL := range URLs {
						i := Image{
							User:          user.name,
							URL:           URL,
							UnixTimestamp: post.UnixTimestamp,
							ProgressBar:   user.progressBar,
						}

						atomic.AddInt64(&user.progressBar.Total, 1)

						filename := path.Base(i.URL)
						pathname := fmt.Sprintf("%s/%s", user.name, filename)

						// If there is a file that exists, we skip adding it and move on to the next one.
						// Or, if update mode is enabled, then we can simply stop searching.
						_, err := os.Stat(pathname)
						if err == nil {
							atomic.AddUint64(&alreadyExists, 1)
							user.progressBar.Increment()
							continue
						}

						atomic.AddUint64(&totalFound, 1)
						imageChannel <- i
					} // Done adding URLs from a single post

				} // Done searching all posts on a page

			}() // Function that asynchronously adds all URLs to download queue

		} // loop that searches blog, page by page

	}() // Function that asynchronously adds all downloadables from a blog to a queue
	return imageChannel
}
