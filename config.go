package main

import (
	"time"

	"github.com/blang/semver"
)

type Config struct {
	numDownloaders    int
	requestRate       int
	updateMode        bool
	forceCheck        bool
	serverMode        bool
	serverSleep       time.Duration
	downloadDirectory string

	ignorePhotos   bool
	ignoreVideos   bool
	ignoreAudio    bool
	useProgressBar bool

	Version semver.Version
}
