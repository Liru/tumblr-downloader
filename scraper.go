package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
	"sync/atomic"
	"time"
)

func scrape(user string, limiter <-chan time.Time) <-chan Image {
	imageChannel := make(chan Image)
	fmt.Println(user)
	go func() {
		defer close(imageChannel)

		for i := 1; i < 4; i++ {
			<-limiter

			tumblrUrl := fmt.Sprintf("http://%s.tumblr.com/page/%d", user, i)
			fmt.Println(user, "is on page", i)

			doc, _ := goquery.NewDocument(tumblrUrl)
			// TODO: Error handling
			s := strings.Trim(doc.Find("#blog").Text(), " \f\n\r\t")

			if len(s) == 0 {
				fmt.Println("blog is empty at", i, "; aborting")
				break
			}

			// TODO: have scraper go to each post individually, as stuff may be lost otherwise

			doc.Find(".image-link.clearfix").Each(func(i int, s *goquery.Selection) {
				class, _ := s.Attr("href")
				// fmt.Println(class)
				atomic.AddUint64(&totalFound, 1)
				imageChannel <- Image{User: user, Url: class}
			})
		}

		fmt.Println("Done scraping for", user)

	}()
	return imageChannel
}
