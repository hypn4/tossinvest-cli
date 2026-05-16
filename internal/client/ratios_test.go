package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockRatiosFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statements/comprehensive" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ra, err := c.GetStockRatios(context.Background(), "NAS0250224006", "DEBT_RATIO", "Q")
	if err != nil {
		t.Fatalf("GetStockRatios error: %v", err)
	}

	// Verify the selector body was sent
	var sent struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body not JSON: %v (body=%q)", err, string(gotBody))
	}
	if sent.FactorCode != "DEBT_RATIO" || sent.Period != "Q" {
		t.Fatalf("unexpected request body: %+v", sent)
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

func TestGetStockRatiosLowercaseInputsNormalize(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "current_ratio", "q")
	if err != nil {
		t.Fatalf("expected lowercase inputs to be normalized; got error: %v", err)
	}
	var sent struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("body parse: %v", err)
	}
	if sent.FactorCode != "CURRENT_RATIO" || sent.Period != "Q" {
		t.Fatalf("expected uppercase in body; got %+v", sent)
	}
}

func TestGetStockRatiosRejectsBadFactor(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "BAD_FACTOR", "Q")
	if err == nil {
		t.Fatalf("expected error for bad factor")
	}
}

func TestGetStockRatiosRejectsBadPeriod(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "DEBT_RATIO", "X")
	if err == nil {
		t.Fatalf("expected error for bad period")
	}
}
