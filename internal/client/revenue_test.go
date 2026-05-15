package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetSalesCompositionFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	sales := mustReadFile(t, filepath.Join(root, "sales-compositions-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v1/companies/NAS116LTR-E0/sales-compositions":
			w.Write(sales)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	sc, err := c.GetSalesComposition(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetSalesComposition error: %v", err)
	}
	if sc.CompanyCode != "NAS116LTR-E0" {
		t.Fatalf("expected companyCode NAS116LTR-E0, got %q", sc.CompanyCode)
	}
	if sc.FiscalYear != 2025 {
		t.Fatalf("expected FY2025, got %d", sc.FiscalYear)
	}
	if len(sc.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sc.Items))
	}
	if sc.Items[0].Business != "클라이언트" || sc.Items[0].Ratio != 56.11 {
		t.Fatalf("unexpected first item: %+v", sc.Items[0])
	}
}
