package main

import (
	"log"
	"time"

	"github.com/blang/semver"
	"github.com/burntsushi/toml"
)

var cfg Config

// Config is a struct that contains all the configuration options
// and possibilities for the downloader to run.
type Config struct {
	NumDownloaders    int           `toml:"num_downloaders"`
	RequestRate       int           `toml:"rate"`
	ForceCheck        bool          `toml:"force"`
	ServerMode        bool          `toml:"server_mode"`
	ServerSleep       time.Duration `toml:"sleep_time"`
	DownloadDirectory string        `toml:"directory"`

	IgnorePhotos   bool `toml:"ignore_photos"`
	IgnoreVideos   bool `toml:"ignore_videos"`
	IgnoreAudio    bool `toml:"ignore_audio"`
	UseProgressBar bool `toml:"use_progress_bar"`

	version semver.Version // don't want to be able to decode into this
}

func loadConfig() {
	var err error
	if _, err = toml.DecodeFile("config.toml", &cfg); err != nil {
		// TODO: Do something if config.toml isn't detected.
		log.Fatal(err)
	}
	if cfg.DownloadDirectory == "" {
		cfg.DownloadDirectory = "."
	}
}
