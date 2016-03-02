package main

import (
	"log"
	"os"
	"path"
	"time"
)

func downloader(id int, limiter <-chan time.Time, fileChan <-chan File) {
	for f := range fileChan {

		err := os.MkdirAll(path.Join(downloadDirectory, f.User), 0755)
		if err != nil {
			log.Fatal(err)
		}

		<-limiter
		showProgress(f)
		f.Download()

	}
}
