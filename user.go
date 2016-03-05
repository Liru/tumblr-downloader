package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/cheggaaa/pb"
)

var userVerificationRegex = regexp.MustCompile(`[A-Za-z0-9]*`)

// User represents a tumblr user blog. It stores details that help
// to download files efficiently.
type User struct {
	name, tag     string
	lastPostID    int64
	highestPostID int64
	progressBar   *pb.ProgressBar
}

func newUser(name string) (*User, error) {

	if !userVerificationRegex.MatchString(name) {
		return nil, errors.New("newUser: Invalid username format: " + name)
	}

	query := fmt.Sprintf("https://api.tumblr.com/v2/blog/%s.tumblr.com/avatar/16", name)
	resp, err := http.Get(query)
	if err != nil {
		return nil, errors.New("newUser: Couldn't connect to tumblr to check user validity")
	}
	defer resp.Body.Close()

	var js map[string]interface{}
	contents, _ := ioutil.ReadAll(resp.Body)

	// Valid users return images from this call, even default ones.
	// If there is no error while unmarshaling this, then we have valid json.
	// Which means that this is an invalid user.
	if json.Unmarshal(contents, &js) == nil {
		return nil, errors.New("newUser: User not found: " + name)
	}

	return &User{
		name:          name,
		lastPostID:    0,
		highestPostID: 0,
	}, nil
}
