package main

import (
	"testing"
	"time"
)

func TestStrIntCmp(t *testing.T) {
	var tests = []struct {
		oldVal, newVal string
		result         bool
	}{
		// Test basic strings.
		{"0", "1", true},
		{"1", "0", false},
		//Test string length.
		{"9", "10", true},
		{"10", "9", false},
		{"500", "1000", true},
		{"123456789", "987654321", true},
		{"123456789", "9876543210", true},
		{"1234567890", "987654321", false},
		{"0", "10", true},
		{"314", "159", false},
	}

	for i, test := range tests {
		result := strIntLess(test.oldVal, test.newVal)
		if result != test.result {
			t.Errorf("#%d: strIntLess(%s,%s)=%t; want %t",
				i, test.oldVal, test.newVal, result, test.result)
		}
	}
}

func TestShouldFinishScraping(t *testing.T) {

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
