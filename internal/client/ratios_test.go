package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockRatiosFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statements/comprehensive" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ra, err := c.GetStockRatios(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockRatios error: %v", err)
	}
	if ra.Factor.Code != "DEBT_RATIO" || ra.Factor.DisplayName != "부채비율" {
		t.Fatalf("unexpected factor: %+v", ra.Factor)
	}
	if ra.Period != "Q" {
		t.Fatalf("unexpected period: %q", ra.Period)
	}
	if ra.RangeLabel != "3년" {
		t.Fatalf("unexpected range: %q", ra.RangeLabel)
	}
	if len(ra.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(ra.Items))
	}
	if ra.Items[0].Code != "TOTAL_SHAREHOLDERS_EQUITY" {
		t.Fatalf("unexpected first item: %+v", ra.Items[0])
	}
	if ra.Items[2].Code != "DEBT_RATIO" || ra.Items[2].Unit != "PERCENT" {
		t.Fatalf("unexpected last item: %+v", ra.Items[2])
	}
	if len(ra.Items[0].Values) != 4 {
		t.Fatalf("expected 4 values per item, got %d", len(ra.Items[0].Values))
	}
	if ra.Items[2].Values[2].Value != 34.35 {
		t.Fatalf("unexpected debt ratio Q4-2025: %v", ra.Items[2].Values[2].Value)
	}
}
