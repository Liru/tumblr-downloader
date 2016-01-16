package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync/atomic"
	"time"
)

var (
	gfyRequest = "https://gfycat.com/cajax/get/%s"
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

// Gfy houses the Gfycat response.
type Gfy struct {
	GfyItem struct {
		Mp4Url  string `json:"mp4Url"`
		WebmURL string `json:"webmUrl"`
	} `json:"gfyItem"`
}

// GetGfycatURL gets the appropriate Gfycat URL for download, from a "normal" link.
func GetGfycatURL(slug string) string {
	gfyURL := fmt.Sprintf(gfyRequest, slug)

	var resp *http.Response
	for {
		resp2, err := http.Get(gfyURL)
		if err != nil {
			log.Println(err)
		} else {
			resp = resp2
			break
		}
	}
	defer resp.Body.Close()

	gfyData, _ := ioutil.ReadAll(resp.Body)

	var gfy Gfy

	err := json.Unmarshal(gfyData, &gfy)
	if err != nil {
		log.Println(err)
	}

	return gfy.GfyItem.Mp4Url
}
