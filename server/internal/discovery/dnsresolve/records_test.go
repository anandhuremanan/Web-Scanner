package dnsresolve

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"testing"

	"websitemapper/internal/discovery"
)

// fakeRecords answers record questions for one domain and counts them.
type fakeRecords struct {
	mu  sync.Mutex
	mx  []*net.MX
	ns  []*net.NS
	txt []string
	// policies are TXT records at other names, such as _dmarc.example.com.
	policies map[string][]string
	asked    int
}

func (f *fakeRecords) count() {
	f.mu.Lock()
	f.asked++
	f.mu.Unlock()
}

func (f *fakeRecords) LookupMX(context.Context, string) ([]*net.MX, error) {
	f.count()
	if f.mx == nil {
		return nil, errors.New("no MX")
	}
	return f.mx, nil
}

func (f *fakeRecords) LookupNS(context.Context, string) ([]*net.NS, error) {
	f.count()
	return f.ns, nil
}

func (f *fakeRecords) LookupTXT(_ context.Context, name string) ([]string, error) {
	f.count()
	if strings.HasPrefix(name, "_") {
		if t, ok := f.policies[name]; ok {
			return t, nil
		}
		return nil, errors.New("no such name")
	}
	return f.txt, nil
}

func TestDomainRecordsAndServices(t *testing.T) {
	r := &fakeRecords{
		mx:  []*net.MX{{Host: "alt1.aspmx.l.google.com.", Pref: 5}, {Host: "ASPMX.L.GOOGLE.COM.", Pref: 1}},
		ns:  []*net.NS{{Host: "lara.ns.cloudflare.com."}, {Host: "bob.ns.cloudflare.com."}},
		txt: []string{"v=spf1 include:_spf.google.com include:sendgrid.net ~all", "google-site-verification=abc123", "MS=ms4455", strings.Repeat("x", 2000)},
	}
	got := domainRecords(context.Background(), r, "example.com")
	if got == nil {
		t.Fatal("no records")
	}
	wantMX := []discovery.MXRecord{{Host: "aspmx.l.google.com", Priority: 1}, {Host: "alt1.aspmx.l.google.com", Priority: 5}}
	if !reflect.DeepEqual(got.MX, wantMX) {
		t.Errorf("MX = %+v", got.MX)
	}
	if !reflect.DeepEqual(got.NS, []string{"bob.ns.cloudflare.com", "lara.ns.cloudflare.com"}) {
		t.Errorf("NS = %v", got.NS)
	}
	// A very long record is cut, not dropped.
	long := false
	for _, s := range got.TXT {
		long = long || (len(s) == maxTXTLength+3 && strings.HasSuffix(s, "..."))
	}
	if len(got.TXT) != 4 || !long {
		t.Errorf("TXT = %d records, long one cut: %v", len(got.TXT), long)
	}
	var names []string
	for _, s := range got.Services {
		names = append(names, s.Name)
		if s.Evidence == "" {
			t.Errorf("service %s has no evidence", s.Name)
		}
	}
	want := []string{"Cloudflare (DNS)", "Google (site verification)", "Google Workspace (mail)", "Google Workspace (sends mail)",
		"Microsoft 365 (domain verification)", "SendGrid (sends mail)"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("services = %v\nwant       %v", names, want)
	}

	// Nothing readable is nil, not an empty record set.
	if got := domainRecords(context.Background(), &fakeRecords{}, "example.com"); got != nil {
		t.Errorf("empty = %+v", got)
	}
}

