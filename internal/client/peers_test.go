package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetTICSIndustryFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	tics := mustReadFile(t, filepath.Join(root, "tics-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v2/companies/NAS116LTR-E0/tics":
			w.Write(tics)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ind, err := c.GetTICSIndustry(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetTICSIndustry error: %v", err)
	}
	if len(ind.Major) != 1 {
		t.Fatalf("expected 1 major entry, got %d", len(ind.Major))
	}
	if ind.Major[0].Title != "컴퓨터와 주변기기" {
		t.Fatalf("unexpected title: %q", ind.Major[0].Title)
	}
	if ind.Major[0].CompanyCount != 85 {
		t.Fatalf("expected companyCount 85, got %d", ind.Major[0].CompanyCount)
	}
	if len(ind.Major[0].Rankings) != 3 {
		t.Fatalf("expected 3 rankings, got %d", len(ind.Major[0].Rankings))
	}
	if ind.Major[0].Rankings[0].TypeName != "시가총액" {
		t.Fatalf("unexpected first ranking type: %q", ind.Major[0].Rankings[0].TypeName)
	}
}
