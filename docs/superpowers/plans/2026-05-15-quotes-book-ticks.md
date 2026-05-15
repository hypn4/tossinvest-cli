# Quotes book/ticks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `tossctl quotes book <sym>` and `tossctl quotes ticks <sym>` commands that expose the orderbook (KR 10-level / US top-of-book) and trade ticks discovered in the order-page reverse engineering, plus a `--follow` NDJSON streaming mode on `quotes ticks` with `cumulativeVolume` dedup.

**Architecture:** Three new domain types (`OrderBook`, `OrderBookLevel`, `Tick`) → two new client methods (`GetOrderBook`, `GetTicks`) + one streaming helper (`StreamTicks`) → output formatters that handle table/json/csv plus NDJSON for `--follow` → `tossctl quotes book|ticks` cobra commands. Everything else stays single-shot per advisor recommendation. Auth is **public** (`wts-info-api`) — no certified-API headers needed; the existing `Client.getJSON` already sends the session cookies it has.

**Tech Stack:** Go 1.x, cobra CLI, existing `internal/client.Client.getJSON` helper, `os/signal` for Ctrl-C handling in `--follow`, JSON line output. No new dependencies.

---

## Reference

- Capture write-up: [`docs/reverse-engineering/order-page-deep-dive.md`](../../reverse-engineering/order-page-deep-dive.md) sections `3.4 호가창`, `3.5 체결 틱`, appendix `C 호가창 (Orderbook)`
- RPC catalog: [`docs/reverse-engineering/rpc-catalog.md`](../../reverse-engineering/rpc-catalog.md) Quote table rows for `/api/v3/stock-prices/{code}/quotes` and `/api/v2/stock-prices/{code}/ticks`
- Raw captures: `.captures/2026-05-15/us-soxl/quotes-orderbook.network-response`, `ticks-120.network-response`

Key facts the implementation must respect:

1. v3 quote endpoint uses **English keys** (`offerPrices/offerVolumes/bidPrices/bidVolumes`); v1/v2 use Korean-flavored keys (`sellPrices/sellQuantities`). **We use v3 only** — single endpoint, single shape.
2. **KR returns 10 levels** + `midPrices`/`estimatedPrice`/`upperLimit`/`lowerLimit` (these come from `/details`, not `/quotes` — but the orderbook `singlePrice` flag is in `/quotes`). **US returns top-of-book (1 level)**.
3. Tick response is **newest-first**, no date in `time` (`HH:MM:SS` only). Dedup key = `cumulativeVolume` (monotonic per session).
4. `--follow` must emit oldest-first NDJSON (so downstream consumers see a chronological stream) even though the source is newest-first.

## File Structure

| File | Status | Responsibility |
| --- | --- | --- |
| `internal/domain/models.go` | Modify | Add `OrderBook`, `OrderBookLevel`, `Tick` types |
| `internal/client/quotes.go` | Create | `GetOrderBook`, `GetTicks`, `StreamTicks`, raw envelope decoding |
| `internal/client/quotes_test.go` | Create | Unit tests using httptest mock servers |
| `internal/output/quotes.go` | Create | `WriteOrderBook`, `WriteTicks`, `WriteTicksNDJSON` formatters |
| `internal/output/quotes_test.go` | Create | Formatter tests covering table/json/csv/ndjson |
| `cmd/tossctl/quotes.go` | Create | `tossctl quotes book` + `tossctl quotes ticks [--follow]` commands |
| `cmd/tossctl/root.go` | Modify | Register `newQuotesCmd(opts)` |
| `fixtures/responses/public/quotes-us-book.json` | Create | Sanitized SOXL orderbook (1 level) |
| `fixtures/responses/public/quotes-kr-book.json` | Create | Sanitized Samsung orderbook (10 levels) |
| `fixtures/responses/public/ticks-us.json` | Create | Sanitized SOXL ticks |
| `docs/reverse-engineering/rpc-catalog.md` | Modify | Update CLI mapping for orderbook + ticks rows |
| `CHANGELOG.md` | Modify | Add release notes |

---

## Task 1: Add OrderBook and Tick domain types

**Files:**
- Modify: `internal/domain/models.go`

- [ ] **Step 1: Add types at the bottom of the file**

Append to `internal/domain/models.go`:

