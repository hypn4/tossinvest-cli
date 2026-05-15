package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockFinancialsFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Stability       json.RawMessage `json:"stability"`
		Revenue         json.RawMessage `json:"revenue"`
		OperatingIncome json.RawMessage `json:"operatingIncome"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-financials-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/stability/NAS0250224006":
			w.Write(bundle.Stability)
		case "/api/v2/stock-infos/revenue-and-net-profit/NAS0250224006":
			w.Write(bundle.Revenue)
		case "/api/v2/stock-infos/operating-income/NAS0250224006":
			w.Write(bundle.OperatingIncome)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	fin, err := c.GetStockFinancials(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockFinancials error: %v", err)
	}
	if fin.Stability.Position != "LOW" {
		t.Fatalf("expected stability LOW, got %q", fin.Stability.Position)
	}
	if fin.Stability.CurrentRatio != 478.247 {
		t.Fatalf("unexpected currentRatio: %v", fin.Stability.CurrentRatio)
	}
	if fin.Revenue.RecentFiscalYear != 2026 || fin.Revenue.RecentFiscalQuarter != 1 {
		t.Fatalf("unexpected revenue period: %d Q%d", fin.Revenue.RecentFiscalYear, fin.Revenue.RecentFiscalQuarter)
	}
	if len(fin.Revenue.Graph) != 4 {
		t.Fatalf("expected 4 revenue points, got %d", len(fin.Revenue.Graph))
	}
	if fin.Revenue.Graph[3].Period != "2026-03" {
		t.Fatalf("unexpected last period: %q", fin.Revenue.Graph[3].Period)
	}
	if fin.OperatingIncome.Position != "HIGH" {
		t.Fatalf("expected op-income HIGH, got %q", fin.OperatingIncome.Position)
	}
	if len(fin.OperatingIncome.Graph) != 4 {
		t.Fatalf("expected 4 op-income points, got %d", len(fin.OperatingIncome.Graph))
	}
}
