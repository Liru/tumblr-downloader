package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sync/atomic"
)

type Image struct {
	User string
	Url  string
}

func (i Image) Download() {
	resp, err := http.Get(i.Url)
	defer resp.Body.Close()

	if err != nil {
	}

	pic, err := ioutil.ReadAll(resp.Body)
	if err != nil {
	}

	file := "downloads/" + i.User + "/" + path.Base(i.Url)

	err = ioutil.WriteFile(file, pic, 0644)
	if err != nil {
		log.Fatal(err)
	}

	atomic.AddUint64(&totalDownloaded, 1)

}

func (i Image) String() string {
	return i.User + "-" + path.Base(i.Url)
}