```go
type OrderBookLevel struct {
	Price     float64 `json:"price"`
	PriceKRW  float64 `json:"price_krw,omitempty"`
	Volume    float64 `json:"volume"`
}

type OrderBook struct {
	ProductCode    string           `json:"product_code"`
	Symbol         string           `json:"symbol,omitempty"`
	Name           string           `json:"name,omitempty"`
	Market         string           `json:"market,omitempty"`
	Currency       string           `json:"currency,omitempty"`
	Last           float64          `json:"last,omitempty"`
	LastKRW        float64          `json:"last_krw,omitempty"`
	Offers         []OrderBookLevel `json:"offers"`
	Bids           []OrderBookLevel `json:"bids"`
	OfferVolumeSum float64          `json:"offer_volume_sum,omitempty"`
	BidVolumeSum   float64          `json:"bid_volume_sum,omitempty"`
	SinglePrice    bool             `json:"single_price,omitempty"`
	EstimatedPrice float64          `json:"estimated_price,omitempty"`
	EstimatedVolume float64         `json:"estimated_volume,omitempty"`
	FetchedAt      time.Time        `json:"fetched_at"`
}

type Tick struct {
	Time             string    `json:"time"`
	ProductCode      string    `json:"product_code"`
	Price            float64   `json:"price"`
	PriceKRW         float64   `json:"price_krw,omitempty"`
	Base             float64   `json:"base,omitempty"`
	Volume           float64   `json:"volume"`
	TradeType        string    `json:"trade_type"`
	CumulativeVolume float64   `json:"cumulative_volume"`
	FetchedAt        time.Time `json:"fetched_at,omitempty"`
}
```

- [ ] **Step 2: Verify the file still compiles**

Run: `go build ./internal/domain/...`
Expected: no output (build success).

- [ ] **Step 3: Commit**

```bash
git add internal/domain/models.go
git commit -m "feat(domain): add OrderBook, OrderBookLevel, Tick types"
```

---

## Task 2: Drop sanitized fixtures for orderbook + ticks

**Files:**
- Create: `fixtures/responses/public/quotes-us-book.json`
- Create: `fixtures/responses/public/quotes-kr-book.json`
- Create: `fixtures/responses/public/ticks-us.json`

- [ ] **Step 1: Create the US orderbook fixture**

The raw capture at `.captures/2026-05-15/us-soxl/quotes-orderbook.network-response` contains the response. Copy its JSON content (only one level, no PII) into the fixture:

```bash
cat > fixtures/responses/public/quotes-us-book.json <<'JSON'
{
  "result": {
    "close": 168.60,
    "closeKrw": 251517,
    "offerPrices": [168.79],
    "offerPricesKrw": [251800],
    "offerVolumes": [7],
    "bidPrices": [168.60],
    "bidPricesKrw": [251517],
    "bidVolumes": [2],
    "offerVolume": 7,
    "bidVolume": 2
  }
}
JSON
```

- [ ] **Step 2: Create the KR orderbook fixture (10 levels)**

Sample from a Samsung Electronics capture. Use the literal arrays observed in the deep-dive doc:

```bash
cat > fixtures/responses/public/quotes-kr-book.json <<'JSON'
{
  "result": {
    "close": 273500,
    "offerPrices": [278500, 278000, 277500, 277000, 276500, 276000, 275500, 275000, 274500, 274000],
    "offerVolumes": [42254, 65779, 31428, 28978, 18195, 59953, 21617, 39916, 34249, 28041],
    "bidPrices": [273500, 273000, 272500, 272000, 271500, 271000, 270500, 270000, 269500, 269000],
    "bidVolumes": [10069, 31985, 23652, 23670, 27173, 42680, 48332, 75745, 40444, 93203],
    "midPrices": [0, 0],
    "midOfferVolumes": [0, 0],
    "midBidVolumes": [0, 0],
    "singlePrice": false,
    "estimatedPrice": 0,
    "estimatedVolume": 0,
    "offerVolume": 370410,
    "bidVolume": 506978
  }
}
JSON
```

- [ ] **Step 3: Create the US ticks fixture (3 sample ticks)**

We only need a few ticks for the test, not the full 120. Use:

```bash
cat > fixtures/responses/public/ticks-us.json <<'JSON'
{
  "result": [
    {"time":"21:09:40","code":"US20100311002","price":168.60,"priceKrw":251517,"base":186.19,"baseKrw":277758,"volume":2,"tradeType":"SELL","cumulativeVolume":1731781},
    {"time":"21:09:39","code":"US20100311002","price":168.66,"priceKrw":251606,"base":186.19,"baseKrw":277758,"volume":1,"tradeType":"BUY","cumulativeVolume":1731779},
    {"time":"21:09:38","code":"US20100311002","price":168.65,"priceKrw":251592,"base":186.19,"baseKrw":277758,"volume":10,"tradeType":"BUY","cumulativeVolume":1731778}
  ]
}
JSON
```

- [ ] **Step 4: Validate fixtures parse as JSON**

Run: `for f in fixtures/responses/public/quotes-us-book.json fixtures/responses/public/quotes-kr-book.json fixtures/responses/public/ticks-us.json; do python3 -m json.tool "$f" >/dev/null && echo "$f ok"; done`
Expected output:
```
fixtures/responses/public/quotes-us-book.json ok
fixtures/responses/public/quotes-kr-book.json ok
fixtures/responses/public/ticks-us.json ok
```

- [ ] **Step 5: Update fixture manifest**

Read `fixtures/responses/public/manifest.json` and append entries. The existing entries follow a stable shape; mirror it:

