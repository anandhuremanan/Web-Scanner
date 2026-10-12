package store_test

import (
	"reflect"
	"sort"
	"testing"

	"websitemapper/internal/discovery"
	"websitemapper/internal/results"
	"websitemapper/internal/scan"
	"websitemapper/internal/store"
	"websitemapper/internal/store/storetest"
)

// TestHostsByNetwork: hosts can be listed by the network they are served
// from, and the result summarizes the networks.
func TestHostsByNetwork(t *testing.T) {
	s := storetest.New(t, store.Options{})
	cf := discovery.Network{ASN: 13335, Name: "Cloudflare, Inc.", Country: "US"}
	hz := discovery.Network{ASN: 47583, Name: "AS-HOSTINGER", Country: "IN"}
	record(t, s, "net", func(a *results.Aggregator) {
		add := func(host string, nets ...discovery.Network) {
			a.Add(discovery.Finding{Host: host, Source: discovery.SourceCT})
			a.Add(discovery.Finding{Host: host, DNS: &discovery.DNSInfo{Resolved: true, Addresses: []string{"203.0.113.1"}, Networks: nets}})
		}
		add("example.com", cf)
		add("www.example.com", cf)
		add("api.example.com", hz)
		add("both.example.com", cf, hz)
		add("plain.example.com")
		a.Add(discovery.Finding{Host: "gone.example.com", Source: discovery.SourceCT})
	})
	r, err := s.OpenResult(ctx, "net")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	want := []results.Provider{
		{ASN: 13335, Name: "Cloudflare, Inc.", Country: "US", Hosts: 3},
		{ASN: 47583, Name: "AS-HOSTINGER", Country: "IN", Hosts: 2},
	}
	if got := r.Summary().Providers; !reflect.DeepEqual(got, want) {
		t.Errorf("providers = %+v, want %+v", got, want)
	}
	names := func(q scan.HostQuery) []string {
		page, err := r.Hosts(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, h := range page.Hosts {
			out = append(out, h.Hostname)
		}
		sort.Strings(out)
		return out
	}
	if got := names(scan.HostQuery{ASN: 47583}); !reflect.DeepEqual(got, []string{"api.example.com", "both.example.com"}) {
		t.Errorf("hosts in 47583 = %v", got)
	}
	if got := names(scan.HostQuery{ASN: 13335, Search: "www"}); !reflect.DeepEqual(got, []string{"www.example.com"}) {
		t.Errorf("hosts in 13335 matching www = %v", got)
	}
	if got := names(scan.HostQuery{ASN: 64999}); len(got) != 0 {
		t.Errorf("hosts in an unknown network = %v", got)
	}
	if got := names(scan.HostQuery{}); len(got) != 6 {
		t.Errorf("all hosts = %v", got)
	}
}
