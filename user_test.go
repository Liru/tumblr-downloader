package main

import "testing"

func TestNewUser(t *testing.T) {

	var tests = []struct {
		name   string
		result *User
		err    error
	}{
		{name: "demo", result: &User{name: "demo"}},
	}

	for i, test := range tests {
		u, _ := newUser(test.name)

		if u.name != test.result.name {
			t.Errorf("#%d: newUser(%s).name = %s, want %s",
				i, test.name, u.name, test.result.name)
		}

		if u.lastPostID != test.result.lastPostID {
			t.Errorf("#%d: newUser(%s).lastPostID = %d, want %d",
				i, test.name, u.lastPostID, test.result.lastPostID)
		}

		if u.highestPostID != test.result.highestPostID {
			t.Errorf("#%d: newUser(%s).highestPostID = %d, want %d",
				i, test.name, u.highestPostID, test.result.highestPostID)
		}
	}
}