```bash
python3 - <<'PY'
import json, pathlib
p = pathlib.Path("fixtures/responses/public/manifest.json")
data = json.loads(p.read_text())
new = [
    {"file": "quotes-us-book.json", "url": "https://wts-info-api.tossinvest.com/api/v3/stock-prices/US20100311002/quotes", "method": "GET"},
    {"file": "quotes-kr-book.json", "url": "https://wts-info-api.tossinvest.com/api/v3/stock-prices/A005930/quotes", "method": "GET"},
    {"file": "ticks-us.json", "url": "https://wts-info-api.tossinvest.com/api/v2/stock-prices/US20100311002/ticks?count=3", "method": "GET"},
]
existing = {e["file"] for e in data.get("fixtures", [])}
for entry in new:
    if entry["file"] not in existing:
        data["fixtures"].append(entry)
p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")
print("manifest updated")
PY
```

Expected: `manifest updated`.

- [ ] **Step 6: Commit**

```bash
git add fixtures/responses/public/
git commit -m "feat(fixtures): add sanitized orderbook and ticks samples"
```

---

## Task 3: Write the failing client tests for GetOrderBook

**Files:**
- Create: `internal/client/quotes_test.go`

- [ ] **Step 1: Write the initial test file with two GetOrderBook test cases**

```go
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
```

- [ ] **Step 2: Run the new tests; they must fail**

Run: `go test ./internal/client/ -run TestGetOrderBook -v`
Expected: compile error or `undefined: Client.GetOrderBook`. That is the failing state.

- [ ] **Step 3: Commit the failing tests**

```bash
git add internal/client/quotes_test.go
git commit -m "test(client): add failing GetOrderBook tests (US 1-level, KR 10-level)"
```

---

## Task 4: Implement GetOrderBook

**Files:**
- Create: `internal/client/quotes.go`

- [ ] **Step 1: Create the file with raw envelope + GetOrderBook**

```go
package client

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type orderBookEnvelope struct {
	Result struct {
		Close           float64   `json:"close"`
		CloseKrw        float64   `json:"closeKrw"`
		OfferPrices     []float64 `json:"offerPrices"`
		OfferPricesKrw  []float64 `json:"offerPricesKrw"`
		OfferVolumes    []float64 `json:"offerVolumes"`
		BidPrices       []float64 `json:"bidPrices"`
		BidPricesKrw    []float64 `json:"bidPricesKrw"`
		BidVolumes      []float64 `json:"bidVolumes"`
		OfferVolume     float64   `json:"offerVolume"`
		BidVolume       float64   `json:"bidVolume"`
		SinglePrice     bool      `json:"singlePrice"`
		EstimatedPrice  float64   `json:"estimatedPrice"`
		EstimatedVolume float64   `json:"estimatedVolume"`
	} `json:"result"`
}

// GetOrderBook fetches the latest orderbook snapshot for a symbol. KR markets
// return up to 10 levels per side; US returns only top-of-book.
func (c *Client) GetOrderBook(ctx context.Context, symbol string) (domain.OrderBook, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.OrderBook{}, err
	}

	info, _ := c.getStockInfo(ctx, productCode)

	endpoint := fmt.Sprintf("%s/api/v3/stock-prices/%s/quotes", c.infoBaseURL, productCode)
	var envelope orderBookEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.OrderBook{}, err
	}

	offers := assembleLevels(envelope.Result.OfferPrices, envelope.Result.OfferPricesKrw, envelope.Result.OfferVolumes)
	bids := assembleLevels(envelope.Result.BidPrices, envelope.Result.BidPricesKrw, envelope.Result.BidVolumes)

	// Offers come back highest→lowest; normalize to lowest→highest for table rendering.
	sort.SliceStable(offers, func(i, j int) bool { return offers[i].Price < offers[j].Price })
	// Bids come back highest→lowest already, which matches our presentation expectation.
	sort.SliceStable(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })

	return domain.OrderBook{
		ProductCode:     productCode,
		Symbol:          info.Symbol,
		Name:            info.Name,
		Market:          info.Market.DisplayName,
		Currency:        info.Currency,
		Last:            envelope.Result.Close,
		LastKRW:         envelope.Result.CloseKrw,
		Offers:          offers,
		Bids:            bids,
		OfferVolumeSum:  envelope.Result.OfferVolume,
		BidVolumeSum:    envelope.Result.BidVolume,
		SinglePrice:     envelope.Result.SinglePrice,
		EstimatedPrice:  envelope.Result.EstimatedPrice,
		EstimatedVolume: envelope.Result.EstimatedVolume,
		FetchedAt:       time.Now().UTC(),
	}, nil
}

func assembleLevels(prices, pricesKrw, volumes []float64) []domain.OrderBookLevel {
	n := len(prices)
	out := make([]domain.OrderBookLevel, 0, n)
	for i := 0; i < n; i++ {
		level := domain.OrderBookLevel{Price: prices[i]}
		if i < len(pricesKrw) {
			level.PriceKRW = pricesKrw[i]
		}
		if i < len(volumes) {
			level.Volume = volumes[i]
		}
		out = append(out, level)
	}
	return out
}

// Suppress "imported and not used" errors while later tasks add GetTicks; remove later.
var _ = strconv.Itoa
var _ = url.Parse
```

