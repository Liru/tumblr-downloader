package main

import (
	"fmt"
	"os"
	"time"
)

func downloader(id int, limiter <-chan time.Time, imgChan <-chan Image) {
	for img := range imgChan {

		os.MkdirAll("downloads/"+img.User, 0755)

		<-limiter
		fmt.Println(img)
		img.Download()

	}
}
