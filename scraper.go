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
	"strings"
	"sync/atomic"
	"time"
)

// A Post is reflective of the JSON used in the tumblr API.
// It contains a PhotoURL, and, optionally, an array of photos.
// If Photos isn't empty, it typically contains at least one URL which matches PhotoURL.
type Post struct {
	PhotoURL string `json:"photo-url-1280"`
	Photos   []Post `json:"photos,omitempty"`
}

// A Blog is the outer container for Posts. It is necessary for easier JSON deserialization,
// even though it's useless in and of itself.
type Blog struct {
	Posts []Post `json:"posts"`
}

func scrape(user blog, limiter <-chan time.Time) <-chan Image {
	imageChannel := make(chan Image)
	fmt.Println(user)
	go func() {
		defer close(imageChannel)

		for i := 1; ; i++ {
			<-limiter

			var tumblrURL string
			if user.tag == "" {
				tumblrURL = fmt.Sprintf("http://%s.tumblr.com/api/read/json?start=%d&num=50&type=photo", user.name, (i-1)*50)
			} else {
				tumblrURL = fmt.Sprintf("http://%s.tumblr.com/api/read/json?start=%d&num=50&type=photo&tagged=%s", user.name, (i-1)*50, url.QueryEscape(user.tag))
			}
			fmt.Println(user.name, "is on page", i)
			resp, err := http.Get(tumblrURL)

			// XXX: Ugly as shit. This could probably be done better.
			if err != nil {
				i--
				log.Println(user, err)
				continue
			}
			defer resp.Body.Close()

			contents, _ := ioutil.ReadAll(resp.Body)

			// This is returned as pure JSON. We need to filter out the variable and the ending semicolon.
			contents = []byte(strings.Replace(string(contents), "var tumblr_api_read = ", "", 1))
			contents = []byte(strings.Replace(string(contents), ";", "", -1))

			var blog Blog
			json.Unmarshal(contents, &blog)

			if len(blog.Posts) == 0 {
				break
			}

			for _, post := range blog.Posts {
				if len(post.Photos) == 0 {

					i := Image{User: user.name, URL: post.PhotoURL}

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

				} else {
					for _, photo := range post.Photos { // FIXME: This is messy.

						i := Image{User: user.name, URL: photo.PhotoURL}

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

		}

		fmt.Println("Done scraping for", user.name)

	}()
	return imageChannel
}
