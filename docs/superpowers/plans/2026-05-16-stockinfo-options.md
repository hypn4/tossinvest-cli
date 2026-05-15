# Stock Info Deep Tab + Options Read-Only Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Plan tasks use checkbox (`- [ ]`) syntax.

**Goal:** Expose two pieces of newly reverse-engineered surface area:
1. `/api/v1/stock-detail/ui/{code}/info` — the full 종목정보 deep tab (one endpoint, ~13 sections).
2. US options as first-class products — Toss already serves option prices/orderbook/ticks/charts through the same stock endpoints when given an `OPT_…` productCode. PR6 adds three thin commands (`options info`, `options nearest-atm`, `options chart`) and a one-line `us-o` prefix fix in `chartProductPrefix`.

**Architecture:**
- Three new client methods (`GetStockInfoDetail`, `GetOptionInstrument`, `GetNearestATMOption`) — small, mechanical, mirror existing patterns.
- One small fix in `internal/client/chart.go` `chartProductPrefix` to recognize `OPT_` productCodes and route to `us-o`.
- Two new output writers (`WriteStockInfoDetail`, `WriteOptionInstrument`) — JSON pass-through, simple table summary.
- Three new CLI commands (`tossctl stock info`, `tossctl options info`, `tossctl options nearest-atm`); existing `chart get` works on OPT_ codes once the prefix fix lands.

**Tech Stack:** Go, cobra (no new dependencies). Reuse `Client.getJSON`, existing fixture pattern.

---

## Reference

- RE write-up: [`docs/reverse-engineering/stock-info-deep-tab.md`](../../reverse-engineering/stock-info-deep-tab.md), [`docs/reverse-engineering/options.md`](../../reverse-engineering/options.md)
- Raw captures: `.captures/2026-05-16/sndk-options-stockinfo/`
- Existing patterns: `internal/client/quote.go` (`GetQuote`), `internal/client/chart.go` (`chartProductPrefix`), `internal/output/quote.go`
- Fork-only: branch `feat/pr6-stockinfo-options` is off `feat/order-page-integration`; push to `origin` only.

Key constraints:

1. Options chain enumeration endpoint is **NOT yet reverse-engineered**. PR6 implements everything that works on a known `OPT_…` productCode. Chain discovery is deferred until browser capture.
2. `/api/v1/stock-detail/ui/{code}/info` returns `{"result":null}` for some product codes — handle as "no info available", not an error.
3. Options charting requires `us-o` product prefix (vs `us-s` for stocks); a one-line branch in `chartProductPrefix` handles this.
4. Stock-info deep tab response is 30KB+ JSON; the CLI's default table view should be a one-line-per-section summary; full payload via `--output json`.

---

## Task 1: Fix `chartProductPrefix` to route `OPT_…` to `us-o`

**Files:**
- Modify: `internal/client/chart.go`
- Modify: `internal/client/chart_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/client/chart_test.go`:

```go
func TestChartProductPrefix_OptionRoutesToUSO(t *testing.T) {
	cases := []struct {
		name        string
		marketCode  string
		productCode string
		want        string
	}{
		{"OPT_ with NSQ market", "NSQ", "OPT_SNDK260515C01395000_20260506", "us-o"},
		{"OPT_ with empty market", "", "OPT_SNDK260515C01395000_20260506", "us-o"},
		{"NAS stock unaffected", "NSQ", "NAS0250224006", "us-s"},
		{"KR stock unaffected", "KSP", "A005930", "kr-s"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := chartProductPrefix(c.marketCode, c.productCode)
			if got != c.want {
				t.Fatalf("chartProductPrefix(%q,%q)=%q; want %q", c.marketCode, c.productCode, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test (must fail)**

```
go test ./internal/client/ -run TestChartProductPrefix_OptionRoutesToUSO -v
```

Expected: subtests for `OPT_…` cases fail (current code falls through to `us-s`).

- [ ] **Step 3: Implement**

In `internal/client/chart.go`, edit `chartProductPrefix` to add the OPT_ branch BEFORE the market-code switch:

```go
func chartProductPrefix(marketCode, productCode string) string {
	if strings.HasPrefix(productCode, "OPT_") {
		return "us-o"
	}
	mc := strings.ToLower(strings.TrimSpace(marketCode))
	switch mc {
	case "ksp", "ksq", "krx", "knx":
		return "kr-s"
	case "nys", "nas", "amx":
		return "us-s"
	}
	if strings.HasPrefix(productCode, "A") {
		return "kr-s"
	}
	return "us-s"
}
```

- [ ] **Step 4: Run test (must pass)**

`go test ./internal/client/ -run TestChartProductPrefix_OptionRoutesToUSO -v` → PASS.

- [ ] **Step 5: Run regression**

`go test ./internal/client/` → ok.

- [ ] **Step 6: Commit**

```bash
git add internal/client/chart.go internal/client/chart_test.go
git commit -m "feat(client): chartProductPrefix routes OPT_ codes to us-o"
```

---

## Task 2: Domain — `StockInfoDetail` raw + `OptionInstrument`

**Files:**
- Modify: `internal/domain/models.go`

- [ ] **Step 1: Add types at end of `internal/domain/models.go`**

```go
// StockInfoDetail is the full 종목정보 deep-tab payload. Sections are
// untyped on purpose: there are 13+ section types and the schemas vary
// independently (FINANCES, EARNINGS_AND_CONSENSUS, etc.). Consumers
// inspect `Sections[i].Type` and unmarshal `Data` themselves.
//
// Endpoint: GET /api/v1/stock-detail/ui/{productCode}/info
type StockInfoDetail struct {
	ProductCode string              `json:"product_code"`
	Sections    []StockInfoSection  `json:"sections"`
	FetchedAt   time.Time           `json:"fetched_at"`
}

