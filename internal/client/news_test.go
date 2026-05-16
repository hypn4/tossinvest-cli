package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

func TestListStockNewsPaginationURLs(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))

	page1 := []byte(`{"result":{"pagingParam":{"number":1,"size":20},"body":[
		{"id":"a","title":"item 1","summary":"","source":{"code":"x","name":"X"},"createdAt":"2026-05-16T00:00:00"}
	],"lastPage":false}}`)
	page2 := []byte(`{"result":{"pagingParam":{"number":2,"size":20},"body":[
		{"id":"b","title":"item 2","summary":"","source":{"code":"x","name":"X"},"createdAt":"2026-05-15T00:00:00"}
	],"lastPage":true}}`)

	var gotQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v2/news/companies/NAS116LTR-E0":
			gotQueries = append(gotQueries, r.URL.RawQuery)
			// Differentiate by number= query
			if !strings.Contains(r.URL.RawQuery, "number=") {
				w.Write(page1)
			} else {
				w.Write(page2)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	items, err := c.ListStockNews(context.Background(), "NAS0250224006", 5)
	if err != nil {
		t.Fatalf("ListStockNews error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items (page1 + page2 then lastPage), got %d", len(items))
	}
	if len(gotQueries) != 2 {
		t.Fatalf("expected 2 news requests, got %d (queries=%v)", len(gotQueries), gotQueries)
	}
	if strings.Contains(gotQueries[0], "number=") {
		t.Errorf("page-1 URL should omit number=; got query %q", gotQueries[0])
	}
	if !strings.Contains(gotQueries[1], "number=2") {
		t.Errorf("page-2 URL should contain number=2; got query %q", gotQueries[1])
	}
}
