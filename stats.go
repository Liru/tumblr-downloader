package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var gStats = NewGlobalStats()

// GlobalStats keeps track of various statistics during a download process.
type GlobalStats struct {
	filesDownloaded uint64
	filesFound      uint64
	alreadyExists   uint64
	hardlinked      uint64

	// bytesDownloaded only counts bytes from files.
	bytesDownloaded uint64
	// bytesOverhead counts bytes from the json scraping.
	bytesOverhead uint64
	// bytesSaved indicated space saved due to hardlinking instead of downloading.
	bytesSaved uint64

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

	fmt.Println()
	// XXX: Optimize this if necessary.
	for k, v := range g.nowScraping.Blog {
		if v {
			fmt.Println(k.GetStatus())
		}
	}
	fmt.Println()

	filesFound := atomic.LoadUint64(&g.filesFound)
	alreadyExists := atomic.LoadUint64(&g.alreadyExists)
	filesDownloaded := atomic.LoadUint64(&g.filesDownloaded)
	hardlinked := atomic.LoadUint64(&g.hardlinked)
	bytesDownloaded := atomic.LoadUint64(&g.bytesDownloaded)
	bytesOverhead := atomic.LoadUint64(&g.bytesOverhead)
	bytesSaved := atomic.LoadUint64(&g.bytesSaved)

	fmt.Println(filesDownloaded, "/", filesFound-alreadyExists, "files downloaded.")
	if alreadyExists != 0 {
		fmt.Println(alreadyExists, "previously downloaded.")
	}
	if hardlinked != 0 {
		fmt.Println(hardlinked, "new hardlinks.")
	}
	fmt.Println(byteSize(bytesDownloaded), "of files downloaded during this session.")
	fmt.Println(byteSize(bytesOverhead), "of data downloaded as JSON overhead.")
	fmt.Println(byteSize(bytesSaved), "of bandwidth saved due to hardlinking.")
}
