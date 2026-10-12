package dnsresolve

import (
	"context"
	"errors"
	"net/netip"
	"sort"
	"sync"
	"testing"

	"websitemapper/internal/discovery"
)

// fakeTXT answers TXT questions from a map and counts them.
type fakeTXT struct {
	mu      sync.Mutex
	answers map[string]string
	asked   map[string]int
	failing map[string]int // name -> how many times to fail first
}

func (f *fakeTXT) LookupTXT(_ context.Context, name string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.asked == nil {
		f.asked = map[string]int{}
	}
	f.asked[name]++
	if f.failing[name] > 0 {
		f.failing[name]--
		return nil, errors.New("timeout")
	}
	if a, ok := f.answers[name]; ok {
		return []string{a}, nil
	}
	return nil, errors.New("no such host")
}

const cloudflareV6 = "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.7.4.6.0.6.2.origin6.asn.cymru.com"

func cymru() *fakeTXT {
	return &fakeTXT{answers: map[string]string{
		"225.211.79.145.origin.asn.cymru.com": "47583 | 145.79.208.0/21 | IN | ripencc | 2023-01-01",
		"AS47583.asn.cymru.com":               "47583 | CY | ripencc | 2011-04-04 | AS-HOSTINGER, CY",
		cloudflareV6:                          "13335 | 2606:4700::/44 | US | arin | 2011-11-01",
		"AS13335.asn.cymru.com":               "13335 | US | arin | 2010-07-14 | CLOUDFLARENET - Cloudflare, Inc., US",
		"4.3.2.1.origin.asn.cymru.com":        "64500 64501 | 1.2.3.0/24 | US | arin | 2020-01-01",
		"AS64500.asn.cymru.com":               "64500 | US | arin | 2020-01-01 | EXAMPLE-NET - Example Inc., US",
		"9.9.9.9.origin.asn.cymru.com":        "19281 | 9.9.9.0/24 | US | arin | 2017-01-01",
	}}
}

func TestNetworkLookup(t *testing.T) {
	txt := cymru()
	n := NewNetworks(txt)
	ctx := context.Background()

	got, ok := n.Lookup(ctx, netip.MustParseAddr("145.79.211.225"))
	want := discovery.Network{ASN: 47583, Name: "AS-HOSTINGER", Country: "IN", Prefix: "145.79.208.0/21"}
	if !ok || got != want {
		t.Errorf("IPv4 = %+v, %v; want %+v", got, ok, want)
	}
	got, ok = n.Lookup(ctx, netip.MustParseAddr("2606:4700::1"))
	want = discovery.Network{ASN: 13335, Name: "Cloudflare, Inc.", Country: "US", Prefix: "2606:4700::/44"}
	if !ok || got != want {
		t.Errorf("IPv6 = %+v, %v; want %+v", got, ok, want)
	}
	// An address announced by two networks: the first is used.
	if got, ok := n.Lookup(ctx, netip.MustParseAddr("1.2.3.4")); !ok || got.ASN != 64500 || got.Name != "Example Inc." {
		t.Errorf("two origins = %+v, %v", got, ok)
	}
	// Nobody announces it.
	if _, ok := n.Lookup(ctx, netip.MustParseAddr("203.0.113.9")); ok {
		t.Error("an unknown address must not get a network")
	}

	// Answers are remembered: asking again asks nobody.
	before := txt.asked["225.211.79.145.origin.asn.cymru.com"]
	n.Lookup(ctx, netip.MustParseAddr("145.79.211.225"))
	if txt.asked["225.211.79.145.origin.asn.cymru.com"] != before {
		t.Error("a remembered address was looked up again")
	}
}

func TestNetworkLookupRetriesAndToleratesAMissingName(t *testing.T) {
	txt := cymru()
	txt.failing = map[string]int{"225.211.79.145.origin.asn.cymru.com": 1}
	n := NewNetworks(txt)
	if got, ok := n.Lookup(context.Background(), netip.MustParseAddr("145.79.211.225")); !ok || got.ASN != 47583 {
		t.Errorf("after one failure = %+v, %v", got, ok)
	}
	// The network is known but its name cannot be read: still reported,
	// and asked for again next time rather than remembered without a name.
	got, ok := n.Lookup(context.Background(), netip.MustParseAddr("9.9.9.9"))
	if !ok || got.ASN != 19281 || got.Name != "" {
		t.Errorf("no name = %+v, %v", got, ok)
	}
	txt.mu.Lock()
	txt.answers["AS19281.asn.cymru.com"] = "19281 | US | arin | 2017-01-01 | QUAD9-AS-1 - Quad9, US"
	txt.mu.Unlock()
	if got, _ := n.Lookup(context.Background(), netip.MustParseAddr("9.9.9.9")); got.Name != "Quad9" {
		t.Errorf("name on the next lookup = %q", got.Name)
	}
}

func TestOperatorName(t *testing.T) {
	for in, want := range map[string]string{
		"CLOUDFLARENET - Cloudflare, Inc., US": "Cloudflare, Inc.",
		"AS-HOSTINGER, CY":                     "AS-HOSTINGER",
		"AMAZON-02 - Amazon.com, Inc., US":     "Amazon.com, Inc.",
		"GOOGLE, US":                           "GOOGLE",
		"Some Network":                         "Some Network",
	} {
		if got := operatorName(in); got != want {
			t.Errorf("operatorName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The engine adds networks to resolved hosts, skipping private addresses.
func TestEngineAddsNetworks(t *testing.T) {
	r := &fakeResolver{addrs: map[string][]string{
		"example.com":       {"145.79.211.225", "2606:4700::1"},
		"intra.example.com": {"10.0.0.8"},
	}}
	got := run(t, r, Options{MaxHosts: 10, Concurrency: 1, Networks: NewNetworks(cymru())},
		discovery.HostView{Name: "example.com"}, discovery.HostView{Name: "intra.example.com"})
	var asns []int
	for _, n := range got["example.com"].Networks {
		asns = append(asns, n.ASN)
	}
	sort.Ints(asns)
	if len(asns) != 2 || asns[0] != 13335 || asns[1] != 47583 {
		t.Errorf("example.com networks = %v", asns)
	}
	if n := got["intra.example.com"].Networks; len(n) != 0 {
		t.Errorf("a private address got a network: %+v", n)
	}
}
