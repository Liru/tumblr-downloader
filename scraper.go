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
	"sync/atomic"
	"time"
)

// A Post is reflective of the JSON used in the tumblr API.
// It contains a PhotoURL, and, optionally, an array of photos.
// If Photos isn't empty, it typically contains at least one URL which matches PhotoURL.
type Post struct {
	Type          string
	PhotoURL      string `json:"photo-url-1280"`
	Photos        []Post `json:"photos,omitempty"`
	UnixTimestamp int64  `json:"unix-timestamp"`

	// for regular posts
	RegularBody string `json:"regular-body"`

	// for answer posts
	Answer string
}

// A Blog is the outer container for Posts. It is necessary for easier JSON deserialization,
// even though it's useless in and of itself.
type Blog struct {
	Posts []Post `json:"posts"`
}

var inlineSearch = regexp.MustCompile(`(http:\/\/\d{2}\.media\.tumblr\.com\/\w{32}\/tumblr_inline_\w+\.\w+)`) // FIXME: Possibly buggy/unoptimized.

func scrape(user blog, limiter <-chan time.Time) <-chan Image {
	imageChannel := make(chan Image)
	fmt.Println(user)
	go func() {
		defer close(imageChannel)

		for i := 1; ; i++ {
			<-limiter

			base := fmt.Sprintf("http://%s.tumblr.com/api/read/json", user.name)

			tumblrURL, err := url.Parse(base)
			if err != nil {
				log.Fatal(err)
			}

			vals := url.Values{}
			vals.Set("num", "50")
			vals.Add("start", strconv.Itoa((i-1)*50))
			// vals.Add("type", "photo")

			if user.tag != "" {
				vals.Add("tagged", user.tag)
			}

			tumblrURL.RawQuery = vals.Encode()

			// fmt.Println(tumblrURL)

			fmt.Println(user.name, "is on page", i)
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

			for _, post := range blog.Posts {

				var URLs []string

				switch post.Type {
				case "photo":
					if len(post.Photos) == 0 {
						URLs = append(URLs, post.PhotoURL)
					} else {
						for _, photo := range post.Photos {
							URLs = append(URLs, photo.PhotoURL)
						}
					}

				case "answer":
					URLs = inlineSearch.FindAllString(post.Answer, -1)
				case "regular":
					URLs = inlineSearch.FindAllString(post.RegularBody, -1)
				default:
					continue
				}

				// fmt.Println(URLs)

				for _, URL := range URLs {
					i := Image{
						User:          user.name,
						URL:           URL,
						UnixTimestamp: post.UnixTimestamp,
					}

					filename := path.Base(i.URL)
					pathname := fmt.Sprintf("downloads/%s/%s", user.name, filename)

					// If there is a file that exists, we skip adding it and move on to the next one.
					// Or, if update mode is enabled, then we can simply stop searching.
					_, err := os.Stat(pathname)
					if err == nil {
						if updateMode {
							return
						}
						atomic.AddUint64(&alreadyExists, 1)
						continue

					}

					atomic.AddUint64(&totalFound, 1)
					imageChannel <- i
				}

			}

		}

		fmt.Println("Done scraping for", user.name)

	}()
	return imageChannel
}
