package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync/atomic"
	"time"
)

// An Image contains information on a particular image URL, as well as the user where the URL was found.
type Image struct {
	User          string
	URL           string
	UnixTimestamp int64
}

// Download downloads an image specified in an Image's URL.
func (i Image) Download() {
	var resp *http.Response
	for {
		resp2, err := http.Get(i.URL)
		if err != nil {
			log.Println(err)
		} else {
			resp = resp2
			break
		}
	}
	defer resp.Body.Close()

	pic, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	file := "downloads/" + i.User + "/" + path.Base(i.URL)

	err = ioutil.WriteFile(file, pic, 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chtimes(file, time.Now(), time.Unix(i.UnixTimestamp, 0))
	if err != nil {
		log.Println(err)
	}

	atomic.AddUint64(&totalDownloaded, 1)

}

// Standard String method for the Stringer interface.
func (i Image) String() string {
	date := time.Unix(i.UnixTimestamp, 0)
	return i.User + " - " + date.Format("2006-01-02 15:04:05") + " - " + path.Base(i.URL)
}
