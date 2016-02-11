package main

import (
	"os"
	"time"
)

func downloader(id int, limiter <-chan time.Time, imgChan <-chan Image) {
	for img := range imgChan {

		os.MkdirAll(img.User, 0755)

		<-limiter
		Update(img)
		img.Download()

	}
}
