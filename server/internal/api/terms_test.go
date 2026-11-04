package api_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	"websitemapper/internal/api"
)

// logBuffer collects log output written by the server's goroutines.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestScansNeedAcceptedTerms(t *testing.T) {
	var logs logBuffer
	v := &fakeVerifier{}
	srv := serverWithLog(t, api.Access{Verifier: v, RequireTerms: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	h := map[string]string{"X-Forwarded-For": "203.0.113.9"}

	// Not confirmed: refused before the visitor check, so the visitor's
	// one-time token is not used up.
	for _, body := range []string{
		`{"target":"example.com","verificationToken":"human"}`,
		`{"target":"example.com","verificationToken":"human","acceptedTerms":false}`,
	} {
		code, out, _ := send(t, "POST", srv.URL+"/api/scans", body, h)
		if code != http.StatusBadRequest || !strings.Contains(out["error"].(string), "Terms of Use") {
			t.Errorf("%s = %d %v", body, code, out)
		}
	}
	if len(v.visitors) != 0 {
		t.Errorf("the visitor check ran for a refused request: %v", v.visitors)
	}

	// Confirmed: the scan starts, and the confirmation is logged with the
	// scan, but not the visitor's address.
	code, out, _ := send(t, "POST", srv.URL+"/api/scans", `{"target":"example.com","verificationToken":"human","acceptedTerms":true}`, h)
	if code != http.StatusAccepted {
		t.Fatalf("confirmed = %d %v", code, out)
	}
	got := logs.String()
	for _, want := range []string{"event=scan_requested", "scan_id=" + out["id"].(string), "terms_accepted=true", "outcome=started", "example.com"} {
		if !strings.Contains(got, want) {
			t.Errorf("log has no %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "203.0.113.9") {
		t.Errorf("the log records the visitor's address:\n%s", got)
	}
}

// Without RequireTerms (the default) a request need not say anything, and
// the log records that it did not.
func TestTermsAreOptionalByDefault(t *testing.T) {
	var logs logBuffer
	srv := serverWithLog(t, api.Access{}, slog.New(slog.NewTextHandler(&logs, nil)))
	if code, out, _ := send(t, "POST", srv.URL+"/api/scans", `{"target":"example.com"}`, nil); code != http.StatusAccepted {
		t.Fatalf("create = %d %v", code, out)
	}
	if !strings.Contains(logs.String(), "terms_accepted=false") {
		t.Errorf("log:\n%s", logs.String())
	}
}
