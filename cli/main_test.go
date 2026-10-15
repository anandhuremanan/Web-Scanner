package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"websitemapper/internal/discovery"
	"websitemapper/internal/results"
	"websitemapper/internal/scan"
	"websitemapper/internal/store"
)

func TestParseArgs(t *testing.T) {
	o, err := parseArgs([]string{"-mode", "full", "-pages", "50", "-print", "urls", "example.com"}, io.Discard)
	if err != nil || o.mode != scan.ModeFull || o.pages != 50 || o.print != "urls" || o.target != "example.com" {
		t.Errorf("parsed %+v, %v", o, err)
	}
	if o, err := parseArgs([]string{"example.com"}, io.Discard); err != nil || o.mode != scan.ModeLight {
		t.Errorf("default mode = %v, %v; want standard (light)", o.mode, err)
	}
	for _, bad := range [][]string{
		{},
		{"a.com", "b.com"},
		{"-mode", "deep", "a.com"},
		{"-print", "everything", "a.com"},
		{"-pages", "0", "a.com"},
	} {
		if _, err := parseArgs(bad, io.Discard); err == nil {
			t.Errorf("parseArgs(%q) accepted", bad)
		}
	}
}

func TestUsageAndVersionExitCleanly(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"-version"}, io.Discard, &out); code != exitOK || !strings.Contains(out.String(), "webscanner") {
		t.Errorf("-version: code %d, %q", code, out.String())
	}
	if code := run([]string{"-mode", "x", "a.com"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("bad mode exit = %d", code)
	}
	// An invalid target is refused before anything is contacted.
	if code := run([]string{"-quiet", "-no-files", "-no-memory", "not a domain"}, io.Discard, io.Discard); code != exitUsage {
		t.Errorf("invalid target exit = %d", code)
	}
}

func TestHelpers(t *testing.T) {
	at := time.Date(2026, 10, 15, 9, 30, 5, 0, time.UTC)
	if got := defaultOut("shop.example.com", at); got != "webscanner-shop.example.com-20261015-093005" {
		t.Errorf("defaultOut = %q", got)
	}
	for in, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567"} {
		if got := num(in); got != want {
			t.Errorf("num(%d) = %q", in, got)
		}
	}
	if plainCell("=HYPERLINK(1)") != "'=HYPERLINK(1)" || plainCell("Home") != "Home" {
		t.Error("plainCell must defuse formulas only")
	}
}

// storedResult records a small scan in a store and opens its result.
func storedResult(t *testing.T) scan.ResultReader {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(store.Options{Dir: t.TempDir(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Create(ctx, scan.Scan{ID: "s1", Status: scan.StatusRunning}); err != nil {
		t.Fatal(err)
	}
	urls, err := s.OpenURLs(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	tgt, _ := discovery.ParseTarget("example.com")
	a := results.NewAggregator(tgt, results.Limits{}, urls)
	a.Add(discovery.Finding{Host: "example.com", Source: discovery.SourceTarget})
	a.Add(discovery.Finding{Host: "example.com", DNS: &discovery.DNSInfo{Resolved: true, Addresses: []string{"93.184.216.34"},
		Networks: []discovery.Network{{ASN: 15133, Name: "Edgecast", Country: "US"}}}})
	a.Add(discovery.Finding{Host: "api.example.com", Source: discovery.SourceCT})
	a.Add(discovery.Finding{URL: "https://example.com/about", Source: discovery.SourceSitemap, Hint: discovery.HintSitemap})
	a.Add(discovery.Finding{URL: "https://example.com/=cmd", Source: discovery.SourceHTML, Hint: discovery.HintLink,
		Response: &discovery.Response{Status: 200, ContentType: "text/html", Title: "=SUM(A1)"}})
	a.Add(discovery.Finding{URL: "https://api.example.com/v1/items", Source: discovery.SourceArchive, Hint: discovery.HintArchive})
	res := a.Result()
	if err := s.SaveResult(ctx, scan.Result{ScanID: "s1", Status: scan.StatusCompleted, Counts: a.Counts(), Result: res}); err != nil {
		t.Fatal(err)
	}
	urls.Close()
	rd, err := s.OpenResult(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rd.Close() })
	return rd
}

func TestWriteFiles(t *testing.T) {
	rd := storedResult(t)
	dir := filepath.Join(t.TempDir(), "out")
	files, err := writeFiles(context.Background(), dir, rd)
	if err != nil || len(files) != 3 {
		t.Fatalf("files %v, err %v", files, err)
	}

	// result.json: valid JSON, hosts with their URLs.
	b, _ := os.ReadFile(filepath.Join(dir, "result.json"))
	var doc struct {
		Status    string `json:"status"`
		Providers []struct {
			ASN int `json:"asn"`
		} `json:"providers"`
		Hosts []struct {
			Hostname string `json:"hostname"`
			URLs     []struct {
				URL string `json:"url"`
			} `json:"urls"`
		} `json:"hosts"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("result.json is not valid JSON: %v\n%s", err, b)
	}
	total := 0
	for _, h := range doc.Hosts {
		total += len(h.URLs)
	}
	if doc.Status != "completed" || len(doc.Hosts) != 2 || total != 3 || len(doc.Providers) != 1 {
		t.Errorf("result.json = %+v", doc)
	}

	// urls.csv: one row per URL, and page text cannot run as a formula.
	rows := readCSV(t, filepath.Join(dir, "urls.csv"))
	if len(rows) != 4 || rows[0][0] != "url" {
		t.Fatalf("urls.csv rows = %v", rows)
	}
	for _, r := range rows[1:] {
		if r[0] == "https://example.com/=cmd" && r[7] != "'=SUM(A1)" {
			t.Errorf("title cell = %q", r[7])
		}
	}

	hosts := readCSV(t, filepath.Join(dir, "hosts.csv"))
	if len(hosts) != 3 || hosts[1][0] != "example.com" || hosts[1][5] != "Edgecast" || hosts[1][6] != "15133" {
		t.Errorf("hosts.csv = %v", hosts)
	}
}

func TestPrintList(t *testing.T) {
	rd := storedResult(t)
	var hosts, urls bytes.Buffer
	if err := printList(context.Background(), &hosts, rd, "hosts"); err != nil {
		t.Fatal(err)
	}
	if err := printList(context.Background(), &urls, rd, "urls"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(hosts.String()); len(got) != 2 || got[0] != "example.com" {
		t.Errorf("hosts = %q", got)
	}
	if got := strings.Fields(urls.String()); len(got) != 3 {
		t.Errorf("urls = %q", got)
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
