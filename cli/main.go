// Command webscanner lists what is publicly discoverable about a domain, its
// subdomains and the pages on them, from your own machine.
//
// It runs the same discovery engines as the Web Scanner website, without the
// website's limits on how much one scan may do. The limits that protect the
// scanned sites and the public sources stay: at most 5 requests per second to
// any one host, one web archive request every 3 seconds, and no guessing,
// form submission or port scanning.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"websitemapper/internal/config"
	"websitemapper/internal/discovery/subdomains"
	"websitemapper/internal/resource"
	"websitemapper/internal/results"
	"websitemapper/internal/scan"
	"websitemapper/internal/store"
)

// version is set when a release is built (-ldflags "-X main.version=...").
var version = "dev"

// Exit codes.
const (
	exitOK          = 0
	exitError       = 1
	exitUsage       = 2
	exitInterrupted = 130
)

type options struct {
	target   string
	mode     scan.Mode
	out      string
	noFiles  bool
	print    string
	pages    int
	depth    int
	timeout  time.Duration
	quiet    bool
	verbose  bool
	noMemory bool
}

const usage = `webscanner lists what is publicly discoverable about a domain: its
subdomains and the pages on them, and where each one was found.

Usage:
  webscanner [flags] <domain or URL>

Examples:
  webscanner example.com
  webscanner -mode passive example.com
  webscanner -mode full -pages 1000 https://www.example.com/
  webscanner -print urls -no-files example.com > urls.txt

Modes:
  passive    public sources only; never contacts the site
  standard   also checks which hosts are live and reads their robots.txt
             and sitemaps (default)
  full       also follows links on each live host

Only scan domains you own or have permission to inspect.

Flags:
`

var modeNames = map[string]scan.Mode{
	"passive": scan.ModePassive, "standard": scan.ModeLight, "light": scan.ModeLight, "full": scan.ModeFull,
}

