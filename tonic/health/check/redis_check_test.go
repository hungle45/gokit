package check

import (
	"reflect"
	"testing"
)

func TestParseRedisInfo(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
	}{
		{name: "empty", in: "", want: map[string]string{}},
		{name: "single", in: "k:1\r\n", want: map[string]string{"k": "1"}},
		{name: "comments", in: "# a\r\nk1:v1\r\n", want: map[string]string{"k1": "v1"}},
		{name: "malformed", in: "one\r\ntwo:2\r\nthree:3:extra\r\n", want: map[string]string{"two": "2", "three": "3:extra"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRedisInfo(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRedisCheckerName(t *testing.T) {
	c := NewRedisChecker(nil, false)
	if c.Name() != "redis" {
		t.Fatalf("expected redis name, got %s", c.Name())
	}
}
