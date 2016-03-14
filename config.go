package main

import (
	"time"

	"github.com/blang/semver"
	"github.com/burntsushi/toml"
)

// Config is a struct that contains all the configuration options
// and possibilities for the downloader to run.
type Config struct {
	numDownloaders    int           `toml:"num_downloaders"`
	requestRate       int           `toml:"rate"`
	updateMode        bool          `toml:"update_mode"`
	forceCheck        bool          `toml:"force"`
	serverMode        bool          `toml:"server_mode"`
	serverSleep       time.Duration `toml:"sleep_time"`
	downloadDirectory string        `toml:"directory"`

	ignorePhotos   bool `toml:"ignore_photos"`
	ignoreVideos   bool `toml:"ignore_videos"`
	ignoreAudio    bool `toml:"ignore_audio"`
	useProgressBar bool `toml:"use_progress_bar"`

	version semver.Version // don't want to be able to decode into this
}

func loadConfig() {
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {
		// TODO: Do something if config.toml isn't detected.
	}
}
