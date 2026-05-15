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

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func TestGetOrderBookUS(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/stock-infos/US20100311002":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
		case r.URL.Path == "/api/v3/stock-prices/US20100311002/quotes":
			http.ServeFile(w, r, filepath.Join(root, "quotes-us-book.json"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	book, err := c.GetOrderBook(context.Background(), "US20100311002")
	if err != nil {
		t.Fatalf("GetOrderBook returned error: %v", err)
	}
	if len(book.Offers) != 1 || len(book.Bids) != 1 {
		t.Fatalf("US book should be top-of-book, got %d offers / %d bids", len(book.Offers), len(book.Bids))
	}
	if book.Offers[0].Price != 168.79 || book.Offers[0].Volume != 7 {
		t.Fatalf("unexpected offer level: %+v", book.Offers[0])
	}
	if book.Bids[0].Price != 168.60 || book.Bids[0].Volume != 2 {
		t.Fatalf("unexpected bid level: %+v", book.Bids[0])
	}
	if book.OfferVolumeSum != 7 || book.BidVolumeSum != 2 {
		t.Fatalf("unexpected volume sums: offer=%v bid=%v", book.OfferVolumeSum, book.BidVolumeSum)
	}
}

func TestGetOrderBookKR(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/stock-infos/A005930":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
		case r.URL.Path == "/api/v3/stock-prices/A005930/quotes":
			http.ServeFile(w, r, filepath.Join(root, "quotes-kr-book.json"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	book, err := c.GetOrderBook(context.Background(), "005930")
	if err != nil {
		t.Fatalf("GetOrderBook returned error: %v", err)
	}
	if len(book.Offers) != 10 || len(book.Bids) != 10 {
		t.Fatalf("KR book should be 10 levels, got %d offers / %d bids", len(book.Offers), len(book.Bids))
	}
	// Offers must be sorted ascending in price for table rendering;
	// the API returns offers DESC (highest sell first). The client normalizes.
	if !(book.Offers[0].Price < book.Offers[1].Price) {
		t.Fatalf("offers should be price-ascending after normalization: %+v", book.Offers[:2])
	}
	// Bids must be sorted descending (best bid first); the API returns that order.
	if !(book.Bids[0].Price > book.Bids[1].Price) {
		t.Fatalf("bids should be price-descending: %+v", book.Bids[:2])
	}
	if !strings.EqualFold(book.Currency, "KRW") {
		t.Fatalf("expected KRW currency, got %s", book.Currency)
	}
}

func TestGetTicks(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/stock-infos/US20100311002":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
		case r.URL.Path == "/api/v2/stock-prices/US20100311002/ticks":
			if r.URL.Query().Get("count") != "3" {
				t.Fatalf("expected count=3 query param, got %q", r.URL.RawQuery)
			}
			http.ServeFile(w, r, filepath.Join(root, "ticks-us.json"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ticks, err := c.GetTicks(context.Background(), "US20100311002", 3)
	if err != nil {
		t.Fatalf("GetTicks returned error: %v", err)
	}
	if len(ticks) != 3 {
		t.Fatalf("expected 3 ticks, got %d", len(ticks))
	}
	// Source order is newest-first; our client must preserve order so callers can
	// dedup deterministically with cumulativeVolume.
	if ticks[0].CumulativeVolume != 1731781 || ticks[2].CumulativeVolume != 1731778 {
		t.Fatalf("unexpected order: %+v", ticks)
	}
	if ticks[0].TradeType != "SELL" || ticks[1].TradeType != "BUY" {
		t.Fatalf("trade type decoding broken: %+v", ticks[:2])
	}
}

func TestStreamTicksDedupsByCumulativeVolume(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/stock-infos/US20100311002":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
		case r.URL.Path == "/api/v2/stock-prices/US20100311002/ticks":
			calls.Add(1)
			http.ServeFile(w, r, filepath.Join(root, "ticks-us.json"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	emitted := []domain.Tick{}
	stream, err := c.StreamTicks(StreamTicksOptions{
		Symbol:   "US20100311002",
		Count:    3,
		Interval: 1, // 1ns — effectively immediate; we drive iterations with ctx cancel
		Since:    0,
		OnError:  nil,
	})
	if err != nil {
		t.Fatalf("StreamTicks returned error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		// Drain the first emit (3 ticks) then trigger a second poll that should emit zero new ticks.
		first := 0
		for tick := range stream.Ticks() {
			emitted = append(emitted, tick)
			first++
			if first == 3 {
				// Allow one more poll cycle to confirm dedup; then stop.
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
		t.Fatalf("expected 3 emitted ticks, got %d", len(emitted))
	}
	// Oldest-first ordering for downstream NDJSON
	if emitted[0].CumulativeVolume != 1731778 || emitted[2].CumulativeVolume != 1731781 {
		t.Fatalf("expected oldest-first order: %+v", emitted)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected at least 2 poll cycles, got %d", calls.Load())
	}
}
