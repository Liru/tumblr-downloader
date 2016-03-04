package main

import "github.com/cheggaaa/pb"

type User struct {
	name, tag     string
	lastPostID    int64
	highestPostID int64
	progressBar   *pb.ProgressBar
}
