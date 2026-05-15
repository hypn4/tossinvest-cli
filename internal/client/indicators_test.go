package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockIndicatorsFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-indicators-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stock-detail/ui/wts/NAS0250224006/investment-indicators" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ind, err := c.GetStockIndicators(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockIndicators error: %v", err)
	}
	val, ok := ind.Sections["가치평가"]
	if !ok {
		t.Fatalf("missing 가치평가 section")
	}
	if val["displayPer"] != "45.4배" {
		t.Fatalf("unexpected PER: %v", val["displayPer"])
	}
	earnings := ind.Sections["수익"]
	if earnings["roe"] != "39.3%" {
		t.Fatalf("unexpected ROE: %v", earnings["roe"])
	}
}
