package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"websitemapper/internal/results"
	"websitemapper/internal/scan"
)

// writeFiles writes a result to dir: result.json (everything), hosts.csv and
// urls.csv. Hosts and URLs are read from the store and written one at a
// time, so a result of any size is never held in memory.
func writeFiles(ctx context.Context, dir string, rd scan.ResultReader) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	steps := []struct {
		name  string
		write func(io.Writer) error
	}{
		{"result.json", func(w io.Writer) error { return writeJSON(ctx, w, rd) }},
		{"hosts.csv", func(w io.Writer) error { return writeHostsCSV(ctx, w, rd) }},
		{"urls.csv", func(w io.Writer) error { return writeURLsCSV(ctx, w, rd) }},
	}
	var files []string
	for _, s := range steps {
		path := filepath.Join(dir, s.name)
		if err := writeFile(path, s.write); err != nil {
			return files, fmt.Errorf("writing %s: %w", path, err)
		}
		files = append(files, path)
	}
	return files, nil
}

func writeFile(path string, write func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 64<<10)
	err = write(bw)
	if ferr := bw.Flush(); err == nil {
		err = ferr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// summary is a scan.Result without "hosts"; hostOnly a results.Host without
// "urls". (A nil field with the same JSON name hides the embedded one.)
type summary struct {
	scan.Result
	Hosts *struct{} `json:"hosts,omitempty"`
}

type hostOnly struct {
	results.Host
	URLs *struct{} `json:"urls,omitempty"`
}

// writeJSON writes the result as one JSON document, the same shape as the
// website's /results endpoint: the summary, then hosts with their URLs.
func writeJSON(ctx context.Context, w io.Writer, rd scan.ResultReader) error {
	hosts, err := rd.Hosts(ctx, scan.HostQuery{})
	if err != nil {
		return err
	}
	head, err := openObject(summary{Result: rd.Summary()})
	if err != nil {
		return err
	}
	if _, err := w.Write(head); err != nil {
		return err
	}
	io.WriteString(w, `,"hosts":[`)
	for i, h := range hosts.Hosts {
		if i > 0 {
			io.WriteString(w, ",")
		}
		head, err := openObject(hostOnly{Host: h})
		if err != nil {
			return err
		}
		w.Write(head)
		io.WriteString(w, `,"urls":[`)
		first := true
		_, err = rd.URLs(ctx, scan.URLQuery{Host: h.Hostname}, func(u results.URL) error {
			b, err := json.Marshal(u)
			if err != nil {
				return err
			}
			if !first {
				io.WriteString(w, ",")
			}
			first = false
			_, err = w.Write(b)
			return err
		})
		if err != nil {
			return err
		}
		io.WriteString(w, "]}")
	}
	_, err = io.WriteString(w, "]}\n")
	return err
}

// openObject encodes v, a struct, without its closing brace.
func openObject(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	b = bytes.TrimSuffix(b, []byte("}"))
	if len(b) < 2 {
		return nil, errors.New("cannot extend an empty JSON object")
	}
	return b, nil
}

func writeHostsCSV(ctx context.Context, w io.Writer, rd scan.ResultReader) error {
	hosts, err := rd.Hosts(ctx, scan.HostQuery{})
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.Write([]string{"host", "state", "sources", "addresses", "cname", "network", "asn", "country",
		"http_status", "title", "certificate_issuer", "certificate_expires", "urls"})
	for _, h := range hosts.Hosts {
		row := []string{h.Hostname, string(h.State), joinSources(h.Sources), "", "", "", "", "", "", "", "", "",
			strconv.Itoa(h.Counts.URLs)}
		if d := h.DNS; d != nil {
			row[3], row[4] = strings.Join(d.Addresses, " "), d.CNAME
			if len(d.Networks) > 0 {
				n := d.Networks[0]
				row[5], row[6], row[7] = n.Name, strconv.Itoa(n.ASN), n.Country
			}
		}
		if hp := h.HTTP; hp != nil && hp.Reachable {
			row[8], row[9] = strconv.Itoa(hp.Status), hp.Title
			if c := hp.Certificate; c != nil {
				row[10], row[11] = c.Issuer, c.NotAfter.Format(time.DateOnly)
			}
		}
		if err := cw.Write(csvRow(row...)); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeURLsCSV(ctx context.Context, w io.Writer, rd scan.ResultReader) error {
	cw := csv.NewWriter(w)
	cw.Write([]string{"url", "host", "type", "asset_kind", "state", "status", "content_type", "title", "sources", "first_archived"})
	_, err := rd.URLs(ctx, scan.URLQuery{}, func(u results.URL) error {
		status, archived := "", ""
		if u.Status != 0 {
			status = strconv.Itoa(u.Status)
		}
		if u.Archived != nil {
			archived = u.Archived.FirstSeen.Format(time.DateOnly)
		}
		return cw.Write(csvRow(u.URL, u.Hostname, string(u.Type), string(u.AssetKind), string(u.State), status,
			u.ContentType, u.Title, joinSources(u.Sources), archived))
	})
	cw.Flush()
	if err == nil {
		err = cw.Error()
	}
	return err
}

// printList prints the result's hostnames or URLs, one per line.
func printList(ctx context.Context, w io.Writer, rd scan.ResultReader, what string) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	if what == "hosts" {
		hosts, err := rd.Hosts(ctx, scan.HostQuery{})
		if err != nil {
			return err
		}
		for _, h := range hosts.Hosts {
			fmt.Fprintln(bw, h.Hostname)
		}
		return nil
	}
	_, err := rd.URLs(ctx, scan.URLQuery{}, func(u results.URL) error {
		_, err := fmt.Fprintln(bw, u.URL)
		return err
	})
	return err
}

func joinSources[S ~string](sources []S) string {
	out := make([]string, len(sources))
	for i, s := range sources {
		out[i] = string(s)
	}
	return strings.Join(out, " ")
}

// csvRow guards every cell: a scanned site chooses more of them than the
// title (a content type, a network name).
func csvRow(cells ...string) []string {
	for i, c := range cells {
		cells[i] = plainCell(c)
	}
	return cells
}

// plainCell keeps a spreadsheet from running text taken from a web page as
// a formula.
func plainCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// printSummary prints what the scan found, its notes, and where the files are.
//
// Much of it is text the scanned domain's owner chose (DNS records, error
// messages quoting their servers), so it is passed through terminalSafe:
// control characters could otherwise retitle the terminal, plant links or
// rewrite lines already printed.
func printSummary(w io.Writer, sc scan.Scan, res scan.Result, files []string) {
	var b strings.Builder
	writeSummary(&b, sc, res, files)
	io.WriteString(w, terminalSafe(b.String()))
}

// terminalSafe replaces control characters, other than newlines and tabs,
// with a visible \xNN.
func terminalSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
			fmt.Fprintf(&b, "\\x%02x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeSummary(w io.Writer, sc scan.Scan, res scan.Result, files []string) {
	c := res.Counts
	fmt.Fprintln(w)
	switch {
	case sc.Status == scan.StatusFailed:
		fmt.Fprintf(w, "The scan of %s failed: %s\n", sc.Domain, sc.Error)
	case sc.Status == scan.StatusCancelled:
		fmt.Fprintf(w, "Stopped after %s. Kept what was found so far:\n", seconds(sc.DurationMs))
	default:
		fmt.Fprintf(w, "Finished in %s.\n", seconds(sc.DurationMs))
	}
	fmt.Fprintf(w, "  %s hosts (%s resolve, %s answer HTTP)\n", num(c.Hosts), num(c.HostsResolved), num(c.HostsReachable))
	fmt.Fprintf(w, "  %s URLs: %s pages, %s API-like, %s assets\n", num(c.URLs), num(c.Pages), num(c.APIs), num(c.Assets))

	if len(res.Providers) > 0 {
		fmt.Fprintln(w, "\nServed from:")
		for i, p := range res.Providers {
			if i == 5 {
				fmt.Fprintf(w, "  ... and %d more networks\n", len(res.Providers)-5)
				break
			}
			name := p.Name
			if name == "" {
				name = "unknown"
			}
			fmt.Fprintf(w, "  %-45s AS%-7d %s hosts\n", name, p.ASN, num(p.Hosts))
		}
	}
	if res.DNS != nil && len(res.DNS.Services) > 0 {
		fmt.Fprintln(w, "\nServices the domain's DNS records reveal:")
		for _, s := range res.DNS.Services {
			fmt.Fprintf(w, "  %s\n", s.Name)
		}
	}
	if res.DNS != nil && res.DNS.Mail != nil {
		m := res.DNS.Mail
		fmt.Fprintln(w, "\nMail security records:")
		for _, row := range []struct {
			name    string
			records []string
		}{{"SPF", m.SPF}, {"DMARC", m.DMARC}, {"MTA-STS", m.MTASTS}, {"TLS reporting", m.TLSRPT}} {
			state := "not published"
			if len(row.records) > 0 {
				state = row.records[0]
			}
			fmt.Fprintf(w, "  %-14s %s\n", row.name, state)
		}
	}
	if len(res.DanglingAliases) > 0 {
		fmt.Fprintln(w, "\nAliases to names that do not exist (check whether they are still needed):")
		for _, h := range res.DanglingAliases {
			fmt.Fprintf(w, "  %s\n", h)
		}
	}
	if len(sc.Errors) > 0 {
		fmt.Fprintln(w, "\nNotes:")
		for _, e := range sc.Errors {
			for _, line := range strings.Split(e.Message, "\n") {
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
	if len(files) > 0 {
		fmt.Fprintln(w, "\nSaved:")
		for _, f := range files {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
}

func num(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func seconds(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1f s", d.Seconds())
	}
	return d.Round(time.Second).String()
}
