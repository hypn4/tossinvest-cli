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

func TestGetSignalDetail(t *testing.T) {
	t.Parallel()

	root := signalsFixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/dashboard/wts/overview/ai-signals/detail" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("productCode"); got != "US20190226001" {
			t.Fatalf("unexpected productCode: %s", got)
		}
		if got := r.URL.Query().Get("productType"); got != "STOCKS" {
			t.Fatalf("unexpected productType: %s", got)
		}
		http.ServeFile(w, r, filepath.Join(root, "signals-detail.json"))
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	detail, err := c.GetSignalDetail(context.Background(), "US20190226001")
	if err != nil {
		t.Fatalf("GetSignalDetail returned error: %v", err)
	}
	if detail.SignalDirection != 1 {
		t.Fatalf("expected bullish (1), got %d", detail.SignalDirection)
	}
	if detail.Description != "실적 개선 효과" {
		t.Fatalf("unexpected description: %s", detail.Description)
	}
	if len(detail.DescriptionItems) != 3 {
		t.Fatalf("expected 3 description bullets, got %d", len(detail.DescriptionItems))
	}
	if len(detail.News) != 2 {
		t.Fatalf("expected 2 news headlines, got %d", len(detail.News))
	}
	if detail.News[0].AgencyName != "벤징가" {
		t.Fatalf("first agency should be 벤징가, got %s", detail.News[0].AgencyName)
	}
	if len(detail.Keywords) != 3 || detail.Keywords[0] != "실적 개선" {
		t.Fatalf("unexpected keywords: %+v", detail.Keywords)
	}
	if len(detail.Related) != 1 || detail.Related[0].AssetName != "피스클노트 홀딩스" {
		t.Fatalf("unexpected related list: %+v", detail.Related)
	}
}
