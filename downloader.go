package main

import (
	"sync/atomic"
	"time"
)

func downloader(id int, limiter <-chan time.Time, img <-chan image) {
	atomic.AddUint64(&totalDownloaded, 1)
}
