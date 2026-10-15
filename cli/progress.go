package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"websitemapper/internal/scan"
)

var phaseText = map[scan.Phase]string{
	scan.PhaseQueued:     "Starting",
	scan.PhaseSubdomains: "Discovering subdomains",
	scan.PhaseArchive:    "Searching web archives",
	scan.PhaseSitemaps:   "Reading robots.txt and sitemaps",
	scan.PhaseResolving:  "Resolving hosts",
	scan.PhaseProbing:    "Checking which hosts are live",
	scan.PhaseCrawling:   "Crawling",
	scan.PhaseFinalizing: "Saving",
}

// progress shows a scan's progress: one line rewritten in place on a
// terminal, and a line per change otherwise (for logs and pipes).
type progress struct {
	w        io.Writer
	off      bool
	terminal bool
	last     string
	width    int
}

func newProgress(w io.Writer, quiet bool) *progress {
	p := &progress{w: w, off: quiet}
	if f, ok := w.(*os.File); ok {
		if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
			p.terminal = true
		}
	}
	return p
}

func (p *progress) show(sc scan.Scan) {
	if p.off {
		return
	}
	phase := phaseText[sc.Phase]
	if phase == "" {
		phase = string(sc.Phase)
	}
	if pr := sc.Progress; pr != nil && pr.Total > 0 {
		phase += fmt.Sprintf(" %s/%s", num(pr.Completed), num(pr.Total))
	}
	line := fmt.Sprintf("%s... %s hosts, %s URLs", phase, num(sc.Counts.Hosts), num(sc.Counts.URLs))
	if line == p.last {
		return
	}
	p.last = line
	if !p.terminal {
		fmt.Fprintln(p.w, line)
		return
	}
	pad := ""
	if n := p.width - len(line); n > 0 {
		pad = strings.Repeat(" ", n)
	}
	p.width = len(line)
	fmt.Fprint(p.w, "\r"+line+pad)
}

// note prints a message on its own line.
func (p *progress) note(msg string) {
	if p.off {
		return
	}
	if p.terminal && p.width > 0 {
		fmt.Fprintln(p.w)
		p.width = 0
	}
	fmt.Fprintln(p.w, msg)
}

// done ends the progress line.
func (p *progress) done() {
	if !p.off && p.terminal && p.width > 0 {
		fmt.Fprintln(p.w)
		p.width = 0
	}
}
