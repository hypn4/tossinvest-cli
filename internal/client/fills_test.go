package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func fillsFixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func TestListCompactExecutions(t *testing.T) {
	t.Parallel()

	root := fillsFixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/trading/orders/histories/compact/executed" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("productCode"); got != "US20100311002" {
			t.Fatalf("unexpected productCode: %s", got)
		}
		if got := r.URL.Query().Get("timeUnit"); got != "thirty_minute" {
			t.Fatalf("unexpected timeUnit: %s", got)
		}
		http.ServeFile(w, r, filepath.Join(root, "compact-executed-us.json"))
	}))
	defer server.Close()

	sess := session.Session{Cookies: map[string]string{"SESSION": "test"}}
	c := New(Config{HTTPClient: server.Client(), CertBaseURL: server.URL, Session: &sess})
	fills, err := c.ListCompactExecutions(context.Background(), "US20100311002", "thirty_minute")
	if err != nil {
		t.Fatalf("ListCompactExecutions returned error: %v", err)
	}
	if len(fills) != 2 {
		t.Fatalf("expected 2 fills, got %d", len(fills))
	}
	if fills[0].TradeType != "buy" {
		t.Fatalf("unexpected first trade type: %s", fills[0].TradeType)
	}
	if fills[1].Quantity != 35 {
		t.Fatalf("unexpected second quantity: %v", fills[1].Quantity)
	}
}
