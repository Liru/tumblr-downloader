package main

import (
	"log"
	"os"
	"path"
	"time"
)

func downloader(id int, limiter <-chan time.Time, imgChan <-chan Image) {
	for img := range imgChan {

		err := os.MkdirAll(path.Join(downloadDirectory, img.User), 0755)
		if err != nil {
			log.Fatal(err)
		}

		<-limiter
		showProgress(img)
		img.Download()

	}
}
