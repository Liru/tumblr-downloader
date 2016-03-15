package main

import "encoding/json"

// A Post is reflective of the JSON used in the tumblr API.
// It contains a PhotoURL, and, optionally, an array of photos.
// If Photos isn't empty, it typically contains at least one URL which matches PhotoURL.
type Post struct {
	// ID is a json.Number because "early" versions of tumblr's API use
	// strings to denote ID. However, the ID now randomly switches to int
	// because Tumblr a shit. We have to analyze both types of response.
	//
	// TODO: IDs start as int, then switch to string later. Possible caching
	// bug on tumblr's end?
	ID            json.Number `json:",Number"`
	Type          string
	PhotoURL      string `json:"photo-url-1280"`
	Photos        []Post `json:"photos,omitempty"`
	UnixTimestamp int64  `json:"unix-timestamp"`
	PhotoCaption  string `json:"photo-caption"`

	// for regular posts
	RegularBody string `json:"regular-body"`

	// for answer posts
	Answer string

	// for videos
	Video        string `json:"video-player"`
	VideoCaption string `json:"video-caption"` // For links to outside sites.
}

// A TumbleLog is the outer container for Posts. It is necessary for easier JSON deserialization,
// even though it's useless in and of itself.
type TumbleLog struct {
	Posts      []Post `json:"posts"`
	TotalPosts int    `json:"posts-total"`
}
