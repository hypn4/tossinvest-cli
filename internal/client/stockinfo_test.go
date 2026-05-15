package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestGetStockInfoDetailFromFixture(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-detail-ui-info-snxx.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/search/stocks":
			w.Write([]byte(`{"result":{"stocks":[{"stockCode":"NAS0250224006","stockName":"SNDK","matchType":"EXACT"}]}}`))
		case "/api/v1/stock-detail/ui/NAS0250224006/info":
			w.Write(body)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	detail, err := c.GetStockInfoDetail(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockInfoDetail error: %v", err)
	}
	if detail.ProductCode != "NAS0250224006" {
		t.Fatalf("unexpected product code: %s", detail.ProductCode)
	}
	if len(detail.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(detail.Sections))
	}
	if detail.Sections[0].Type != "OVERVIEW" || detail.Sections[1].Type != "INDICATORS" {
		t.Fatalf("unexpected section order: %+v", detail.Sections)
	}
	var overview struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(detail.Sections[0].Data, &overview); err != nil {
		t.Fatalf("unmarshal OVERVIEW: %v", err)
	}
	if overview.Name != "샌디스크" {
		t.Fatalf("unexpected OVERVIEW.name: %s", overview.Name)
	}
	_ = domain.StockInfoSection{} // ensure import is used if needed
}
