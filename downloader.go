package main

import (
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"
)

func downloader(id int, limiter <-chan time.Time, imgChan <-chan Image) {
	for img := range imgChan {

		os.MkdirAll("downloads/"+img.User, 0755)
		filename := path.Base(img.Url)
		pathname := fmt.Sprintf("downloads/%s/%s", img.User, filename)

		if _, err := os.Stat(pathname); os.IsNotExist(err) {
			<-limiter
			fmt.Println(img)
			img.Download()
		} else { // file already exists. Or another error happened. Screw the latter scenario.
			atomic.AddUint64(&alreadyExists, 1)
		}

	}
}
