package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
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
