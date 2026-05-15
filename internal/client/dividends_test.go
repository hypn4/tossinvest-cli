package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockDividendsAAPL(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		YieldCard json.RawMessage `json:"yieldCard"`
		Years     json.RawMessage `json:"years"`
		History   json.RawMessage `json:"history"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-dividends-aapl.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-infos/US19801212001/dividends/yield-ratio/histories":
			w.Write(bundle.YieldCard)
		case "/api/v1/stock-infos/dividend/US19801212001/years":
			w.Write(bundle.Years)
		case "/api/v1/stock-infos/dividend/US19801212001/summary":
			w.Write(bundle.History)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	div, err := c.GetStockDividends(context.Background(), "US19801212001")
	if err != nil {
		t.Fatalf("GetStockDividends error: %v", err)
	}
	if div.Summary.Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", div.Summary.Currency)
	}
	if div.Summary.TTMDividendTotalCount != 4 {
		t.Fatalf("expected 4 TTM count, got %d", div.Summary.TTMDividendTotalCount)
	}
	if div.Summary.DividendGrowthRatio == nil || *div.Summary.DividendGrowthRatio < 0.01 {
		t.Fatalf("expected positive growth, got %v", div.Summary.DividendGrowthRatio)
	}
	if len(div.RecentYears.Payouts) != 4 {
		t.Fatalf("expected 4 recent payouts, got %d", len(div.RecentYears.Payouts))
	}
	if div.RecentYears.RangeLabel != "3년" {
		t.Fatalf("expected 3년 range, got %q", div.RecentYears.RangeLabel)
	}
	if div.RecentYears.TotalCash != 1.04 {
		t.Fatalf("expected total 1.04, got %v", div.RecentYears.TotalCash)
	}
	if len(div.FullHistory) != 4 {
		t.Fatalf("expected 4 history rows, got %d", len(div.FullHistory))
	}
	if div.RecentYears.Payouts[3].ExDate != "2026-05-09" {
		t.Fatalf("unexpected last exDate: %q", div.RecentYears.Payouts[3].ExDate)
	}
}

func TestGetStockDividendsSNDKEmpty(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		YieldCard json.RawMessage `json:"yieldCard"`
		Years     json.RawMessage `json:"years"`
		History   json.RawMessage `json:"history"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-dividends-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-infos/NAS0250224006/dividends/yield-ratio/histories":
			w.Write(bundle.YieldCard)
		case "/api/v1/stock-infos/dividend/NAS0250224006/years":
			w.Write(bundle.Years)
		case "/api/v1/stock-infos/dividend/NAS0250224006/summary":
			w.Write(bundle.History)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	div, err := c.GetStockDividends(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockDividends error: %v", err)
	}
	if div.Summary.DividendCount != 0 {
		t.Fatalf("expected 0 dividend count, got %d", div.Summary.DividendCount)
	}
	if len(div.RecentYears.Payouts) != 0 {
		t.Fatalf("expected 0 recent payouts, got %d", len(div.RecentYears.Payouts))
	}
	if len(div.FullHistory) != 0 {
		t.Fatalf("expected 0 history rows, got %d", len(div.FullHistory))
	}
	if div.Summary.Currency != "" {
		t.Fatalf("expected empty currency, got %q", div.Summary.Currency)
	}
}
