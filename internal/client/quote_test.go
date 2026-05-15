package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestGetQuoteFromFixtures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// New code-path endpoints are tried first; if a fixture is not yet
		// available we return 404 so GetQuote falls back to the legacy
		// /api/v1/product/stock-prices path the original fixtures still cover.
		switch r.URL.Path {
		case "/api/v3/stock-prices/details",
			"/api/v1/stock-infos/header/A005930":
			http.NotFound(w, r)
			return
		}
		fixturePath := fixturePathForRequest(t, r.URL.Path)
		http.ServeFile(w, r, fixturePath)
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		InfoBaseURL: server.URL,
	})

	quote, err := client.GetQuote(context.Background(), "005930")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}

	if quote.ProductCode != "A005930" {
		t.Fatalf("unexpected product code: %s", quote.ProductCode)
	}

	if quote.Symbol != "005930" {
		t.Fatalf("unexpected symbol: %s", quote.Symbol)
	}

	if quote.Name != "삼성전자" {
		t.Fatalf("unexpected name: %s", quote.Name)
	}

	if quote.Last != 193900 {
		t.Fatalf("unexpected last price: %v", quote.Last)
	}

	if quote.ReferencePrice != 187900 {
		t.Fatalf("unexpected reference price: %v", quote.ReferencePrice)
	}

	if quote.Volume != 27306483 {
		t.Fatalf("unexpected volume: %v", quote.Volume)
	}
}

func TestGetQuoteResolvesUSSymbolViaSearch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/search/stocks":
			_, _ = w.Write([]byte(`{"result":{"stocks":[{"stockCode":"US20220809012","stockName":"TSLL","matchType":"EXACT"}]}}`))
		case r.URL.Path == "/api/v2/stock-infos/US20220809012":
			_, _ = w.Write([]byte(`{"result":{"symbol":"TSLL","name":"TSLL","currency":"USD","status":"N","market":{"code":"NSQ","displayName":"NASDAQ"}}}`))
		case r.URL.Path == "/api/v3/stock-prices/details",
			r.URL.Path == "/api/v1/stock-infos/header/US20220809012":
			http.NotFound(w, r)
		case r.URL.Path == "/api/v1/product/stock-prices":
			_, _ = w.Write([]byte(`{"result":[{"productCode":"US20220809012","currency":"USD","base":14.38,"close":15.36,"volume":13409779}]}`))
		case r.URL.Path == "/api/v1/stock-detail/ui/US20220809012/common":
			_, _ = w.Write([]byte(`{"result":{"badges":[],"notices":[]}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		InfoBaseURL: server.URL,
	})

	quote, err := client.GetQuote(context.Background(), "TSLL")
	if err != nil {
		t.Fatalf("GetQuote returned error: %v", err)
	}
	if quote.ProductCode != "US20220809012" {
		t.Fatalf("unexpected product code: %s", quote.ProductCode)
	}
	if quote.Symbol != "TSLL" {
		t.Fatalf("unexpected symbol: %s", quote.Symbol)
	}
	if quote.Last != 15.36 {
		t.Fatalf("unexpected last price: %v", quote.Last)
	}
}

func fixturePathForRequest(t *testing.T, path string) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}

	root := filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")

	switch path {
	case "/api/v2/stock-infos/A005930":
		return mustPublicFixturePath(t, filepath.Join(root, "stock-info.json"))
	case "/api/v1/stock-detail/ui/A005930/common":
		return mustPublicFixturePath(t, filepath.Join(root, "stock-detail-common.json"))
	case "/api/v1/product/stock-prices":
		return mustPublicFixturePath(t, filepath.Join(root, "stock-price.json"))
	default:
		t.Fatalf("unexpected request path: %s", path)
		return ""
	}
}

func mustPublicFixturePath(t *testing.T, path string) string {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture missing: %s: %v", path, err)
	}
	return path
}

func TestStreamQuoteEmitsOnPriceOrVolumeChange(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	stockInfo := mustReadFile(t, filepath.Join(root, "stock-info.json"))

	bodies := [][]byte{
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:00","open":166,"high":172,"low":161,"close":167.10,"volume":1500000,"base":186.19}]}`),
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:01","open":166,"high":172,"low":161,"close":167.10,"volume":1501000,"base":186.19}]}`),
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:01","open":166,"high":172,"low":161,"close":167.10,"volume":1501000,"base":186.19}]}`),
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:02","open":166,"high":172,"low":161,"close":167.20,"volume":1501000,"base":186.19}]}`),
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v2/stock-infos/"):
			w.Write(stockInfo)
		case strings.Contains(r.URL.Path, "/api/v3/stock-prices/details"):
			idx := int(calls.Add(1)) - 1
			if idx >= len(bodies) {
				idx = len(bodies) - 1
			}
			w.Write(bodies[idx])
		case strings.HasPrefix(r.URL.Path, "/api/v1/stock-infos/header/"):
			w.Write([]byte(`{"result":{"sections":[]}}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/stock-detail/ui/"):
			w.Write([]byte(`{"result":{"badges":[],"notices":[]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	stream, err := c.StreamQuote(StreamQuoteOptions{
		Symbol:   "US20100311002",
		Interval: 1,
	})
	if err != nil {
		t.Fatalf("StreamQuote error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	emitted := []domain.Quote{}

	go func() {
		defer cancel()
		for q := range stream.Quotes() {
			emitted = append(emitted, q)
			if len(emitted) == 3 {
				time.Sleep(20 * time.Millisecond)
				stream.Close()
				return
			}
		}
	}()

	if err := stream.Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run returned %v", err)
	}

	if len(emitted) != 3 {
		t.Fatalf("expected 3 emits, got %d: %+v", len(emitted), emitted)
	}
	if emitted[0].Last != 167.10 || emitted[1].Volume != 1501000 || emitted[2].Last != 167.20 {
		t.Fatalf("unexpected emits: %+v", emitted)
	}
}
