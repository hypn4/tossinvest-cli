package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetCompanyOverviewFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/stock-infos/NAS0250224006/overview" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ov, err := c.GetCompanyOverview(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetCompanyOverview error: %v", err)
	}
	if ov.Company.CEO != "David V. Goeckeler" {
		t.Fatalf("unexpected CEO: %q", ov.Company.CEO)
	}
	if ov.EnterpriseValueKrw != 151607171797266 {
		t.Fatalf("unexpected EV KRW: %v", ov.EnterpriseValueKrw)
	}
	if ov.Company.IndustryName != "하드웨어및주변장치" {
		t.Fatalf("unexpected industry: %q", ov.Company.IndustryName)
	}
	if ov.Company.SharesOutstanding != 148089758 {
		t.Fatalf("unexpected sharesOutstanding: %d", ov.Company.SharesOutstanding)
	}
}