// The engine reads the records for the target domain only.
func TestEngineReadsRecordsForTheDomainOnly(t *testing.T) {
	rec := &fakeRecords{ns: []*net.NS{{Host: "ns1.example.net."}}}
	r := &fakeResolver{addrs: map[string][]string{"example.com": {"93.184.216.34"}, "www.example.com": {"93.184.216.34"}}}
	got := run(t, r, Options{MaxHosts: 10, Concurrency: 1, Records: rec},
		discovery.HostView{Name: "example.com"}, discovery.HostView{Name: "www.example.com"})
	if d := got["example.com"]; d.Records == nil || !reflect.DeepEqual(d.Records.NS, []string{"ns1.example.net"}) {
		t.Errorf("domain records = %+v", d.Records)
	}
	if got["www.example.com"].Records != nil {
		t.Error("a subdomain must not carry the domain's records")
	}
	if rec.asked != 6 {
		t.Errorf("record questions = %d, want 6 (MX, NS, TXT and three mail policies, once)", rec.asked)
	}
}

// aliasLookuper answers like a DNS server would for the given names.
type aliasLookuper struct{}

func (aliasLookuper) Lookup(_ context.Context, host string) ([]netip.Addr, string, error) {
	notFound := &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	switch host {
	case "shop.example.com": // an alias whose target was deleted
		return nil, "old-shop.herokuapp.com.", notFound
	case "gone.example.com": // simply does not exist
		return nil, "", notFound
	case "ok.example.com": // an alias that works
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, "ok.cdn.example.net.", nil
	}
	return nil, "", notFound
}

func TestDanglingAlias(t *testing.T) {
	tgt, _ := discovery.ParseTarget("example.com")
	var mu sync.Mutex
	got := map[string]*discovery.DNSInfo{}
	hosts := []discovery.HostView{{Name: "shop.example.com"}, {Name: "gone.example.com"}, {Name: "ok.example.com"}}
	err := NewWith(aliasLookuper{}, Options{MaxHosts: 10, Concurrency: 1}).Discover(context.Background(),
		discovery.Input{Target: tgt, State: state{hosts}}, func(f discovery.Finding) {
			mu.Lock()
			got[f.Host] = f.DNS
			mu.Unlock()
		})
	if err != nil {
		t.Fatal(err)
	}
	if d := got["shop.example.com"]; d.Resolved || !d.DanglingAlias || d.CNAME != "old-shop.herokuapp.com" {
		t.Errorf("deleted target = %+v", d)
	}
	if d := got["gone.example.com"]; d.DanglingAlias || d.CNAME != "" || d.Error != "no such host" {
		t.Errorf("missing name = %+v", d)
	}
	if d := got["ok.example.com"]; !d.Resolved || d.DanglingAlias || d.CNAME != "ok.cdn.example.net" {
		t.Errorf("working alias = %+v", d)
	}
}

func TestMailSecurity(t *testing.T) {
	r := &fakeRecords{
		txt: []string{"v=spf1 include:_spf.google.com -all", "google-site-verification=abc"},
		policies: map[string][]string{
			"_dmarc.example.com":     {"v=DMARC1; p=reject; rua=mailto:dmarc@example.com"},
			"_smtp._tls.example.com": {"v=TLSRPTv1; rua=mailto:tls@example.com"},
			"_mta-sts.example.com":   {"something else"},
		},
	}
	got := domainRecords(context.Background(), r, "example.com").Mail
	if got == nil {
		t.Fatal("no mail security")
	}
	if !reflect.DeepEqual(got.SPF, []string{"v=spf1 include:_spf.google.com -all"}) {
		t.Errorf("SPF = %v", got.SPF)
	}
	if len(got.DMARC) != 1 || len(got.TLSRPT) != 1 {
		t.Errorf("DMARC = %v, TLS-RPT = %v", got.DMARC, got.TLSRPT)
	}
	// A record that is not a policy is not reported as one.
	if len(got.MTASTS) != 0 {
		t.Errorf("MTA-STS = %v", got.MTASTS)
	}

	// None published at all: nothing to report.
	if m := domainRecords(context.Background(), &fakeRecords{ns: []*net.NS{{Host: "ns1.example.net."}}}, "example.com").Mail; m != nil {
		t.Errorf("no policies = %+v", m)
	}
}
