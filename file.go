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

// A File contains information on a particular tumblr URL, as well as the user where the URL was found.
type File struct {
	User          string
	URL           string
	UnixTimestamp int64
	Filename      string
}

func newFile(URL string) File {
	return File{
		URL:      URL,
		Filename: path.Base(URL),
	}
}

// Download downloads a file specified in the file's URL.
func (f File) Download() {
	var resp *http.Response
	for {
		resp2, err := http.Get(f.URL)
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
		log.Fatal("ReadAll:", err)
	}
	filename := path.Join(cfg.downloadDirectory, f.User, path.Base(f.Filename))

	err = ioutil.WriteFile(filename, pic, 0644)
	if err != nil {
		log.Fatal("WriteFile:", err)
	}

	err = os.Chtimes(filename, time.Now(), time.Unix(f.UnixTimestamp, 0))
	if err != nil {
		log.Println(err)
	}

	pBar.Increment()
	atomic.AddUint64(&gStats.filesDownloaded, 1)
	atomic.AddUint64(&gStats.totalSizeDownloaded, uint64(len(pic)))

}

// String is the standard method for the Stringer interface.
func (f File) String() string {
	date := time.Unix(f.UnixTimestamp, 0)
	return f.User + " - " + date.Format("2006-01-02 15:04:05") + " - " + path.Base(f.Filename)
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
			log.Println("GetGfycatURL: ", err)
		} else {
			resp = resp2
			break
		}
	}
	defer resp.Body.Close()

	gfyData, err := ioutil.ReadAll(resp.Body)
	checkFatalError(err)

	var gfy Gfy

	err = json.Unmarshal(gfyData, &gfy)
	checkFatalError(err, "Gfycat Unmarshal:")

	return gfy.GfyItem.Mp4Url
}

func getGfycatFiles(b, slug string) []File {
	var files []File
	regexResult := gfycatSearch.FindStringSubmatch(b)
	if regexResult != nil {
		for i, v := range regexResult[1:] {
			gfyFile := newFile(GetGfycatURL(v))
			if slug != "" {
				gfyFile.Filename = fmt.Sprintf("%s_gfycat_%02d.mp4", slug, i+1)
			}
			files = append(files, gfyFile)
		}
	}
	return files
}
