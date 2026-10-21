package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"websitemapper/internal/api"
	"websitemapper/internal/scan"
	"websitemapper/internal/store"
)

// viewerPage is the page that shows a result in the browser. It reads the
// result through the same API the website uses.
//
//go:embed viewer.html
var viewerPage string

// scanDir is the folder inside a result folder that keeps the scan's
// database, which the viewer reads.
const scanDir = "scan"

const viewUsage = `webscanner view shows a saved result in your browser.

Usage:
  webscanner view [flags] <result folder>

The page is served from this computer only (127.0.0.1) and stops when you
press Ctrl-C.

Flags:
`

func runView(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("webscanner view", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, viewUsage)
		fs.PrintDefaults()
	}
	addr := fs.String("addr", "127.0.0.1:0", "address to serve on (default a free port on this computer)")
	noOpen := fs.Bool("no-open", false, "print the address instead of opening the browser")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if fs.NArg() != 1 {
		fs.Usage()
		fmt.Fprintln(stderr, "webscanner: give the result folder to show")
		return exitUsage
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := serveView(ctx, fs.Arg(0), *addr, !*noOpen, stderr); err != nil {
		fmt.Fprintln(stderr, "webscanner:", err)
		return exitError
	}
	return exitOK
}

// serveView serves the result in folder until ctx ends.
func serveView(ctx context.Context, folder, addr string, open bool, stderr io.Writer) error {
	if err := loopbackOnly(addr); err != nil {
		return err
	}
	dir, id, err := findScan(folder)
	if err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Read-only: the folder may be any folder the user named, and nothing in
	// it may be changed or removed (a stray .db file is not ours to delete).
	repo, err := store.Open(store.Options{Dir: dir, ReadOnly: true, Log: log})
	if err != nil {
		return err
	}
	defer repo.Close()
	rd, err := repo.OpenResult(ctx, id)
	if err != nil {
		return fmt.Errorf("%s has no readable result: %w", folder, err)
	}
	domain := rd.Summary().Domain.Canonical
	rd.Close()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/"
	srv := &http.Server{
		Handler:           viewHandler(viewScans{repo}, id, ln.Addr().String(), log),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()

	fmt.Fprintf(stderr, "Showing %s at %s\nPress Ctrl-C to stop.\n", domain, url)
	if open {
		if err := openBrowser(url); err != nil {
			fmt.Fprintln(stderr, "Could not open the browser; open the address above yourself.")
		}
	}
	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shut)
}

// findScan finds the scan database in a result folder: the folder's "scan"
// subfolder, or the folder itself. It must hold exactly one scan.
func findScan(folder string) (dir, id string, err error) {
	for _, d := range []string{filepath.Join(folder, scanDir), folder} {
		matches, _ := filepath.Glob(filepath.Join(d, "*.db"))
		switch len(matches) {
		case 0:
			continue
		case 1:
			return d, strings.TrimSuffix(filepath.Base(matches[0]), ".db"), nil
		default:
			return "", "", fmt.Errorf("%s holds more than one scan", d)
		}
	}
	if _, err := os.Stat(folder); err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("%s has no saved scan to show (results written before version 1.1, or with -no-files, cannot be shown)", folder)
}

// loopbackOnly refuses addresses other computers could reach: the viewer
// has no password, and its results are for this computer only.
func loopbackOnly(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("-addr %q: %w", addr, err)
	}
	if host == "localhost" {
		return nil
	}
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("-addr %q: the viewer serves this computer only; use 127.0.0.1 or localhost", addr)
}

// viewHandler serves the viewer page at / and the result API under /api/.
// It answers only GET requests addressed to host, so that no other web page
// can reach it through a name that points at this computer.
func viewHandler(scans api.Scans, id, host string, log *slog.Logger) http.Handler {
	page := strings.ReplaceAll(viewerPage, "{{SCAN_ID}}", id)
	apiHandler := api.NewHandler(scans, log, api.Access{})
	_, port, _ := net.SplitHostPort(host)
	allowed := map[string]bool{host: true, "localhost:" + port: true, "127.0.0.1:" + port: true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case !allowed[r.Host]:
			http.Error(w, "unknown host", http.StatusForbidden)
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.Error(w, "this viewer is read-only", http.StatusMethodNotAllowed)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			apiHandler.ServeHTTP(w, r)
		case r.URL.Path == "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'")
			_, _ = io.WriteString(w, page)
		default:
			http.NotFound(w, r)
		}
	})
}

// viewScans gives the API read-only access to a saved scan.
type viewScans struct{ repo *store.Store }

var errReadOnly = errors.New("this viewer is read-only")

func (v viewScans) Create(context.Context, scan.CreateRequest) (scan.Scan, error) {
	return scan.Scan{}, errReadOnly
}
func (v viewScans) Get(ctx context.Context, id string) (scan.Scan, error) { return v.repo.Get(ctx, id) }
func (v viewScans) OpenResult(ctx context.Context, id string) (scan.ResultReader, error) {
	return v.repo.OpenResult(ctx, id)
}
func (v viewScans) Cancel(context.Context, string, string) (scan.Scan, error) {
	return scan.Scan{}, errReadOnly
}
func (v viewScans) Stats() scan.Stats { return scan.Stats{} }

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// keepScan moves the finished scan's database from the work folder into the
// result folder, so that "webscanner view" can show it later. The store
// must be closed.
func keepScan(work, id, out string) error {
	dst := filepath.Join(out, scanDir)
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}
	files, _ := filepath.Glob(filepath.Join(work, id+".db*")) // the database and any SQLite side files
	for _, src := range files {
		if err := moveFile(src, filepath.Join(dst, filepath.Base(src))); err != nil {
			return err
		}
	}
	return nil
}

// moveFile renames src to dst, copying when they are on different drives.
func moveFile(src, dst string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
