package main

import (
	"testing"
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
