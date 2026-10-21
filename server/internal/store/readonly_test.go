package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"websitemapper/internal/scan"
	"websitemapper/internal/store"
)

// TestReadOnlyNeverTouchesFiles: opening a folder only to read it must
// leave every file as it was, including databases that are not scans.
func TestReadOnlyNeverTouchesFiles(t *testing.T) {
	dir := t.TempDir()
	foreign := filepath.Join(dir, "app.db")
	if err := os.WriteFile(foreign, []byte("someone else's database"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(store.Options{Dir: dir, ReadOnly: true, Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenResult(context.Background(), "app"); !errors.Is(err, scan.ErrNotFound) {
		t.Errorf("OpenResult of a foreign file = %v", err)
	}
	if err := s.Create(context.Background(), scan.Scan{ID: "new"}); err == nil {
		t.Error("a read-only store must not create scans")
	}
	s.Close()
	if b, err := os.ReadFile(foreign); err != nil || string(b) != "someone else's database" {
		t.Fatalf("the foreign file was changed or removed: %q, %v", b, err)
	}
	// A folder that does not exist is not created.
	missing := filepath.Join(dir, "nope")
	if _, err := store.Open(store.Options{Dir: missing, ReadOnly: true, Log: quiet}); err == nil {
		t.Error("opening a missing folder read-only should fail")
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Error("a read-only open created the folder")
	}
}

// A finished scan is still readable in read-only mode, and an interrupted
// one is shown as such without its file being rewritten.
func TestReadOnlyReadsResults(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Options{Dir: dir, Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	record(t, s, "done", nil)
	if err := s.Create(ctx, scan.Scan{ID: "running", Status: scan.StatusRunning}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	before, _ := os.Stat(filepath.Join(dir, "running.db"))

	ro, err := store.Open(store.Options{Dir: dir, ReadOnly: true, Log: quiet})
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if r, err := ro.OpenResult(ctx, "done"); err != nil {
		t.Errorf("finished result: %v", err)
	} else {
		r.Close()
	}
	if sc, err := ro.Get(ctx, "running"); err != nil || sc.Status != scan.StatusFailed {
		t.Errorf("interrupted scan = %+v, %v", sc, err)
	}
	after, _ := os.Stat(filepath.Join(dir, "running.db"))
	if !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Error("the interrupted scan's file was rewritten")
	}
}
