package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockEstimatesFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Date            json.RawMessage `json:"date"`
		Revenue         json.RawMessage `json:"revenue"`
		EPS             json.RawMessage `json:"eps"`
		OperatingIncome json.RawMessage `json:"operatingIncome"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-estimates-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/companies/NAS0250224006/financial/estimate/date":
			w.Write(bundle.Date)
		case "/api/v2/companies/NAS0250224006/financial/estimate/revenue":
			w.Write(bundle.Revenue)
		case "/api/v2/companies/NAS0250224006/financial/estimate/eps":
			w.Write(bundle.EPS)
		case "/api/v2/companies/NAS0250224006/financial/estimate/operating-income":
			w.Write(bundle.OperatingIncome)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	est, err := c.GetStockEstimates(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockEstimates error: %v", err)
	}
	if est.Headline.RevenueEst == nil || *est.Headline.RevenueEst != 7736730000.0 {
		t.Fatalf("unexpected headline revenueEst: %v", est.Headline.RevenueEst)
	}
	if est.Headline.OperatingIncomeEst != nil {
		t.Fatalf("expected nil OperatingIncomeEst in headline; got %v", *est.Headline.OperatingIncomeEst)
	}
	if len(est.Revenue.Graph) != 4 {
		t.Fatalf("expected 4 revenue points, got %d", len(est.Revenue.Graph))
	}
	if est.Revenue.Position == nil || *est.Revenue.Position != "HIGH" {
		t.Fatalf("expected revenue position HIGH, got %v", est.Revenue.Position)
	}
	if len(est.EPS.Graph) != 4 {
		t.Fatalf("expected 4 EPS points, got %d", len(est.EPS.Graph))
	}
	if est.EPS.Graph[3].Surprise == nil || *est.EPS.Graph[3].Surprise < 100 {
		t.Fatalf("expected large positive surprise on last EPS point; got %v", est.EPS.Graph[3].Surprise)
	}
	// operating-income has null position (no analyst coverage for that metric)
	if est.OperatingIncome.Position != nil {
		t.Fatalf("expected nil OperatingIncome.Position; got %q", *est.OperatingIncome.Position)
	}
	if len(est.OperatingIncome.Graph) != 4 {
		t.Fatalf("expected 4 OI points, got %d", len(est.OperatingIncome.Graph))
	}
	// All OI points have null est
	for i, p := range est.OperatingIncome.Graph {
		if p.OperatingIncomeEst != nil {
			t.Fatalf("OI[%d] expected nil est, got %v", i, *p.OperatingIncomeEst)
		}
	}
}
