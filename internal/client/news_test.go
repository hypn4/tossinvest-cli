package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestListStockNewsSamsung(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	news := mustReadFile(t, filepath.Join(root, "stock-news-samsung.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v2/news/companies/NAS116LTR-E0":
			w.Write(news)
		default:
			t.Fatalf("unexpected path: %s (query=%s)", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	items, err := c.ListStockNews(context.Background(), "NAS0250224006", 20)
	if err != nil {
		t.Fatalf("ListStockNews error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Source.Name != "아주경제" {
		t.Fatalf("unexpected first source: %q", items[0].Source.Name)
	}
}

func TestListStockFilingsSamsung(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	filings := mustReadFile(t, filepath.Join(root, "stock-filings-samsung.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v1/stock-detail/companies/NAS116LTR-E0/filings":
			w.Write(filings)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	items, err := c.ListStockFilings(context.Background(), "NAS0250224006", 20)
	if err != nil {
		t.Fatalf("ListStockFilings error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[1].Form != "EARNINGS" {
		t.Fatalf("expected second item form=EARNINGS, got %q", items[1].Form)
	}
	if items[1].EarningCall == nil || items[1].EarningCall.Status != "ENDED" {
		t.Fatalf("expected earningCall.status=ENDED, got %+v", items[1].EarningCall)
	}
	if items[0].EarningCall != nil {
		t.Fatalf("expected first item (HTML) to have nil earningCall")
	}
}

func TestListStockNewsRejectsBadCount(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.ListStockNews(context.Background(), "NAS0250224006", 0)
	if err == nil {
		t.Fatalf("expected error for count=0")
	}
}
