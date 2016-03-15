package main

import (
	"fmt"
	"sync"
)

var gStats = NewGlobalStats()

// GlobalStats keeps track of various statistics during a download process.
type GlobalStats struct {
	filesDownloaded uint64
	filesFound      uint64
	alreadyExists   uint64

	// bytesDownloaded only counts bytes from files.
	bytesDownloaded uint64
	// bytesOverhead counts bytes from the json scraping.
	bytesOverhead uint64

	// NowScraping is used to show which blogs are being scraped.
	nowScraping Tracker
}

// Tracker is a record-keeping structure that tracks which users are still
// in the scraping/downloading phase.
type Tracker struct {
	sync.RWMutex
	Blog map[*User]bool
}

// NewGlobalStats does the initialization for a new set of global stats.
func NewGlobalStats() *GlobalStats {
	return &GlobalStats{
		nowScraping: Tracker{
			Blog: make(map[*User]bool),
		},
	}
}

// PrintStatus prints the current status of each active user.
//
// It currently prints active (scraping and downloading) blogs.
// Not sure if it should be changed to also include finished blogs.
func (g *GlobalStats) PrintStatus() {
	g.nowScraping.RLock()
	defer g.nowScraping.RUnlock()

	// XXX: Optimize this if necessary.
	for k, v := range g.nowScraping.Blog {
		if v {
			fmt.Println(k.GetStatus())
		}
	}

	fmt.Println(g.filesDownloaded, "/", g.filesFound-g.alreadyExists, "files downloaded.")
	if g.alreadyExists != 0 {
		fmt.Println(g.alreadyExists, "previously downloaded.")
	}
	fmt.Println(byteSize(g.bytesDownloaded), "of files downloaded during this session.")
	fmt.Println(byteSize(g.bytesOverhead), "of data downloaded as JSON overhead.")
}
