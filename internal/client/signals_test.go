package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func signalsFixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func TestListSignals(t *testing.T) {
	t.Parallel()

	root := signalsFixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/dashboard/wts/overview/ai-signals" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"productCodes"`) {
			t.Fatalf("body missing productCodes: %s", body)
		}
		http.ServeFile(w, r, filepath.Join(root, "signals-batch.json"))
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	signals, err := c.ListSignals(context.Background(), []string{
		"US20190226001", "US20100311002", "US20100311003", "NAS0230126004", "US20040819002",
	})
	if err != nil {
		t.Fatalf("ListSignals returned error: %v", err)
	}
	if len(signals) != 5 {
		t.Fatalf("expected 5 signals, got %d", len(signals))
	}
	byCode := map[string]string{}
	for _, s := range signals {
		byCode[s.ProductCode] = s.ReasoningDescription
	}
	if byCode["US20100311002"] != "차익실현 매도" {
		t.Fatalf("unexpected reasoning for SOXL: %q", byCode["US20100311002"])
	}
	// Verify JSON encoding stays user-friendly
	if _, err := json.Marshal(signals); err != nil {
		t.Fatalf("signals do not marshal: %v", err)
	}
}