- [ ] **Step 2: Run the tests; they must pass**

Run: `go test ./internal/client/ -run TestGetOrderBook -v`
Expected: `--- PASS: TestGetOrderBookUS` and `--- PASS: TestGetOrderBookKR`.

- [ ] **Step 3: Commit**

```bash
git add internal/client/quotes.go
git commit -m "feat(client): implement GetOrderBook (KR 10-level, US 1-level)"
```

---

## Task 5: Write the failing GetTicks test

**Files:**
- Modify: `internal/client/quotes_test.go`

- [ ] **Step 1: Append the GetTicks test**

Add at the bottom of `internal/client/quotes_test.go`:

```go
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
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/client/ -run TestGetTicks -v`
Expected: `undefined: Client.GetTicks`.

- [ ] **Step 3: Commit**

```bash
git add internal/client/quotes_test.go
git commit -m "test(client): add failing GetTicks test"
```

---

## Task 6: Implement GetTicks

**Files:**
- Modify: `internal/client/quotes.go`

- [ ] **Step 1: Replace the placeholder imports stubs and add GetTicks**

Replace the trailing two `var _ = …` lines with the GetTicks implementation. The full block to append (after the existing `assembleLevels` function) is:

```go
type tickEnvelope struct {
	Result []struct {
		Time             string  `json:"time"`
		Code             string  `json:"code"`
		Price            float64 `json:"price"`
		PriceKrw         float64 `json:"priceKrw"`
		Base             float64 `json:"base"`
		BaseKrw          float64 `json:"baseKrw"`
		Volume           float64 `json:"volume"`
		TradeType        string  `json:"tradeType"`
		CumulativeVolume float64 `json:"cumulativeVolume"`
	} `json:"result"`
}

// GetTicks returns up to count recent trade ticks newest-first.
func (c *Client) GetTicks(ctx context.Context, symbol string, count int) ([]domain.Tick, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 50
	}

	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v2/stock-prices/%s/ticks", c.infoBaseURL, productCode))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("count", strconv.Itoa(count))
	endpoint.RawQuery = q.Encode()

	var envelope tickEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.Tick, 0, len(envelope.Result))
	fetchedAt := time.Now().UTC()
	for _, raw := range envelope.Result {
		out = append(out, domain.Tick{
			Time:             raw.Time,
			ProductCode:      raw.Code,
			Price:            raw.Price,
			PriceKRW:         raw.PriceKrw,
			Base:             raw.Base,
			Volume:           raw.Volume,
			TradeType:        raw.TradeType,
			CumulativeVolume: raw.CumulativeVolume,
			FetchedAt:        fetchedAt,
		})
	}
	return out, nil
}
```

Then remove the temporary `var _ = strconv.Itoa` / `var _ = url.Parse` lines added in Task 4. `strconv` and `net/url` are now used by `GetTicks`.

- [ ] **Step 2: Run the test; it must pass**

Run: `go test ./internal/client/ -run TestGetTicks -v`
Expected: `--- PASS: TestGetTicks`.

- [ ] **Step 3: Run the whole client suite as a regression check**

Run: `go test ./internal/client/`
Expected: `ok  github.com/junghoonkye/tossinvest-cli/internal/client`.

- [ ] **Step 4: Commit**

```bash
git add internal/client/quotes.go
git commit -m "feat(client): implement GetTicks snapshot fetcher"
```

---

## Task 7: Streaming helper — StreamTicks with cumulativeVolume dedup

**Files:**
- Modify: `internal/client/quotes.go`
- Modify: `internal/client/quotes_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/client/quotes_test.go`:

```go
import "sync/atomic"

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
		Symbol:        "US20100311002",
		Count:         3,
		Interval:      1, // 1ns — effectively immediate; we drive iterations with ctx cancel
		Since:         0,
		OnError:       nil,
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
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/client/ -run TestStreamTicksDedupsByCumulativeVolume -v`
Expected: `undefined: Client.StreamTicks` or `undefined: StreamTicksOptions`.

- [ ] **Step 3: Implement StreamTicks in quotes.go**

Append to `internal/client/quotes.go`:

