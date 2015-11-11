package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Post struct {
	PhotoUrl string `json:"photo-url-1280"`
	Photos   []Post `json:"photos"`
}

type Blog struct {
	Posts []Post `json:"posts"`
}

func scrape(user string, limiter <-chan time.Time) <-chan Image {
	imageChannel := make(chan Image)
	fmt.Println(user)
	go func() {
		defer close(imageChannel)

		for i := 1; ; i++ {
			<-limiter

			tumblrUrl := fmt.Sprintf("http://%s.tumblr.com/api/read/json?start=%d&num=50&type=photo", user, (i-1)*50)
			fmt.Println(user, "is on page", i)

			resp, _ := http.Get(tumblrUrl)

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
					imageChannel <- Image{User: user, Url: post.PhotoUrl}
					atomic.AddUint64(&totalFound, 1)
				} else {
					for _, photo := range post.Photos { // FIXME: This is messy.
						imageChannel <- Image{User: user, Url: photo.PhotoUrl}
						atomic.AddUint64(&totalFound, 1)
					}
				}
			}

		}

		fmt.Println("Done scraping for", user)

	}()
	return imageChannel
}