func parseArgs(args []string, stderr io.Writer) (options, error) {
	fs := flag.NewFlagSet("webscanner", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	var o options
	mode := fs.String("mode", "standard", "how deep to scan: passive, standard or full")
	fs.StringVar(&o.out, "out", "", "folder for the result files (default webscanner-<domain>-<time>)")
	fs.BoolVar(&o.noFiles, "no-files", false, "do not write result files")
	fs.StringVar(&o.print, "print", "", "also print to standard output: hosts or urls, one per line")
	fs.IntVar(&o.pages, "pages", 500, "full mode: most pages requested per host")
	fs.IntVar(&o.depth, "depth", 3, "full mode: how many links deep to follow from each host's start page")
	fs.DurationVar(&o.timeout, "timeout", 0, "stop the scan after this long, keeping what it found (e.g. 30m; default no limit)")
	fs.BoolVar(&o.quiet, "quiet", false, "show no progress or summary")
	fs.BoolVar(&o.verbose, "verbose", false, "log what each step does")
	fs.BoolVar(&o.noMemory, "no-memory", false, "do not remember subdomain sources' answers between runs")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if *showVersion {
		fmt.Fprintln(stderr, "webscanner", version)
		return o, flag.ErrHelp
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return o, errors.New("give exactly one domain or URL to scan")
	}
	o.target = fs.Arg(0)
	m, ok := modeNames[strings.ToLower(*mode)]
	if !ok {
		return o, fmt.Errorf("unknown mode %q: use passive, standard or full", *mode)
	}
	o.mode = m
	if o.print != "" && o.print != "hosts" && o.print != "urls" {
		return o, fmt.Errorf("unknown -print %q: use hosts or urls", o.print)
	}
	if o.pages < 1 || o.depth < 0 {
		return o, errors.New("-pages must be at least 1 and -depth at least 0")
	}
	return o, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	o, err := parseArgs(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	if err != nil {
		fmt.Fprintln(stderr, "webscanner:", err)
		return exitUsage
	}

	// Ctrl-C once: stop the scan and keep what it found. Twice: quit now.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, err := scanAndSave(ctx, o, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "webscanner:", err)
	}
	return code
}

func scanAndSave(ctx context.Context, o options, stdout, stderr io.Writer) (int, error) {
	logOut := io.Discard
	if o.verbose {
		logOut = stderr
	}
	log := slog.New(slog.NewTextHandler(logOut, nil))

	// The settings the server would use with nothing configured, then the
	// changes for a scan on the user's own machine.
	cfg, err := config.LoadFrom(func(string) string { return "" })
	if err != nil {
		return exitError, err
	}
	cfg.Scan.MaxURLs, cfg.Scan.MaxDepth = o.pages, o.depth

	// The scan is stored in a temporary folder while it runs, and the
	// result files are written from it at the end.
	work, err := os.MkdirTemp("", "webscanner-")
	if err != nil {
		return exitError, err
	}
	defer os.RemoveAll(work)
	repo, err := store.Open(store.Options{Dir: work, Log: log})
	if err != nil {
		return exitError, err
	}
	defer repo.Close()

	// Subdomain sources' answers are kept between runs: crt.sh in particular
	// is slow, and its answer then comes at once on the next scan.
	var names subdomains.Memory
	if !o.noMemory {
		if dir, err := memoryDir(); err == nil {
			if n, err := store.OpenNames(dir, 30*24*time.Hour, log); err == nil {
				defer n.Close()
				names = n
			}
		}
	}

	p := newPools(cfg)
	timeout := o.timeout
	if timeout <= 0 {
		timeout = 7 * 24 * time.Hour // effectively none
	}
	svc := scan.NewService(repo, pipeline(cfg.Scan, p, names, log), scan.Options{
		MaxRunning:       1,
		QueueSize:        1,
		ScanTimeout:      timeout,
		ProgressInterval: 500 * time.Millisecond,
		Limits:           results.Limits{}, // no ceilings on hosts or URLs
		Pools:            []*resource.Pool{p.http, p.dns, p.ct, p.lookups, p.archive},
	}, log)
	svc.Start(context.Background())
	defer svc.Shutdown(context.Background())

	sc, err := svc.Create(context.Background(), scan.CreateRequest{Target: o.target, Mode: o.mode, Fresh: true})
	if err != nil {
		return exitUsage, err
	}
	if !o.quiet {
		fmt.Fprintf(stderr, "Scanning %s (%s mode). Ctrl-C stops the scan and keeps what it found.\n",
			sc.Domain, modeLabel(o.mode))
	}

	final, interrupted := watch(ctx, svc, sc.ID, newProgress(stderr, o.quiet))

	rd, err := svc.OpenResult(context.Background(), sc.ID)
	if err != nil {
		return exitError, fmt.Errorf("the scan ended without a result: %s", final.Error)
	}
	defer rd.Close()

	var files []string
	if !o.noFiles {
		dir := o.out
		if dir == "" {
			dir = defaultOut(sc.Domain, time.Now())
		}
		if files, err = writeFiles(context.Background(), dir, rd); err != nil {
			return exitError, err
		}
	}
	if o.print != "" {
		if err := printList(context.Background(), stdout, rd, o.print); err != nil {
			return exitError, err
		}
	}
	if !o.quiet {
		printSummary(stderr, final, rd.Summary(), files)
	}
	switch {
	case interrupted:
		return exitInterrupted, nil
	case final.Status == scan.StatusFailed:
		return exitError, nil
	}
	return exitOK, nil
}

// watch shows progress until the scan finishes. If ctx ends first (Ctrl-C),
// it stops the scan, which keeps what it found, and waits for it to save.
func watch(ctx context.Context, svc *scan.Service, id string, pr *progress) (scan.Scan, bool) {
	interrupted := false
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		sc, err := svc.Get(context.Background(), id)
		if err == nil {
			if sc.Status.Finished() {
				pr.done()
				return sc, interrupted
			}
			pr.show(sc)
		}
		select {
		case <-ctx.Done():
			if !interrupted {
				interrupted = true
				pr.note("Stopping; saving what was found so far (Ctrl-C again to quit at once)...")
				_, _ = svc.Cancel(context.Background(), id, "")
				// A second Ctrl-C ends the program.
				hard := make(chan os.Signal, 1)
				signal.Notify(hard, os.Interrupt)
				go func() { <-hard; os.Exit(exitInterrupted) }()
			}
			ctx = context.Background()
		case <-tick.C:
		}
	}
}

func modeLabel(m scan.Mode) string {
	if m == scan.ModeLight {
		return "standard"
	}
	return string(m)
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9.-]+`)

// defaultOut is the result folder's name: webscanner-<domain>-<time>.
func defaultOut(domain string, now time.Time) string {
	return "webscanner-" + unsafeName.ReplaceAllString(domain, "_") + "-" + now.Format("20060102-150405")
}

// memoryDir is where subdomain sources' answers are kept between runs.
func memoryDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "webscanner")
	return dir, os.MkdirAll(dir, 0o750)
}