```go
// StreamTicksOptions controls a long-running tick stream.
type StreamTicksOptions struct {
	Symbol   string
	Count    int           // request size each poll; default 50
	Interval time.Duration // poll cadence; default 2s
	Since    float64       // resume from this cumulativeVolume (exclusive); 0 emits the initial snapshot
	OnError  func(error)   // optional non-fatal error sink; default discards
}

// TickStream drives a tick poll loop and emits new ticks oldest-first.
type TickStream struct {
	client  *Client
	opts    StreamTicksOptions
	out     chan domain.Tick
	stopOnce sync.Once
	stop    chan struct{}
	cursor  float64
}

// StreamTicks builds a TickStream; call Run(ctx) to drive it.
func (c *Client) StreamTicks(opts StreamTicksOptions) (*TickStream, error) {
	if strings.TrimSpace(opts.Symbol) == "" {
		return nil, fmt.Errorf("StreamTicks: symbol is required")
	}
	if opts.Count <= 0 {
		opts.Count = 50
	}
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	return &TickStream{
		client: c,
		opts:   opts,
		out:    make(chan domain.Tick, opts.Count),
		stop:   make(chan struct{}),
		cursor: opts.Since,
	}, nil
}

// Ticks returns the channel callers consume.
func (s *TickStream) Ticks() <-chan domain.Tick { return s.out }

// Close stops the stream loop. Safe to call multiple times.
func (s *TickStream) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Run polls until ctx is cancelled or Close is called. It blocks; emit loops
// should run it in a goroutine.
func (s *TickStream) Run(ctx context.Context) error {
	defer close(s.out)
	timer := time.NewTimer(0) // fire immediately for first poll
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stop:
			return nil
		case <-timer.C:
			ticks, err := s.client.GetTicks(ctx, s.opts.Symbol, s.opts.Count)
			if err != nil {
				if s.opts.OnError != nil {
					s.opts.OnError(err)
				}
			} else {
				s.emit(ticks)
			}
			timer.Reset(s.opts.Interval)
		}
	}
}

// emit converts the newest-first snapshot into chronological NDJSON-friendly
// order and advances the cumulativeVolume cursor.
func (s *TickStream) emit(snapshot []domain.Tick) {
	// Snapshot is newest→oldest; iterate in reverse for chronological emit.
	for i := len(snapshot) - 1; i >= 0; i-- {
		tick := snapshot[i]
		if tick.CumulativeVolume <= s.cursor {
			continue
		}
		select {
		case s.out <- tick:
			s.cursor = tick.CumulativeVolume
		case <-s.stop:
			return
		}
	}
}
```

Make sure the `sync` and `strings` imports are added to the existing import block at the top of the file.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/client/ -run TestStreamTicksDedupsByCumulativeVolume -v`
Expected: `--- PASS: TestStreamTicksDedupsByCumulativeVolume`.

- [ ] **Step 5: Run the whole client suite**

Run: `go test ./internal/client/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/client/quotes.go internal/client/quotes_test.go
git commit -m "feat(client): add StreamTicks with cumulativeVolume dedup"
```

---

## Task 8: Output formatter for OrderBook

**Files:**
- Create: `internal/output/quotes.go`
- Create: `internal/output/quotes_test.go`

- [ ] **Step 1: Write the failing formatter test**

```go
package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var sampleBook = domain.OrderBook{
	ProductCode:    "US20100311002",
	Symbol:         "SOXL",
	Market:         "AMEX",
	Currency:       "USD",
	Last:           168.60,
	Offers:         []domain.OrderBookLevel{{Price: 168.79, Volume: 7}},
	Bids:           []domain.OrderBookLevel{{Price: 168.60, Volume: 2}},
	OfferVolumeSum: 7,
	BidVolumeSum:   2,
	FetchedAt:      time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
}

func TestWriteOrderBookJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatJSON, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OrderBook
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if parsed.Symbol != "SOXL" || len(parsed.Offers) != 1 {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
}

func TestWriteOrderBookTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatTable, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "SOXL") {
		t.Fatalf("expected SOXL in output: %s", out)
	}
	if !strings.Contains(out, "168.79") || !strings.Contains(out, "168.60") {
		t.Fatalf("expected prices in output: %s", out)
	}
}

func TestWriteOrderBookCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatCSV, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header + at least one offer + one bid line, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "side,price,volume") {
		t.Fatalf("unexpected header: %s", lines[0])
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteOrderBook -v`
Expected: `undefined: WriteOrderBook`.

- [ ] **Step 3: Implement WriteOrderBook**

Create `internal/output/quotes.go`:

```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOrderBook renders an orderbook snapshot.
func WriteOrderBook(w io.Writer, format Format, book domain.OrderBook) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(book)
	case FormatCSV:
		return writeOrderBookCSV(w, book)
	case FormatTable:
		return writeOrderBookTable(w, book)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeOrderBookCSV(w io.Writer, book domain.OrderBook) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"side", "price", "volume"}); err != nil {
		return err
	}
	for _, level := range book.Offers {
		if err := writer.Write([]string{"offer", formatFloat(level.Price), formatFloat(level.Volume)}); err != nil {
			return err
		}
	}
	for _, level := range book.Bids {
		if err := writer.Write([]string{"bid", formatFloat(level.Price), formatFloat(level.Volume)}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeOrderBookTable(w io.Writer, book domain.OrderBook) error {
	if _, err := fmt.Fprintf(
		w,
		"%s (%s)  last=%s  offers=%d  bids=%d\n",
		labelFor(book.Symbol, book.Name, book.ProductCode),
		book.Market,
		formatFloat(book.Last),
		len(book.Offers),
		len(book.Bids),
	); err != nil {
		return err
	}
	if book.SinglePrice {
		if _, err := fmt.Fprintf(
			w, "(single price) est=%s  est_volume=%s\n",
			formatFloat(book.EstimatedPrice), formatFloat(book.EstimatedVolume),
		); err != nil {
			return err
		}
	}
	headers := []string{"SIDE", "PRICE", "VOLUME"}
	rows := make([][]string, 0, len(book.Offers)+len(book.Bids))
	for i := len(book.Offers) - 1; i >= 0; i-- {
		level := book.Offers[i]
		rows = append(rows, []string{"OFFER", formatFloat(level.Price), formatFloat(level.Volume)})
	}
	for _, level := range book.Bids {
		rows = append(rows, []string{"BID", formatFloat(level.Price), formatFloat(level.Volume)})
	}
	return renderTable(w, headers, rows)
}

func labelFor(symbol, name, productCode string) string {
	if symbol != "" && name != "" && symbol != name {
		return fmt.Sprintf("%s — %s", symbol, name)
	}
	if symbol != "" {
		return symbol
	}
	return productCode
}
```

- [ ] **Step 4: Run the tests; they must pass**

Run: `go test ./internal/output/ -run TestWriteOrderBook -v`
Expected: three `PASS` lines.

- [ ] **Step 5: Commit**

```bash
git add internal/output/quotes.go internal/output/quotes_test.go
git commit -m "feat(output): add OrderBook formatter (table/json/csv)"
```

---

## Task 9: Output formatter for Ticks (snapshot)

**Files:**
- Modify: `internal/output/quotes.go`
- Modify: `internal/output/quotes_test.go`

- [ ] **Step 1: Write the failing tick formatter test**

Append to `internal/output/quotes_test.go`:

```go
var sampleTicks = []domain.Tick{
	{Time: "21:09:40", ProductCode: "US20100311002", Price: 168.60, Volume: 2, TradeType: "SELL", CumulativeVolume: 1731781},
	{Time: "21:09:39", ProductCode: "US20100311002", Price: 168.66, Volume: 1, TradeType: "BUY", CumulativeVolume: 1731779},
}

func TestWriteTicksJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTicks(&buf, FormatJSON, sampleTicks); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.Tick
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(parsed))
	}
}

func TestWriteTicksCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTicks(&buf, FormatCSV, sampleTicks); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d", len(lines))
	}
}

func TestWriteTicksTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTicks(&buf, FormatTable, sampleTicks); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "21:09:40") {
		t.Fatalf("expected first tick time in table: %s", buf.String())
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteTicks -v`
Expected: `undefined: WriteTicks`.

- [ ] **Step 3: Implement WriteTicks**

Append to `internal/output/quotes.go`:

```go
// WriteTicks renders a snapshot of ticks newest-first.
func WriteTicks(w io.Writer, format Format, ticks []domain.Tick) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(ticks)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"time", "code", "price", "volume", "trade_type", "cumulative_volume"}); err != nil {
			return err
		}
		for _, t := range ticks {
			if err := writer.Write([]string{
				t.Time, t.ProductCode,
				formatFloat(t.Price), formatFloat(t.Volume),
				t.TradeType, formatFloat(t.CumulativeVolume),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"TIME", "TYPE", "PRICE", "VOLUME", "CUMVOL"}
		rows := make([][]string, 0, len(ticks))
		for _, t := range ticks {
			rows = append(rows, []string{
				t.Time, t.TradeType, formatFloat(t.Price), formatFloat(t.Volume), formatFloat(t.CumulativeVolume),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 4: Run the tests; they must pass**

Run: `go test ./internal/output/ -run TestWriteTicks -v`
Expected: three PASS lines.

- [ ] **Step 5: Commit**

```bash
git add internal/output/quotes.go internal/output/quotes_test.go
git commit -m "feat(output): add Tick snapshot formatter (table/json/csv)"
```

---

## Task 10: NDJSON writer for streaming ticks

**Files:**
- Modify: `internal/output/quotes.go`
- Modify: `internal/output/quotes_test.go`

- [ ] **Step 1: Write the failing NDJSON test**

Append to `internal/output/quotes_test.go`:

```go
func TestWriteTickNDJSON(t *testing.T) {
	var buf bytes.Buffer
	for _, tick := range sampleTicks {
		if err := WriteTickNDJSON(&buf, tick); err != nil {
			t.Fatalf("error: %v", err)
		}
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}
	var first domain.Tick
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first line not valid JSON: %v", err)
	}
	if first.CumulativeVolume != 1731781 {
		t.Fatalf("unexpected first tick: %+v", first)
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteTickNDJSON -v`
Expected: `undefined: WriteTickNDJSON`.

- [ ] **Step 3: Implement WriteTickNDJSON**

Append to `internal/output/quotes.go`:

```go
// WriteTickNDJSON writes a single tick as a single JSON object terminated by '\n'.
// Newline-delimited JSON is what consumers (jq, LLM pipelines) expect from
// `tossctl quotes ticks --follow`.
func WriteTickNDJSON(w io.Writer, tick domain.Tick) error {
	data, err := json.Marshal(tick)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
```

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/output/ -run TestWriteTickNDJSON -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output/quotes.go internal/output/quotes_test.go
git commit -m "feat(output): add WriteTickNDJSON for --follow streams"
```

---

## Task 11: CLI command — tossctl quotes book

**Files:**
- Create: `cmd/tossctl/quotes.go`
- Modify: `cmd/tossctl/root.go`

- [ ] **Step 1: Create the cobra command skeleton with `quotes book`**

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newQuotesCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quotes",
		Short: "Read orderbook and tick data",
	}

	bookCmd := &cobra.Command{
		Use:   "book <symbol>",
		Short: "Show the latest orderbook (호가창)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			book, err := app.client.GetOrderBook(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOrderBook(cmd.OutOrStdout(), app.format, book)
		},
	}

	cmd.AddCommand(bookCmd)
	return cmd
}

// Suppress unused imports until Task 12 lands.
var (
	_ = context.Background
	_ = errors.New
	_ = fmt.Sprintf
	_ = os.Stdin
	_ = signal.Notify
	_ = syscall.SIGINT
	_ = time.Second
)

var (
	_ = client.StreamTicksOptions{}
)
```

- [ ] **Step 2: Register the command in root.go**

Update `cmd/tossctl/root.go`:

```go
		newChartCmd(opts),
		newQuotesCmd(opts),
		newOrderCmd(opts),
```

(Insert `newQuotesCmd(opts),` directly after `newChartCmd(opts),`.)

- [ ] **Step 3: Build and run help to confirm**

Run: `go build ./...`
Expected: no output.

Run: `go run ./cmd/tossctl quotes --help`
Expected output includes:
```
Available Commands:
  book        Show the latest orderbook (호가창)
```

- [ ] **Step 4: Commit**

```bash
git add cmd/tossctl/quotes.go cmd/tossctl/root.go
git commit -m "feat(cli): add tossctl quotes book command"
```

---

## Task 12: CLI command — tossctl quotes ticks (snapshot + --follow)

**Files:**
- Modify: `cmd/tossctl/quotes.go`

- [ ] **Step 1: Replace the unused-import stubs with the ticks subcommand**

In `cmd/tossctl/quotes.go`, delete the trailing `var ( _ = context.Background ... )` blocks and instead extend the cmd builder. The full revised body of `newQuotesCmd`:

```go
func newQuotesCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quotes",
		Short: "Read orderbook and tick data",
	}

	bookCmd := &cobra.Command{
		Use:   "book <symbol>",
		Short: "Show the latest orderbook (호가창)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			book, err := app.client.GetOrderBook(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOrderBook(cmd.OutOrStdout(), app.format, book)
		},
	}

	var (
		ticksCount    int
		ticksFollow   bool
		ticksInterval time.Duration
		ticksSince    float64
	)
	ticksCmd := &cobra.Command{
		Use:   "ticks <symbol>",
		Short: "Show recent trade ticks (체결 틱); --follow for NDJSON stream",
		Long: `Show recent trade ticks for a symbol.

Without --follow this prints a snapshot of the last --count ticks (newest first).

With --follow this becomes a long-running stream: ticks are emitted as
newline-delimited JSON (NDJSON) in chronological order, deduped by
cumulativeVolume. Use Ctrl-C to stop.

Examples:
  tossctl quotes ticks SOXL --count 20
  tossctl quotes ticks SOXL --follow --interval 2s
  tossctl quotes ticks SOXL --follow --since 1731780`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if !ticksFollow {
				ticks, err := app.client.GetTicks(cmd.Context(), args[0], ticksCount)
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteTicks(cmd.OutOrStdout(), app.format, ticks)
			}
			return runTicksFollow(cmd.Context(), app, args[0], ticksCount, ticksInterval, ticksSince, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	ticksCmd.Flags().IntVar(&ticksCount, "count", 50, "Number of ticks per fetch (1..)")
	ticksCmd.Flags().BoolVar(&ticksFollow, "follow", false, "Stream new ticks as NDJSON until Ctrl-C")
	ticksCmd.Flags().DurationVar(&ticksInterval, "interval", 2*time.Second, "Poll interval when --follow is set")
	ticksCmd.Flags().Float64Var(&ticksSince, "since", 0, "Resume from this cumulativeVolume (exclusive)")

	cmd.AddCommand(bookCmd, ticksCmd)
	return cmd
}

func runTicksFollow(ctx context.Context, app *appContext, symbol string, count int, interval time.Duration, since float64, stdout, stderr *os.File) error {
	stream, err := app.client.StreamTicks(client.StreamTicksOptions{
		Symbol:   symbol,
		Count:    count,
		Interval: interval,
		Since:    since,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "tick poll error: %v\n", err)
		},
	})
	if err != nil {
		return userFacingCommandError(err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Run(sigCtx)
	}()

	for {
		select {
		case tick, ok := <-stream.Ticks():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteTickNDJSON(stdout, tick); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
```

Note: `*os.File` for stdout/stderr matches what cobra returns from `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` — both return `io.Writer`, so adjust the signature to `io.Writer` if a build error appears. Use `io.Writer` from the start:

Update the function signature line:

```go
func runTicksFollow(ctx context.Context, app *appContext, symbol string, count int, interval time.Duration, since float64, stdout, stderr io.Writer) error {
```

Add `"io"` to the import block.

- [ ] **Step 2: Build the binary**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Verify help output**

Run: `go run ./cmd/tossctl quotes ticks --help`
Expected output includes:
```
Flags:
      --count int             Number of ticks per fetch (1..) (default 50)
      --follow                Stream new ticks as NDJSON until Ctrl-C
      --interval duration     Poll interval when --follow is set (default 2s)
      --since float           Resume from this cumulativeVolume (exclusive)
```

- [ ] **Step 4: Commit**

```bash
git add cmd/tossctl/quotes.go
git commit -m "feat(cli): add tossctl quotes ticks (--follow NDJSON streaming)"
```

---

## Task 13: Update RPC catalog with CLI mappings

**Files:**
- Modify: `docs/reverse-engineering/rpc-catalog.md`

- [ ] **Step 1: Locate the quotes/ticks rows**

Search the catalog for the orderbook v3 row and the ticks row. Both currently have empty `CLI mapping` text or only describe Toss internals.

- [ ] **Step 2: Update the CLI mapping cell for v3 quotes**

Replace `quotes book` (placeholder) in the v3 row with:

```
quotes book <sym> (since v0.5.x)
```

Do the same for the ticks row, replacing the cell with:

```
quotes ticks <sym> [--follow] (since v0.5.x)
```

Use Edit to make the swap (`Edit replace_all=false`, locate the exact text snippet from the catalog).

- [ ] **Step 3: Commit**

```bash
git add docs/reverse-engineering/rpc-catalog.md
git commit -m "docs(catalog): wire CLI mappings for quotes book/ticks"
```

---

## Task 14: CHANGELOG entry

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Read the existing top of CHANGELOG.md to match formatting**

Run: `head -30 CHANGELOG.md`

- [ ] **Step 2: Add a new "Unreleased" or next version block following the existing style**

Add immediately under the latest version header:

```markdown
### Added
- `tossctl quotes book <sym>` — 토스 호가창 조회. KR 종목은 10단계, US 종목은 Top-of-book.
- `tossctl quotes ticks <sym>` — 최근 체결 틱 스냅샷 (`--count 50` 기본). `--follow` 옵션으로 NDJSON 스트림 (cumulativeVolume 으로 자동 dedup, `--interval 2s`, `--since <cumvol>` 옵션). 참고: docs/reverse-engineering/order-page-deep-dive.md §3.4-3.5.
```

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): note tossctl quotes book/ticks additions"
```

---

## Task 15: Full integration sweep

**Files:** (no edits)

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 2: Vet everything**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Run the full test suite**

Run: `go test ./...`
Expected: every package reports `ok` (or `no test files`).

- [ ] **Step 4: Smoke-test the new commands' help text**

Run: `go run ./cmd/tossctl quotes book --help`
Expected: shows symbol arg + global flags.

Run: `go run ./cmd/tossctl quotes ticks --help`
Expected: shows --follow, --count, --interval, --since.

- [ ] **Step 5: Tag a candidate release marker** (optional; do not push)

```bash
git log --oneline -10
```
Expected: clean sequence of PR2 commits.

- [ ] **Step 6: Final commit if any stragglers**

If `git status` shows any unstaged or untracked artifacts:

```bash
git status
```
Expected: `nothing to commit, working tree clean`.

If anything remains uncommitted, inspect and commit with an appropriate message.

---

## Self-review notes

- All endpoints, body shapes, and field names are taken verbatim from `docs/reverse-engineering/order-page-deep-dive.md` and the matching `.captures/` raw responses.
- `Tick.FetchedAt` is set on the client side; the API does not provide a stable timestamp per tick.
- Tests use `httptest` mock servers + the existing `fixtures/responses/public/` directory; no live network calls.
- `--follow` cancellation: `signal.NotifyContext` covers Ctrl-C and SIGTERM (cron `kill`); the stream goroutine respects `ctx.Done()` immediately because `Run` selects on it.
- `cumulativeVolume` cursor is per-session in Toss's model. Across the KST midnight session boundary the cursor resets; document that in `--help` long text if a user reports drift. Right now we emit "ticks where cumulativeVolume > cursor", which means the very first poll after a reset will emit the new session's snapshot in full — acceptable for v0.
