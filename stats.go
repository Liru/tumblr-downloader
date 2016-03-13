package main

import "sync"

var gStats = &GlobalStats{}

// GlobalStats keeps track of various statistics during a download process.
type GlobalStats struct {
	filesDownloaded uint64
	filesFound      uint64
	alreadyExists   uint64

	totalSizeDownloaded uint64

	// NowScraping is used to show which blogs are being scraped.
	nowScraping Tracker
}

// Tracker is a record-keeping structure that tracks which users are still
// in the scraping/downloading phase.
type Tracker struct {
	sync.RWMutex
	Blog map[string]bool
}