type StockInfoSection struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// OptionInstrument captures the option-specific metadata returned by
// /api/v2/stock-infos/{OPT_…} under result.optionInstrument.
type OptionInstrument struct {
	ProductCode      string  `json:"product_code"`
	MarketCode       string  `json:"market_code,omitempty"`
	RootSymbol       string  `json:"root_symbol,omitempty"`
	Name             string  `json:"name,omitempty"`
	FullName         string  `json:"full_name,omitempty"`
	CompleteName     string  `json:"complete_name,omitempty"`
	UnderlyingSymbol string  `json:"underlying_symbol,omitempty"`
	UnderlyingGuid   string  `json:"underlying_guid,omitempty"`
	UnderlyingName   string  `json:"underlying_name,omitempty"`
	MaturityDate     string  `json:"maturity_date,omitempty"`
	MaturityDateTime string  `json:"maturity_date_time,omitempty"`
	PutCall          string  `json:"put_call,omitempty"`
	StrikePrice      float64 `json:"strike_price,omitempty"`
	BasePrice        float64 `json:"base_price,omitempty"`
	Last             float64 `json:"last,omitempty"`
	Bid              float64 `json:"bid,omitempty"`
	Ask              float64 `json:"ask,omitempty"`
	Mid              float64 `json:"mid,omitempty"`
	ContractUnit     float64 `json:"contract_unit,omitempty"`
	OpenInterest     int     `json:"open_interest,omitempty"`
	Halted           bool    `json:"halted,omitempty"`
	TradingSuspended bool    `json:"trading_suspended,omitempty"`
	BuySuspended     bool    `json:"buy_suspended,omitempty"`
	SellSuspended    bool    `json:"sell_suspended,omitempty"`
	Status           string  `json:"status,omitempty"`
	Overtime         bool    `json:"overtime,omitempty"`
	LiquidationDisplay string `json:"liquidation_display,omitempty"`
	LiquidationDateTime string `json:"liquidation_date_time,omitempty"`
	PennyPilot       bool    `json:"penny_pilot,omitempty"`
	FetchedAt        time.Time `json:"fetched_at"`
}
```

Make sure `encoding/json` is in the import block (it already is per `Quote`'s `BadgeCount` use).

- [ ] **Step 2: Build check**

`go build ./...` → success (no tests yet).

- [ ] **Step 3: Commit**

```bash
git add internal/domain/models.go
git commit -m "feat(domain): add StockInfoDetail and OptionInstrument types"
```

---

## Task 3: Client — `GetStockInfoDetail`

**Files:**
- Create: `internal/client/stockinfo.go`
- Create: `internal/client/stockinfo_test.go`
- Create: `fixtures/responses/public/stock-detail-ui-info-snxx.json` (sanitized excerpt)
- Modify: `fixtures/responses/public/manifest.json`

- [ ] **Step 1: Write a fixture**

Save a minimal version of the real SNDK response in `fixtures/responses/public/stock-detail-ui-info-snxx.json`. Use this 2-section stub so the fixture is portable and easy to assert against:

```json
{
  "result": {
    "sections": [
      {
        "type": "OVERVIEW",
        "data": {
          "logoUrl": "https://static.toss.im/png-icons/securities/icn-sec-fill-SNDK.png",
          "name": "샌디스크",
          "subTitle": "미국|SNDK|나스닥",
          "description": "SD카드, USB 플래시 드라이브 등의 메모리 제품을 판매하는 회사",
          "link": "/companies/NAS116LTR-E0/",
          "extraInfo": [],
          "tics": {"baseDate": "2020-12-01T00:00:00", "major": [], "minor": []}
        }
      },
      {
        "type": "INDICATORS",
        "data": {
          "values": [
            {"label":"시가총액","value":306146648589108,"displayValue":"306.1조원","dollarValue":204766670182,"displayDollarValue":"$2,047.7억","pointDate":"2026-05-14"},
            {"label":"PER","value":45.43,"displayValue":"45.4배","dollarValue":null,"displayDollarValue":"45.4배","pointDate":"2026-05-14"}
          ]
        }
      }
    ]
  }
}
```

- [ ] **Step 2: Add manifest entry**

Append to `fixtures/responses/public/manifest.json` inside the `fixtures` array:

```json
{
  "file": "stock-detail-ui-info-snxx.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/stock-detail/ui/NAS0250224006/info",
  "method": "GET"
}
```

- [ ] **Step 3: Write the failing test**

Create `internal/client/stockinfo_test.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockInfoDetailFromFixture(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-detail-ui-info-snxx.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
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
}
```

- [ ] **Step 4: Run test (must fail)**

`go test ./internal/client/ -run TestGetStockInfoDetailFromFixture -v`
Expected: `undefined: Client.GetStockInfoDetail`.

- [ ] **Step 5: Implement**

Create `internal/client/stockinfo.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// GetStockInfoDetail fetches the 종목정보 deep-tab payload for a product. The
// endpoint returns {"result": null} for products without coverage; this is
// surfaced as an empty Sections slice (no error).
func (c *Client) GetStockInfoDetail(ctx context.Context, symbol string) (domain.StockInfoDetail, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockInfoDetail{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/stock-detail/ui/%s/info", c.infoBaseURL, productCode)
	var envelope struct {
		Result *struct {
			Sections []domain.StockInfoSection `json:"sections"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.StockInfoDetail{}, err
	}
	out := domain.StockInfoDetail{
		ProductCode: productCode,
		FetchedAt:   time.Now().UTC(),
	}
	if envelope.Result != nil {
		out.Sections = envelope.Result.Sections
	}
	return out, nil
}

// Compile-time check that StockInfoSection.Data marshals back as expected.
var _ = json.Marshal
```

- [ ] **Step 6: Run test (must pass)**

`go test ./internal/client/ -run TestGetStockInfoDetailFromFixture -v` → PASS.

- [ ] **Step 7: Run regression**

`go test ./internal/client/` → ok.

- [ ] **Step 8: Commit**

```bash
git add internal/client/stockinfo.go internal/client/stockinfo_test.go fixtures/responses/public/stock-detail-ui-info-snxx.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockInfoDetail for /stock-detail/ui/{code}/info"
```

---

## Task 4: Client — `GetOptionInstrument` + `GetNearestATMOption`

**Files:**
- Create: `internal/client/options.go`
- Create: `internal/client/options_test.go`
- Create fixtures:
  - `fixtures/responses/public/stock-info-opt-sndk-call.json`
  - `fixtures/responses/public/option-default-chart-option-sndk.json`
- Modify: `fixtures/responses/public/manifest.json`

- [ ] **Step 1: Write fixtures**

`fixtures/responses/public/stock-info-opt-sndk-call.json` (trimmed version of the real OPT capture):

```json
{
  "result": {
    "code": "OPT_SNDK260515C01395000_20260506",
    "guid": "OPT_SNDK260515C01395000_20260506",
    "symbol": "SNDK260515C01395000",
    "name": "샌디스크 $1,395 콜",
    "market": {"code": "NSQ", "displayName": "NASDAQ"},
    "currency": "USD",
    "optionPennyPilotPriceSupported": true,
    "optionInstrument": {
      "marketCode": "CBOE",
      "rootSymbol": "SNDK",
      "name": "샌디스크 $1,395 콜",
      "underlyingSymbol": "SNDK",
      "underlyingGuid": "NAS0250224006",
      "underlyingName": "샌디스크",
      "maturityDate": "2026-05-15",
      "maturityDateTime": "2026-05-16T03:50:00.000+09:00",
      "putCall": "CALL",
      "strikePrice": 1395.0,
      "basePrice": 31.8,
      "last": 31.8,
      "bid": 30.1,
      "ask": 33.9,
      "mid": 32.0,
      "contractUnit": 100.0,
      "openInterest": 82,
      "halted": false,
      "tradingSuspended": false,
      "buySuspended": false,
      "sellSuspended": false,
      "status": "LISTED",
      "overtime": false,
      "optionLiquidation": {
        "liquidationDateTime": "2026-05-16T03:50:00.000+09:00",
        "displayLiquidationDateTime": "2시간 46분 후 거래 종료"
      }
    }
  }
}
```

`fixtures/responses/public/option-default-chart-option-sndk.json`:

```json
{"result":"OPT_SNDK260515C01395000_20260506"}
```

Append to `manifest.json`:

```json
{
  "file": "stock-info-opt-sndk-call.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/stock-infos/OPT_SNDK260515C01395000_20260506",
  "method": "GET"
},
{
  "file": "option-default-chart-option-sndk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/option-infos/default-chart-option?underlyingGuid=NAS0250224006",
  "method": "GET"
}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/client/options_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetOptionInstrumentFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-info-opt-sndk-call.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/OPT_SNDK260515C01395000_20260506":
			w.Write(body)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	inst, err := c.GetOptionInstrument(context.Background(), "OPT_SNDK260515C01395000_20260506")
	if err != nil {
		t.Fatalf("GetOptionInstrument error: %v", err)
	}
	if inst.UnderlyingSymbol != "SNDK" || inst.StrikePrice != 1395.0 || inst.PutCall != "CALL" {
		t.Fatalf("unexpected option instrument: %+v", inst)
	}
	if inst.OpenInterest != 82 || inst.ContractUnit != 100.0 {
		t.Fatalf("unexpected OI/contract: %+v", inst)
	}
	if inst.LiquidationDisplay != "2시간 46분 후 거래 종료" {
		t.Fatalf("unexpected liquidation display: %s", inst.LiquidationDisplay)
	}
	if !inst.PennyPilot {
		t.Fatalf("expected PennyPilot true")
	}
}

func TestGetNearestATMOptionFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "option-default-chart-option-sndk.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006":
			http.ServeFile(w, r, filepath.Join(root, "stock-info.json"))
		case "/api/v1/option-infos/default-chart-option":
			if r.URL.Query().Get("underlyingGuid") != "NAS0250224006" {
				t.Fatalf("unexpected underlyingGuid: %s", r.URL.Query().Get("underlyingGuid"))
			}
			w.Write(body)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	code, err := c.GetNearestATMOption(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetNearestATMOption error: %v", err)
	}
	if code != "OPT_SNDK260515C01395000_20260506" {
		t.Fatalf("unexpected nearest ATM code: %s", code)
	}
}
```

- [ ] **Step 3: Run tests (must fail)**

`go test ./internal/client/ -run "TestGetOptionInstrument|TestGetNearestATMOption" -v`
Expected: `undefined: Client.GetOptionInstrument` / `undefined: Client.GetNearestATMOption`.

- [ ] **Step 4: Implement**

Create `internal/client/options.go`:

```go
package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type optionStockInfoEnvelope struct {
	Result struct {
		Code                         string `json:"code"`
		OptionPennyPilotPriceSupport bool   `json:"optionPennyPilotPriceSupported"`
		OptionInstrument             struct {
			MarketCode       string  `json:"marketCode"`
			RootSymbol       string  `json:"rootSymbol"`
			Name             string  `json:"name"`
			FullName         string  `json:"fullName"`
			CompleteName     string  `json:"completeName"`
			UnderlyingSymbol string  `json:"underlyingSymbol"`
			UnderlyingGuid   string  `json:"underlyingGuid"`
			UnderlyingName   string  `json:"underlyingName"`
			MaturityDate     string  `json:"maturityDate"`
			MaturityDateTime string  `json:"maturityDateTime"`
			PutCall          string  `json:"putCall"`
			StrikePrice      float64 `json:"strikePrice"`
			BasePrice        float64 `json:"basePrice"`
			Last             float64 `json:"last"`
			Bid              float64 `json:"bid"`
			Ask              float64 `json:"ask"`
			Mid              float64 `json:"mid"`
			ContractUnit     float64 `json:"contractUnit"`
			OpenInterest     int     `json:"openInterest"`
			Halted           bool    `json:"halted"`
			TradingSuspended bool    `json:"tradingSuspended"`
			BuySuspended     bool    `json:"buySuspended"`
			SellSuspended    bool    `json:"sellSuspended"`
			Status           string  `json:"status"`
			Overtime         bool    `json:"overtime"`
			OptionLiquidation struct {
				LiquidationDateTime        string `json:"liquidationDateTime"`
				DisplayLiquidationDateTime string `json:"displayLiquidationDateTime"`
			} `json:"optionLiquidation"`
		} `json:"optionInstrument"`
	} `json:"result"`
}

// GetOptionInstrument decodes the option-specific metadata block from
// /api/v2/stock-infos/{OPT_…}. Caller must provide a full OPT_ productCode
// (no auto-resolve from "SNDK" — for now use GetNearestATMOption to discover
// or pass the OPT_ directly).
func (c *Client) GetOptionInstrument(ctx context.Context, productCode string) (domain.OptionInstrument, error) {
	if !strings.HasPrefix(productCode, "OPT_") {
		return domain.OptionInstrument{}, fmt.Errorf("GetOptionInstrument: productCode must start with OPT_ (got %q)", productCode)
	}
	endpoint := fmt.Sprintf("%s/api/v2/stock-infos/%s", c.infoBaseURL, productCode)
	var envelope optionStockInfoEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.OptionInstrument{}, err
	}
	oi := envelope.Result.OptionInstrument
	return domain.OptionInstrument{
		ProductCode:         productCode,
		MarketCode:          oi.MarketCode,
		RootSymbol:          oi.RootSymbol,
		Name:                oi.Name,
		FullName:            oi.FullName,
		CompleteName:        oi.CompleteName,
		UnderlyingSymbol:    oi.UnderlyingSymbol,
		UnderlyingGuid:      oi.UnderlyingGuid,
		UnderlyingName:      oi.UnderlyingName,
		MaturityDate:        oi.MaturityDate,
		MaturityDateTime:    oi.MaturityDateTime,
		PutCall:             oi.PutCall,
		StrikePrice:         oi.StrikePrice,
		BasePrice:           oi.BasePrice,
		Last:                oi.Last,
		Bid:                 oi.Bid,
		Ask:                 oi.Ask,
		Mid:                 oi.Mid,
		ContractUnit:        oi.ContractUnit,
		OpenInterest:        oi.OpenInterest,
		Halted:              oi.Halted,
		TradingSuspended:    oi.TradingSuspended,
		BuySuspended:        oi.BuySuspended,
		SellSuspended:       oi.SellSuspended,
		Status:              oi.Status,
		Overtime:            oi.Overtime,
		LiquidationDisplay:  oi.OptionLiquidation.DisplayLiquidationDateTime,
		LiquidationDateTime: oi.OptionLiquidation.LiquidationDateTime,
		PennyPilot:          envelope.Result.OptionPennyPilotPriceSupport,
		FetchedAt:           time.Now().UTC(),
	}, nil
}

// GetNearestATMOption returns the OPT_ productCode for the nearest expiry's
// at-the-money option for an underlying (Toss's default chart option). The
// underlying may be a symbol ("SNDK") or a productCode ("NAS0250224006").
func (c *Client) GetNearestATMOption(ctx context.Context, underlying string) (string, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-infos/default-chart-option", c.infoBaseURL))
	if err != nil {
		return "", err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	endpoint.RawQuery = q.Encode()

	var envelope struct {
		Result string `json:"result"`
	}
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return "", err
	}
	if envelope.Result == "" {
		return "", fmt.Errorf("GetNearestATMOption: empty result for %s", productCode)
	}
	return envelope.Result, nil
}
```

- [ ] **Step 5: Run tests (must pass)**

`go test ./internal/client/ -run "TestGetOptionInstrument|TestGetNearestATMOption" -v` → PASS.

- [ ] **Step 6: Regression**

`go test ./internal/client/` → ok.

- [ ] **Step 7: Commit**

```bash
git add internal/client/options.go internal/client/options_test.go fixtures/responses/public/stock-info-opt-sndk-call.json fixtures/responses/public/option-default-chart-option-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetOptionInstrument + GetNearestATMOption"
```

---

## Task 5: Output — `WriteStockInfoDetail` + `WriteOptionInstrument`

**Files:**
- Create: `internal/output/stockinfo.go`
- Create: `internal/output/stockinfo_test.go`
- Create: `internal/output/options.go`
- Create: `internal/output/options_test.go`

- [ ] **Step 1: Write failing tests**

`internal/output/stockinfo_test.go`:

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

func TestWriteStockInfoDetailTable(t *testing.T) {
	detail := domain.StockInfoDetail{
		ProductCode: "NAS0250224006",
		Sections: []domain.StockInfoSection{
			{Type: "OVERVIEW", Data: json.RawMessage(`{"name":"샌디스크","description":"메모리 회사"}`)},
			{Type: "INDICATORS", Data: json.RawMessage(`{"values":[{"label":"PER","displayValue":"45.4배"}]}`)},
		},
		FetchedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteStockInfoDetail(&buf, FormatTable, detail); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "OVERVIEW") || !strings.Contains(s, "INDICATORS") {
		t.Fatalf("expected section type names in table: %q", s)
	}
}

func TestWriteStockInfoDetailJSON(t *testing.T) {
	detail := domain.StockInfoDetail{
		ProductCode: "NAS0250224006",
		Sections:    []domain.StockInfoSection{{Type: "OVERVIEW", Data: json.RawMessage(`{"name":"샌디스크"}`)}},
	}
	var buf bytes.Buffer
	if err := WriteStockInfoDetail(&buf, FormatJSON, detail); err != nil {
		t.Fatalf("error: %v", err)
	}
	var roundtrip domain.StockInfoDetail
	if err := json.Unmarshal(buf.Bytes(), &roundtrip); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if roundtrip.ProductCode != "NAS0250224006" {
		t.Fatalf("unexpected roundtrip: %+v", roundtrip)
	}
}
```

`internal/output/options_test.go`:

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

func TestWriteOptionInstrumentTable(t *testing.T) {
	inst := domain.OptionInstrument{
		ProductCode:        "OPT_SNDK260515C01395000_20260506",
		UnderlyingSymbol:   "SNDK",
		PutCall:            "CALL",
		StrikePrice:        1395,
		Last:               31.8,
		Bid:                30.1,
		Ask:                33.9,
		OpenInterest:       82,
		MaturityDate:       "2026-05-15",
		LiquidationDisplay: "2시간 46분 후 거래 종료",
		FetchedAt:          time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteOptionInstrument(&buf, FormatTable, inst); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, needle := range []string{"SNDK", "CALL", "1395", "2026-05-15", "거래 종료"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in table output:\n%s", needle, out)
		}
	}
}

func TestWriteOptionInstrumentJSON(t *testing.T) {
	inst := domain.OptionInstrument{ProductCode: "OPT_X", UnderlyingSymbol: "X", StrikePrice: 10}
	var buf bytes.Buffer
	if err := WriteOptionInstrument(&buf, FormatJSON, inst); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OptionInstrument
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.StrikePrice != 10 {
		t.Fatalf("unexpected JSON roundtrip: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run tests (must fail)**

`go test ./internal/output/ -run "WriteStockInfoDetail|WriteOptionInstrument" -v`
Expected: undefined names.

- [ ] **Step 3: Implement**

`internal/output/stockinfo.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockInfoDetail renders the deep-tab payload. Table mode emits one row
// per section (type + byte count); JSON mode emits the full structure including
// raw section bodies; CSV mode emits (section_type, data_bytes) pairs.
func WriteStockInfoDetail(w io.Writer, format Format, detail domain.StockInfoDetail) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(detail)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "section_type,data_bytes"); err != nil {
			return err
		}
		for _, s := range detail.Sections {
			if _, err := fmt.Fprintf(w, "%s,%d\n", s.Type, len(s.Data)); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s  sections=%d  fetched=%s\n",
			detail.ProductCode, len(detail.Sections), detail.FetchedAt.Format("2006-01-02 15:04:05Z07:00")); err != nil {
			return err
		}
		headers := []string{"TYPE", "BYTES"}
		rows := make([][]string, 0, len(detail.Sections))
		for _, s := range detail.Sections {
			rows = append(rows, []string{s.Type, fmt.Sprintf("%d", len(s.Data))})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

`internal/output/options.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOptionInstrument renders a single option's metadata block.
func WriteOptionInstrument(w io.Writer, format Format, inst domain.OptionInstrument) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(inst)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "product_code,underlying,put_call,strike,maturity,last,bid,ask,mid,open_interest,status"); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%s\n",
			inst.ProductCode, inst.UnderlyingSymbol, inst.PutCall,
			formatFloat(inst.StrikePrice), inst.MaturityDate,
			formatFloat(inst.Last), formatFloat(inst.Bid), formatFloat(inst.Ask), formatFloat(inst.Mid),
			inst.OpenInterest, inst.Status)
		return err
	case FormatTable:
		if _, err := fmt.Fprintf(w,
			"%s  %s %s  strike=%s  maturity=%s  status=%s\n",
			inst.ProductCode, inst.UnderlyingSymbol, inst.PutCall,
			formatFloat(inst.StrikePrice), inst.MaturityDate, inst.Status); err != nil {
			return err
		}
		if inst.LiquidationDisplay != "" {
			if _, err := fmt.Fprintf(w, "  %s\n", inst.LiquidationDisplay); err != nil {
				return err
			}
		}
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"last", formatFloat(inst.Last)},
			{"bid", formatFloat(inst.Bid)},
			{"ask", formatFloat(inst.Ask)},
			{"mid", formatFloat(inst.Mid)},
			{"base", formatFloat(inst.BasePrice)},
			{"open_interest", fmt.Sprintf("%d", inst.OpenInterest)},
			{"contract_unit", formatFloat(inst.ContractUnit)},
			{"halted", fmt.Sprintf("%v", inst.Halted)},
			{"trading_suspended", fmt.Sprintf("%v", inst.TradingSuspended)},
			{"overtime", fmt.Sprintf("%v", inst.Overtime)},
			{"penny_pilot", fmt.Sprintf("%v", inst.PennyPilot)},
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 4: Run tests (must pass)**

`go test ./internal/output/` → ok.

- [ ] **Step 5: Commit**

```bash
git add internal/output/stockinfo.go internal/output/stockinfo_test.go internal/output/options.go internal/output/options_test.go
git commit -m "feat(output): WriteStockInfoDetail + WriteOptionInstrument"
```

---

## Task 6: CLI — `tossctl stock info <sym>`

**Files:**
- Create: `cmd/tossctl/stock.go`
- Modify: `cmd/tossctl/root.go` (register the new command)

- [ ] **Step 1: Create `cmd/tossctl/stock.go`**

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newStockCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stock",
		Short: "Stock info (deep tab)",
	}

	infoCmd := &cobra.Command{
		Use:   "info <symbol>",
		Short: "Fetch the 종목정보 deep-tab payload (OVERVIEW / FINANCES / ANALYST / 매출 구성 / 동종업계 / ...)",
		Long: `Fetch the deep stock-info payload behind the Toss 종목정보 tab.

Returns ~13 sections (OVERVIEW, INDICATORS, NEWS, ANNOUNCEMENT, FINANCES,
EARNINGS_AND_CONSENSUS, ANALYST_OPINION, COMPOSITION_OF_REVENUE,
VALUATION_METRICS, STABILITY, TOP_TIER_TREND, STOCK_INFO_SIGNAL, PRICE).

Table mode prints a one-line summary per section. Use --output json for the
full structured payload (consumer-friendly for LLM pipelines).

Examples:
  tossctl stock info SNDK
  tossctl stock info NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			detail, err := app.client.GetStockInfoDetail(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteStockInfoDetail(cmd.OutOrStdout(), app.format, detail)
		},
	}

	cmd.AddCommand(infoCmd)
	return cmd
}
```

- [ ] **Step 2: Register in `cmd/tossctl/root.go`**

In the `cmd.AddCommand(...)` block, add `newStockCmd(opts)` alongside the existing entries (alphabetical placement near `newSignalsCmd` is fine).

- [ ] **Step 3: Build**

`go build ./...` → success.

- [ ] **Step 4: Help check**

`go run ./cmd/tossctl stock info --help` → should show the long help text.

- [ ] **Step 5: Run cmd tests**

`go test ./cmd/tossctl/` → ok.

- [ ] **Step 6: Commit**

```bash
git add cmd/tossctl/stock.go cmd/tossctl/root.go
git commit -m "feat(cli): add tossctl stock info"
```

---

## Task 7: CLI — `tossctl options info` + `tossctl options nearest-atm`

**Files:**
- Create: `cmd/tossctl/options.go`
- Modify: `cmd/tossctl/root.go`

- [ ] **Step 1: Create `cmd/tossctl/options.go`**

```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newOptionsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "options",
		Short: "US option contracts (read-only)",
		Long: `Read-only access to US option contracts.

Toss treats each option as a first-class product with an OCC-style productCode:

  OPT_<rootSymbol><YYMMDD><C|P><strikeMillis8>_<listDateYYYYMMDD>

Example: SNDK 2026-05-15 $1,395 Call = OPT_SNDK260515C01395000_20260506.

Once you have an OPT_ productCode, the existing tossctl commands all work:
  tossctl quote   get   OPT_...
  tossctl quotes  book  OPT_...
  tossctl quotes  ticks OPT_...   [--follow]
  tossctl chart   get   OPT_... --tf 15m   (auto-routes to us-o chart family)

Use 'options nearest-atm <symbol>' to discover the OPT_ code for an
underlying's nearest-expiry at-the-money option. The full strike+expiry
chain enumeration endpoint has not yet been reverse-engineered.`,
	}

	infoCmd := &cobra.Command{
		Use:   "info <OPT_…>",
		Short: "Show option-specific metadata (strike, expiry, OI, bid/ask, halt status)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			inst, err := app.client.GetOptionInstrument(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOptionInstrument(cmd.OutOrStdout(), app.format, inst)
		},
	}

	atmCmd := &cobra.Command{
		Use:   "nearest-atm <underlying>",
		Short: "Return the OPT_ productCode for the underlying's nearest-expiry ATM option",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			code, err := app.client.GetNearestATMOption(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), code)
			return err
		},
	}

	cmd.AddCommand(infoCmd, atmCmd)
	return cmd
}
```

- [ ] **Step 2: Register in `cmd/tossctl/root.go`**

Add `newOptionsCmd(opts)` to the `cmd.AddCommand(...)` block.

- [ ] **Step 3: Build + help**

```
go build ./...
go run ./cmd/tossctl options --help
go run ./cmd/tossctl options info --help
go run ./cmd/tossctl options nearest-atm --help
```

- [ ] **Step 4: Run cmd tests**

`go test ./cmd/tossctl/` → ok.

- [ ] **Step 5: Run full suite**

`go test ./... -count=1` → all packages pass.

- [ ] **Step 6: Commit**

```bash
git add cmd/tossctl/options.go cmd/tossctl/root.go
git commit -m "feat(cli): add tossctl options info + nearest-atm"
```

---

## Task 8: Docs — CHANGELOG + rpc-catalog

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/reverse-engineering/rpc-catalog.md`

