package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestViewLeavesForeignFilesAlone: pointing the viewer at a folder that
// holds someone else's .db file must fail without deleting that file.
func TestViewLeavesForeignFilesAlone(t *testing.T) {
	dir := t.TempDir()
	foreign := filepath.Join(dir, "app.db")
	if err := os.WriteFile(foreign, []byte("not a scan"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := serveView(context.Background(), dir, "127.0.0.1:0", false, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "no readable result") {
		t.Errorf("serveView = %v", err)
	}
	if b, err := os.ReadFile(foreign); err != nil || string(b) != "not a scan" {
		t.Fatalf("the foreign file was changed or removed: %q, %v", b, err)
	}
}

func TestViewServesThisComputerOnly(t *testing.T) {
	for addr, ok := range map[string]bool{
		"127.0.0.1:0": true, "localhost:8080": true, "[::1]:9000": true, "127.0.0.2:0": true,
		"0.0.0.0:8080": false, ":8080": false, "192.168.1.5:80": false, "[::]:80": false, "example.com:80": false,
	} {
		if err := loopbackOnly(addr); (err == nil) != ok {
			t.Errorf("loopbackOnly(%q) = %v, want allowed=%v", addr, err, ok)
		}
	}
}

func TestTerminalSafe(t *testing.T) {
	// A TXT record that would retitle the terminal and plant a link.
	in := "v=spf1 \x1b]0;pwned\x07 \x1b]8;;https://evil\x1b\\click\x1b]8;;\x1b\\ \u009b31m ok\n\tnext"
	got := terminalSafe(in)
	for _, bad := range []string{"\x1b", "\x07", "\u009b"} {
		if strings.Contains(got, bad) {
			t.Errorf("terminalSafe left %q in %q", bad, got)
		}
	}
	if !strings.Contains(got, `\x1b]0;pwned\x07`) || !strings.HasSuffix(got, "ok\n\tnext") {
		t.Errorf("terminalSafe = %q", got)
	}
}

func TestCSVRowGuardsEveryCell(t *testing.T) {
	got := csvRow("https://example.com/", "=1+1", "-2", "plain")
	if got[0] != "https://example.com/" || got[1] != "'=1+1" || got[2] != "'-2" || got[3] != "plain" {
		t.Errorf("csvRow = %q", got)
	}
}
