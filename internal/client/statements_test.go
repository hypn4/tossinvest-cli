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

func TestGetStockStatementsIncQ(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-statements-sndk-inc-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statement-records" {
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
	st, err := c.GetStockStatements(context.Background(), "NAS0250224006", "INC", "Q")
	if err != nil {
		t.Fatalf("GetStockStatements error: %v", err)
	}

	// Verify the request body was the JSON selector
	var sentBody struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sentBody); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	if sentBody.FactorCode != "INC" || sentBody.Period != "Q" {
		t.Fatalf("unexpected request body: %+v", sentBody)
	}

	if st.Factor.Code != "INC" || st.Factor.DisplayName != "손익계산서" {
		t.Fatalf("unexpected factor: %+v", st.Factor)
	}
	if st.Period != "Q" {
		t.Fatalf("unexpected period: %q", st.Period)
	}
	if len(st.Periods) != 4 {
		t.Fatalf("expected 4 periods, got %d", len(st.Periods))
	}
	if st.Periods[0].Period != "2025-06" || st.Periods[3].Period != "2026-03" {
		t.Fatalf("unexpected period order: %s ... %s", st.Periods[0].Period, st.Periods[3].Period)
	}
	if len(st.Periods[0].Items) != 6 {
		t.Fatalf("expected 6 items in first period, got %d", len(st.Periods[0].Items))
	}
	// Verify parent-child link survives
	srev := st.Periods[3].Items[1]
	if srev.Item != "SREV" || srev.ParentItem != "RTLR" {
		t.Fatalf("unexpected SREV item: %+v", srev)
	}
	// Verify revenue grows
	rtlr := st.Periods[3].Items[0]
	if rtlr.Value == nil || *rtlr.Value != 3092.0 {
		t.Fatalf("unexpected RTLR Q1 2026 value: %v", rtlr.Value)
	}
}

func TestGetStockStatementsRejectsBadFactor(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockStatements(context.Background(), "NAS0250224006", "XYZ", "Q")
	if err == nil {
		t.Fatalf("expected error for bad factor")
	}
}

func TestGetStockStatementsRejectsBadPeriod(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockStatements(context.Background(), "NAS0250224006", "INC", "X")
	if err == nil {
		t.Fatalf("expected error for bad period")
	}
}
