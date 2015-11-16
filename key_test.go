package main

import (
	"testing"
)

type commentTest [3]string

var commentTests = []commentTest{
	[...]string{"hello", "hello", ""},
	[...]string{"hello (hi) bye (bye)", "hello  bye ", "hibye"},
	[...]string{`hello \(hi) goodbye (bye)`, "hello (hi) goodbye ", "bye"},
	[...]string{`hello \(hi) goodbye (buh\)bye)`, "hello (hi) goodbye ", "buh)bye"},
}

func TestRemoveComments(t *testing.T) {
	for _, value := range commentTests {
		a, b := removeComments(value[0])
		if !(a == value[1] && b == value[2]) {
			t.Error("For", value[0], "expected", value[1], value[2], "got", a, b)
		}
	}
}

type uidTest [5]string

var uidTests = []uidTest{
	[...]string{"Fred Noname (lol) <awesome@neato.com>",
		"Fred Noname", "lol", "awesome@neato.com", "neato.com"},

	[...]string{"Fred Doe <awesome@neato.com>",
		"Fred Doe", "", "awesome@neato.com", "neato.com"},

	[...]string{"Invalid",
		"", "", "", ""},
	
	[...]string{"Frank Wöckener <fwoeck@dokmatic.com>",
		"Frank Wöckener", "", "fwoeck@dokmatic.com", "dokmatic.com"},
}

func TestParseUID(t *testing.T) {
	for _, expected := range uidTests {
		got := parseUID(expected[0])
		t.Log(got)
		if got.name != expected[1] {
			t.Error("Name mismatch", expected[1], got.name)
		}

		if got.comment != expected[2] {
			t.Error("comment mismatch", expected[2], got.comment)
		}
		if got.email != expected[3] {
			t.Error("email mismatch", expected[3], got.email)
		}
		if got.domain != expected[4] {
			t.Error("domain mismatch", expected[4], got.domain)
		}
	}
}
