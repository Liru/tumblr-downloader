package main

import (
	"testing"
)

func TestByteSize(t *testing.T) {
	var tests = []struct {
		size   uint64
		result string
	}{
		// Basic power tests.
		{0, "0B"},
		{1024, "1.00KB"},
		{1024 * 1024, "1.00MB"},
		{1024 * 1024 * 1024, "1.00GB"},
		{1024 * 1024 * 1024 * 1024, "1.00TB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.00PB"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.00EB"},

		{500, "500B"},
		{1000, "1000B"},
		{1030, "1.01KB"}, // Test for rounding. 1030B =~ 1.00586KB
		{2000, "1.95KB"},
		{1.5 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.50EB"},
	}

	for i, test := range tests {
		result := byteSize(test.size)
		if result != test.result {
			t.Errorf("#%d: byteSize(%d)=%s; want %s",
				i, test.size, result, test.result)
		}
	}
}