- [ ] **Step 1: Add to CHANGELOG.md Unreleased**

Under `### Added`:

```markdown
- `tossctl stock info <symbol>` — full 종목정보 deep tab (OVERVIEW / FINANCES / EARNINGS / ANALYST_OPINION / VALUATION_METRICS / COMPOSITION_OF_REVENUE / etc.) via `/api/v1/stock-detail/ui/{code}/info`.
- `tossctl options info <OPT_…>` — single-option metadata (strike, expiry, bid/ask/mid, OI, contract unit, liquidation countdown, halt/suspend flags) via `/api/v2/stock-infos/{OPT_…}` + `optionInstrument` decoding.
- `tossctl options nearest-atm <underlying>` — nearest-expiry ATM option's OPT_ productCode via `/api/v1/option-infos/default-chart-option`.
- `OPT_…` productCodes route through `chartProductPrefix` to the `us-o` chart family, so `tossctl chart get OPT_... --tf 15m` works unchanged.
```

Under `### Changed`:

```markdown
- `chartProductPrefix` now recognizes `OPT_` productCodes and returns `us-o`.
```

- [ ] **Step 2: Update `docs/reverse-engineering/rpc-catalog.md`**

In the **Quote and Symbol Detail** table, add rows:

```markdown
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/{code}/info` | **full 종목정보 deep tab** | `.result.sections[]` typed `OVERVIEW`/`INDICATORS`/`NEWS`/`ANNOUNCEMENT`/`FINANCES`/`EARNINGS_AND_CONSENSUS`/`ANALYST_OPINION`/`VALUATION_METRICS`/`COMPOSITION_OF_REVENUE`/`STABILITY`/`TOP_TIER_TREND`/`STOCK_INFO_SIGNAL`/`PRICE` | `stock info <sym>` (since v0.6.x) | returns `{"result":null}` for products without coverage. Sections are independent panels; consumers route by `section.type`. See [`stock-info-deep-tab.md`](stock-info-deep-tab.md) |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/option-infos/default-chart-option?underlyingGuid={code}` | nearest-expiry ATM option | `.result` string OCC-style `OPT_<root><YYMMDD><C|P><strikeMillis8>_<listDate>` | `options nearest-atm <sym>` (since v0.6.x) | gateway to the option product family |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/{OPT_…}` | single-option metadata (strike/expiry/bid/ask/mid/OI/contract unit/halt) | `.result.optionInstrument.{rootSymbol, underlyingSymbol, underlyingGuid, maturityDate, putCall, strikePrice, basePrice, last, bid, ask, mid, contractUnit, openInterest, halted, tradingSuspended, status, optionLiquidation.{liquidationDateTime, displayLiquidationDateTime}}` + parent `optionPennyPilotPriceSupported` | `options info <OPT_…>` (since v0.6.x) | same shape as stock `stock-infos` for the wrapper fields; option block is the value-add |
```

(Existing `/api/v3/stock-prices/details`, `/quotes`, `/ticks`, `/c-chart` rows already note that they work for any productCode; add a small note to the chart row: "use `us-o` product family for `OPT_…` codes — `chartProductPrefix` handles this automatically".)

- [ ] **Step 3: Regression suite**

```
go vet ./...
go test ./... -count=1
```

Both must be green.

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md docs/reverse-engineering/rpc-catalog.md
git commit -m "docs: note stock info + options additions"
```

---

## Self-review notes

- **No options chain enumeration in PR6.** Documented in `docs/reverse-engineering/options.md` as a gap requiring browser capture. Implementing what's deterministically working today rather than guessing the chain endpoint.
- **Sections are kept as `json.RawMessage`** in `StockInfoSection.Data` — the 13 panel schemas are independent and frequently change; pinning them all into typed Go structs would force a coupled domain release for every Toss UI shift. Consumers can decode per section type as needed.
- **`GetOptionInstrument` rejects non-`OPT_` codes** at the boundary; auto-resolving `"SNDK"` to an option doesn't make sense (which expiry? which strike?) — caller must explicitly use `nearest-atm` or pass an OPT_ code.
- **Chart prefix fix is one line + a regression test;** keep it that small so it's easy to backport to other product-family additions later (e.g., bonds).
- **Existing `--follow` works on OPT_ codes** through `quote get` / `quotes book` / `quotes ticks` / `chart get` once Task 1 lands — no separate `options --follow` commands needed.

---

## Execution handoff

Plan saved. Execute with **superpowers:subagent-driven-development** — fresh subagent per task, spec compliance review then code quality review.

After Task 8, merge `--no-ff` back into `feat/order-page-integration`, push to fork only.
