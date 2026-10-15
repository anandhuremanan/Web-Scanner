package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"websitemapper/internal/scan"
	"websitemapper/internal/store"
)

// savedScan writes a finished scan "s1" for example.com into dir.
func savedScan(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(store.Options{Dir: dir, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Create(ctx, scan.Scan{ID: "s1", Status: scan.StatusRunning}); err != nil {
		t.Fatal(err)
	}
	urls, err := s.OpenURLs(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	res := scan.Result{ScanID: "s1", Status: scan.StatusCompleted, Domain: scan.Domain{Canonical: "example.com"}}
	if err := s.SaveResult(ctx, res); err != nil {
		t.Fatal(err)
	}
	urls.Close()
	if err := s.Update(ctx, scan.Scan{ID: "s1", Status: scan.StatusCompleted, Phase: scan.PhaseDone}); err != nil {
		t.Fatal(err)
	}
}

func TestFindScan(t *testing.T) {
	out := t.TempDir()
	if _, _, err := findScan(out); err == nil {
		t.Error("a folder without a scan must be refused")
	}
	savedScan(t, filepath.Join(out, scanDir))
	dir, id, err := findScan(out)
	if err != nil || id != "s1" || dir != filepath.Join(out, scanDir) {
		t.Errorf("findScan = %q, %q, %v", dir, id, err)
	}
	// The scan folder itself works too.
	if _, id, err := findScan(filepath.Join(out, scanDir)); err != nil || id != "s1" {
		t.Errorf("findScan(scan folder) = %q, %v", id, err)
	}
}

func TestViewHandler(t *testing.T) {
	dir := t.TempDir()
	savedScan(t, dir)
	repo, err := store.Open(store.Options{Dir: dir, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	h := viewHandler(viewScans{repo}, "s1", "127.0.0.1:9999", slog.New(slog.NewTextHandler(io.Discard, nil)))

	do := func(method, host, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Host = host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := do("GET", "127.0.0.1:9999", "/"); w.Code != 200 || !strings.Contains(w.Body.String(), `const SCAN = "s1"`) {
		t.Errorf("page: %d", w.Code)
	}
	if w := do("GET", "localhost:9999", "/api/scans/s1/summary"); w.Code != 200 || !strings.Contains(w.Body.String(), "example.com") {
		t.Errorf("summary: %d %s", w.Code, w.Body)
	}
	if w := do("GET", "attacker.example:9999", "/api/scans/s1/summary"); w.Code != http.StatusForbidden {
		t.Errorf("another host name must be refused, got %d", w.Code)
	}
	if w := do("POST", "127.0.0.1:9999", "/api/scans"); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("starting a scan must be refused, got %d", w.Code)
	}
}

func TestKeepScan(t *testing.T) {
	work, out := t.TempDir(), t.TempDir()
	savedScan(t, work)
	if err := keepScan(work, "s1", out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, scanDir, "s1.db")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(work, "s1.db")); !os.IsNotExist(err) {
		t.Error("the database should have been moved")
	}
}
