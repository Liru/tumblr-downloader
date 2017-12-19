package main

import (
	"testing"
	"time"
)

func TestShouldFinishScraping(t *testing.T) {
	t.Parallel()

	makeChans := func() (chan time.Time, chan struct{}) {
		l := make(chan time.Time, 1)
		d := make(chan struct{}, 1)
		return l, d
	}

	tests := []struct {
		name   string
		fn     func() (chan time.Time, chan struct{})
		result bool
	}{
		{"LimiterBasic", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			l <- time.Now()
			return l, d
		}, false},
		{"CloseBasic", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			close(d)
			return l, d
		}, true},
		{"BothBasic", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			l <- time.Now()
			close(d)
			return l, d
		}, true},
		{"CloseBeforeLimiter", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			go func() {
				close(d)
				time.Sleep(time.Second)
				l <- time.Now()
			}()
			return l, d
		}, true},
		{"LimiterBeforeClose", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			go func() {
				l <- time.Now()
				time.Sleep(time.Second)
				close(d)
			}()
			return l, d
		}, false},
		{"DelayedCloseBeforeLimiter", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			go func() {
				time.Sleep(time.Second)
				close(d)
				time.Sleep(time.Second)
				l <- time.Now()
			}()
			return l, d
		}, true},
		{"DelayedLimiterBeforeClose", func() (chan time.Time, chan struct{}) {
			l, d := makeChans()
			go func() {
				time.Sleep(time.Second)
				l <- time.Now()
				time.Sleep(time.Second)
				close(d)
			}()
			return l, d
		}, false},
	}

	for i, test := range tests {
		l, d := test.fn()
		result := shouldFinishScraping(l, d)
		if result != test.result {
			t.Errorf("#%d: shouldFinishScraping(%s)=%t; want %t",
				i, test.name, result, test.result)
		}
	}
}

func TestParsePhotoPost(t *testing.T) {
	t.Parallel()
	t.Skip("Test not implemented")
}

func TestParseAnswerPost(t *testing.T) {
	t.Parallel()
	t.Skip("Test not implemented")
}

func TestParseRegularPost(t *testing.T) {
	t.Parallel()
	t.Skip("Test not implemented")
}

func TestParseVideoPost(t *testing.T) {
	t.Parallel()
	t.Skip("Test not implemented")
}

func TestParseAudioPost(t *testing.T) {
	t.Parallel()
	t.Skip("Feature not implemented")
}

// TestParseData tests parseDataForFiles. Aside from the other "Parse"
// tests above, it also checks if there is an invalid post type.
func TestParseData(t *testing.T) {
	t.Parallel()
	t.Skip("Test not implemented")
}

func TestMakeTumblrURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		u      *User
		page   int
		result string
	}{
		{
			&User{name: "InitialUser"},
			1, "https://InitialUser.tumblr.com/api/read/json?num=50&start=0",
		}, {
			&User{name: "OtherPageUser"},
			2, "https://OtherPageUser.tumblr.com/api/read/json?num=50&start=50",
		}, {
			&User{name: "TaggedInitialUser", tag: "test"},
			1, "https://TaggedInitialUser.tumblr.com/api/read/json?num=50&start=0&tagged=test",
		}, {
			&User{name: "TaggedOtherPageUser", tag: "test"},
			2, "https://TaggedOtherPageUser.tumblr.com/api/read/json?num=50&start=50&tagged=test",
		},
	}

	for i, test := range tests {
		result := makeTumblrURL(test.u, test.page).String()

		if result != test.result {
			t.Errorf("#%d: makeTumblrURL(%s)=%s; want %s",
				i, test.u.name, result, test.result)
		}
	}
}
