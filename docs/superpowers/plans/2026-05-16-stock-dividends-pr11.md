# PR11 — Stock dividends (summary + recent payouts + full history)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `tossctl stock dividends <sym>` covering three GET endpoints:
- `/api/v1/stock-infos/{code}/dividends/yield-ratio/histories` — TTM summary card (yield, frequency, dps, growth)
- `/api/v1/stock-infos/dividend/{code}/years` — recent payouts (default 3-year range) with totals
- `/api/v1/stock-infos/dividend/{code}/summary` — full historical payout list

All three use `{stockCode}` (productCode). All three return populated data for dividend-paying stocks (e.g. AAPL) and gracefully empty data for non-payers (e.g. SNDK).

**Architecture:** One client method `GetStockDividends` makes 3 sequential `getJSON` calls and stitches into `domain.StockDividends`. One output writer renders summary + recent payouts in table mode; full history is JSON-only by default (large), surfaced via `--all-history` flag in table mode.

**Tech Stack:** Go, cobra, getJSON, encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `f241d26`.

---

## File Structure

**New:**
- `internal/client/dividends.go` + `_test.go` — `GetStockDividends`
- `internal/output/dividends.go` + `_test.go` — `WriteStockDividends`
- `fixtures/responses/public/stock-dividends-aapl.json` — populated AAPL fixture (bundles all 3)
- `fixtures/responses/public/stock-dividends-sndk.json` — empty SNDK fixture (bundles all 3)

**Modified:**
- `internal/domain/models.go` — append `StockDividends`, `DividendYieldCard`, `DividendYearsPayouts`, `DividendPayout`, `DividendHistory` types
- `cmd/tossctl/stock.go` — register `dividendsCmd` (with `--all-history` flag)
- `fixtures/responses/public/manifest.json` — append both fixture entries
- `CHANGELOG.md` — add line under `## Unreleased`

---

## Task 1: Domain types + client + fixtures + tests

### Step 1: Save AAPL fixture (populated)

Create `fixtures/responses/public/stock-dividends-aapl.json`. Trim to the last 4 payouts in each list to keep fixture small:

```json
{
  "yieldCard": {
    "result": {
      "dividendCount": 4,
      "dividendMonths": [2, 5, 8, 11],
      "dividendCash": 1.03,
      "dividendCashKrw": 1536,
      "dividendCashJpy": 163,
      "dividendYieldRatio": 0.0035,
      "ttmDividendYieldRatio": 0.0035,
      "ttmDividendMonths": ["2025-05", "2025-08", "2025-11", "2026-02"],
      "ttmDps": 1.04,
      "ttmDpsKrw": 1551,
      "ttmDpsJpy": 164,
      "ttmDividendTotalCount": 4,
      "dividendGrowthRatio": 0.0179428877941688,
      "currency": "USD"
    }
  },
  "years": {
    "result": {
      "startDate": "2023-01-01",
      "selectedRange": {"code": 3, "displayName": "3년"},
      "histories": [
        {"exDate": "2025-08-11", "paymentDate": "2025-08-15", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0011, "ttmYieldRatio": 0.0044},
        {"exDate": "2025-11-10", "paymentDate": "2025-11-14", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0011, "ttmYieldRatio": 0.0042},
        {"exDate": "2026-02-09", "paymentDate": "2026-02-13", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0009, "ttmYieldRatio": 0.0039},
        {"exDate": "2026-05-09", "paymentDate": "2026-05-15", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0009, "ttmYieldRatio": 0.0035}
      ],
      "totalCash": 1.04,
      "totalCashKrw": 1551,
      "totalCashJpy": 164
    }
  },
  "history": {
    "result": [
      {"exDate": "2025-08-11", "paymentDate": "2025-08-15", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0011, "ttmYieldRatio": 0.0044},
      {"exDate": "2025-11-10", "paymentDate": "2025-11-14", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0011, "ttmYieldRatio": 0.0042},
      {"exDate": "2026-02-09", "paymentDate": "2026-02-13", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0009, "ttmYieldRatio": 0.0039},
      {"exDate": "2026-05-09", "paymentDate": "2026-05-15", "currency": "USD", "ratio": 0.0, "cash": 0.26, "cashKrw": 388, "cashJpy": 41, "yieldRatio": 0.0009, "ttmYieldRatio": 0.0035}
    ]
  }
}
```

### Step 2: Save SNDK fixture (empty)

Create `fixtures/responses/public/stock-dividends-sndk.json`:

```json
{
  "yieldCard": {
    "result": {
      "dividendCount": 0,
      "dividendMonths": [],
      "dividendCash": 0,
      "dividendCashKrw": null,
      "dividendCashJpy": null,
      "dividendYieldRatio": 0,
      "ttmDividendYieldRatio": 0,
      "ttmDividendMonths": [],
      "ttmDps": 0.0,
      "ttmDpsKrw": null,
      "ttmDpsJpy": null,
      "ttmDividendTotalCount": 0,
      "dividendGrowthRatio": null,
      "currency": null
    }
  },
  "years": {
    "result": {
      "startDate": "2023-01-01",
      "selectedRange": {"code": 3, "displayName": "3년"},
      "histories": [],
      "totalCash": 0,
      "totalCashKrw": 0,
      "totalCashJpy": 0
    }
  },
  "history": {
    "result": []
  }
}
```

### Step 3: Update manifest

Append (with comma):

```json
{
  "file": "stock-dividends-aapl.json",
  "url": "(combined yield-card+years+history fixture for testing only)",
  "method": "GET"
},
{
  "file": "stock-dividends-sndk.json",
  "url": "(combined yield-card+years+history fixture — empty case)",
  "method": "GET"
}
```

### Step 4: Add domain types

Append to `internal/domain/models.go`:

```go
type StockDividends struct {
	ProductCode  string                 `json:"product_code"`
	Summary      DividendYieldCard      `json:"summary"`
	RecentYears  DividendYearsPayouts   `json:"recent_years"`
	FullHistory  []DividendPayout       `json:"full_history"`
	FetchedAt    time.Time              `json:"fetched_at"`
}

// DividendYieldCard is the TTM summary card from
// /api/v1/stock-infos/{code}/dividends/yield-ratio/histories.
type DividendYieldCard struct {
	DividendCount         int      `json:"dividend_count"`
	DividendMonths        []int    `json:"dividend_months"`
	DividendCash          float64  `json:"dividend_cash"`
	DividendCashKrw       *float64 `json:"dividend_cash_krw"`
	DividendYieldRatio    float64  `json:"dividend_yield_ratio"`
	TTMDividendYieldRatio float64  `json:"ttm_dividend_yield_ratio"`
	TTMDividendMonths     []string `json:"ttm_dividend_months"`
	TTMDps                float64  `json:"ttm_dps"`
	TTMDpsKrw             *float64 `json:"ttm_dps_krw"`
	TTMDividendTotalCount int      `json:"ttm_dividend_total_count"`
	DividendGrowthRatio   *float64 `json:"dividend_growth_ratio"`
	Currency              string   `json:"currency"`
}

// DividendYearsPayouts is the recent-range payout list from
// /api/v1/stock-infos/dividend/{code}/years.
type DividendYearsPayouts struct {
	StartDate    string            `json:"start_date"`
	RangeLabel   string            `json:"range_label"`
	Payouts      []DividendPayout  `json:"payouts"`
	TotalCash    float64           `json:"total_cash"`
	TotalCashKrw float64           `json:"total_cash_krw"`
}

// DividendPayout is one historical payout record (used by both `years` and
// `summary` endpoints).
type DividendPayout struct {
	ExDate        string  `json:"ex_date"`
	PaymentDate   string  `json:"payment_date"`
	Currency      string  `json:"currency"`
	Cash          float64 `json:"cash"`
	CashKrw       float64 `json:"cash_krw"`
	YieldRatio    float64 `json:"yield_ratio"`
	TTMYieldRatio float64 `json:"ttm_yield_ratio"`
}
```

### Step 5: Write failing client test

Create `internal/client/dividends_test.go`:

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

func TestGetStockDividendsAAPL(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		YieldCard json.RawMessage `json:"yieldCard"`
		Years     json.RawMessage `json:"years"`
		History   json.RawMessage `json:"history"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-dividends-aapl.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-infos/US19801212001/dividends/yield-ratio/histories":
			w.Write(bundle.YieldCard)
		case "/api/v1/stock-infos/dividend/US19801212001/years":
			w.Write(bundle.Years)
		case "/api/v1/stock-infos/dividend/US19801212001/summary":
			w.Write(bundle.History)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	div, err := c.GetStockDividends(context.Background(), "US19801212001")
	if err != nil {
		t.Fatalf("GetStockDividends error: %v", err)
	}
	if div.Summary.Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", div.Summary.Currency)
	}
	if div.Summary.TTMDividendTotalCount != 4 {
		t.Fatalf("expected 4 TTM count, got %d", div.Summary.TTMDividendTotalCount)
	}
	if div.Summary.DividendGrowthRatio == nil || *div.Summary.DividendGrowthRatio < 0.01 {
		t.Fatalf("expected positive growth, got %v", div.Summary.DividendGrowthRatio)
	}
	if len(div.RecentYears.Payouts) != 4 {
		t.Fatalf("expected 4 recent payouts, got %d", len(div.RecentYears.Payouts))
	}
	if div.RecentYears.RangeLabel != "3년" {
		t.Fatalf("expected 3년 range, got %q", div.RecentYears.RangeLabel)
	}
	if div.RecentYears.TotalCash != 1.04 {
		t.Fatalf("expected total 1.04, got %v", div.RecentYears.TotalCash)
	}
	if len(div.FullHistory) != 4 {
		t.Fatalf("expected 4 history rows, got %d", len(div.FullHistory))
	}
	if div.RecentYears.Payouts[3].ExDate != "2026-05-09" {
		t.Fatalf("unexpected last exDate: %q", div.RecentYears.Payouts[3].ExDate)
	}
}

func TestGetStockDividendsSNDKEmpty(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		YieldCard json.RawMessage `json:"yieldCard"`
		Years     json.RawMessage `json:"years"`
		History   json.RawMessage `json:"history"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-dividends-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-infos/NAS0250224006/dividends/yield-ratio/histories":
			w.Write(bundle.YieldCard)
		case "/api/v1/stock-infos/dividend/NAS0250224006/years":
			w.Write(bundle.Years)
		case "/api/v1/stock-infos/dividend/NAS0250224006/summary":
			w.Write(bundle.History)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	div, err := c.GetStockDividends(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockDividends error: %v", err)
	}
	if div.Summary.DividendCount != 0 {
		t.Fatalf("expected 0 dividend count, got %d", div.Summary.DividendCount)
	}
	if len(div.RecentYears.Payouts) != 0 {
		t.Fatalf("expected 0 recent payouts, got %d", len(div.RecentYears.Payouts))
	}
	if len(div.FullHistory) != 0 {
		t.Fatalf("expected 0 history rows, got %d", len(div.FullHistory))
	}
	if div.Summary.Currency != "" {
		t.Fatalf("expected empty currency, got %q", div.Summary.Currency)
	}
}
```

### Step 6: Run test (FAIL)

Run: `go test ./internal/client/ -run TestGetStockDividends -v`
Expected: FAIL with "undefined: c.GetStockDividends".

### Step 7: Implement client method

Create `internal/client/dividends.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type dividendYieldCardEnvelope struct {
	Result struct {
		DividendCount         int      `json:"dividendCount"`
		DividendMonths        []int    `json:"dividendMonths"`
		DividendCash          float64  `json:"dividendCash"`
		DividendCashKrw       *float64 `json:"dividendCashKrw"`
		DividendYieldRatio    float64  `json:"dividendYieldRatio"`
		TTMDividendYieldRatio float64  `json:"ttmDividendYieldRatio"`
		TTMDividendMonths     []string `json:"ttmDividendMonths"`
		TTMDps                float64  `json:"ttmDps"`
		TTMDpsKrw             *float64 `json:"ttmDpsKrw"`
		TTMDividendTotalCount int      `json:"ttmDividendTotalCount"`
		DividendGrowthRatio   *float64 `json:"dividendGrowthRatio"`
		Currency              string   `json:"currency"`
	} `json:"result"`
}

type dividendYearsEnvelope struct {
	Result struct {
		StartDate     string `json:"startDate"`
		SelectedRange struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedRange"`
		Histories    []dividendPayoutWire `json:"histories"`
		TotalCash    float64              `json:"totalCash"`
		TotalCashKrw float64              `json:"totalCashKrw"`
	} `json:"result"`
}

type dividendHistoryEnvelope struct {
	Result []dividendPayoutWire `json:"result"`
}

type dividendPayoutWire struct {
	ExDate        string  `json:"exDate"`
	PaymentDate   string  `json:"paymentDate"`
	Currency      string  `json:"currency"`
	Cash          float64 `json:"cash"`
	CashKrw       float64 `json:"cashKrw"`
	YieldRatio    float64 `json:"yieldRatio"`
	TTMYieldRatio float64 `json:"ttmYieldRatio"`
}

// GetStockDividends fetches three dividend-related endpoints (TTM summary card,
// recent-range payouts, full history) and stitches them into a single view.
// All three return empty/zero values gracefully for non-dividend-paying stocks.
func (c *Client) GetStockDividends(ctx context.Context, symbol string) (domain.StockDividends, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockDividends{}, err
	}

	var card dividendYieldCardEnvelope
	cardURL := fmt.Sprintf("%s/api/v1/stock-infos/%s/dividends/yield-ratio/histories", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, cardURL, &card); err != nil {
		return domain.StockDividends{}, err
	}

	var years dividendYearsEnvelope
	yearsURL := fmt.Sprintf("%s/api/v1/stock-infos/dividend/%s/years", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, yearsURL, &years); err != nil {
		return domain.StockDividends{}, err
	}

	var hist dividendHistoryEnvelope
	histURL := fmt.Sprintf("%s/api/v1/stock-infos/dividend/%s/summary", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, histURL, &hist); err != nil {
		return domain.StockDividends{}, err
	}

	return domain.StockDividends{
		ProductCode: productCode,
		Summary: domain.DividendYieldCard{
			DividendCount:         card.Result.DividendCount,
			DividendMonths:        card.Result.DividendMonths,
			DividendCash:          card.Result.DividendCash,
			DividendCashKrw:       card.Result.DividendCashKrw,
			DividendYieldRatio:    card.Result.DividendYieldRatio,
			TTMDividendYieldRatio: card.Result.TTMDividendYieldRatio,
			TTMDividendMonths:     card.Result.TTMDividendMonths,
			TTMDps:                card.Result.TTMDps,
			TTMDpsKrw:             card.Result.TTMDpsKrw,
			TTMDividendTotalCount: card.Result.TTMDividendTotalCount,
			DividendGrowthRatio:   card.Result.DividendGrowthRatio,
			Currency:              card.Result.Currency,
		},
		RecentYears: domain.DividendYearsPayouts{
			StartDate:    years.Result.StartDate,
			RangeLabel:   years.Result.SelectedRange.DisplayName,
			Payouts:      convertPayouts(years.Result.Histories),
			TotalCash:    years.Result.TotalCash,
			TotalCashKrw: years.Result.TotalCashKrw,
		},
		FullHistory: convertPayouts(hist.Result),
		FetchedAt:   time.Now().UTC(),
	}, nil
}

func convertPayouts(src []dividendPayoutWire) []domain.DividendPayout {
	out := make([]domain.DividendPayout, len(src))
	for i, p := range src {
		out[i] = domain.DividendPayout{
			ExDate:        p.ExDate,
			PaymentDate:   p.PaymentDate,
			Currency:      p.Currency,
			Cash:          p.Cash,
			CashKrw:       p.CashKrw,
			YieldRatio:    p.YieldRatio,
			TTMYieldRatio: p.TTMYieldRatio,
		}
	}
	return out
}
```

### Step 8: Run test (PASS)

Run: `go test ./internal/client/ -run TestGetStockDividends -v`
Expected: both subtests PASS.

### Step 9: Commit

```bash
git add internal/domain/models.go internal/client/dividends.go internal/client/dividends_test.go fixtures/responses/public/stock-dividends-aapl.json fixtures/responses/public/stock-dividends-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockDividends (TTM summary + recent payouts + full history)"
```

---

## Task 2: Output writer + cobra + CHANGELOG

### Step 1: Output writer test

Create `internal/output/dividends_test.go`:

```go
package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func dividendsTestFixture() domain.StockDividends {
	growthRatio := 0.0179428877941688
	dpsKrw := 1551.0
	cashKrw := 1536.0
	return domain.StockDividends{
		ProductCode: "US19801212001",
		Summary: domain.DividendYieldCard{
			DividendCount:         4,
			DividendMonths:        []int{2, 5, 8, 11},
			DividendCash:          1.03,
			DividendCashKrw:       &cashKrw,
			DividendYieldRatio:    0.0035,
			TTMDividendYieldRatio: 0.0035,
			TTMDividendMonths:     []string{"2025-05", "2025-08", "2025-11", "2026-02"},
			TTMDps:                1.04,
			TTMDpsKrw:             &dpsKrw,
			TTMDividendTotalCount: 4,
			DividendGrowthRatio:   &growthRatio,
			Currency:              "USD",
		},
		RecentYears: domain.DividendYearsPayouts{
			StartDate: "2023-01-01", RangeLabel: "3년",
			Payouts: []domain.DividendPayout{
				{ExDate: "2026-02-09", PaymentDate: "2026-02-13", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0039},
				{ExDate: "2026-05-09", PaymentDate: "2026-05-15", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0035},
			},
			TotalCash: 1.04, TotalCashKrw: 1551,
		},
		FullHistory: []domain.DividendPayout{
			{ExDate: "2026-02-09", PaymentDate: "2026-02-13", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0039},
			{ExDate: "2026-05-09", PaymentDate: "2026-05-15", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0035},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockDividendsTable(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, false); err != nil {
		t.Fatalf("WriteStockDividends error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"US19801212001", "USD",
		"TTM yield", "0.35%", "TTM dps", "$1.04",
		"+1.79%", // growth ratio
		"3년", "2026-05-09", "$0.26",
		"Total: $1.04",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
	// full-history flag off → should NOT include separate "전체 히스토리" header
	if strings.Contains(out, "전체 히스토리") {
		t.Fatalf("expected no full-history section when flag is false; got:\n%s", out)
	}
}

func TestWriteStockDividendsTableWithAllHistory(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, true); err != nil {
		t.Fatalf("WriteStockDividends error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "전체 히스토리") {
		t.Fatalf("expected full-history section when flag is true; got:\n%s", out)
	}
}

func TestWriteStockDividendsTableEmpty(t *testing.T) {
	t.Parallel()
	div := domain.StockDividends{
		ProductCode: "NAS0250224006",
		Summary:     domain.DividendYieldCard{DividendCount: 0, Currency: ""},
		RecentYears: domain.DividendYearsPayouts{RangeLabel: "3년"},
		FullHistory: nil,
		FetchedAt:   time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, false); err != nil {
		t.Fatalf("WriteStockDividends empty error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No dividends") {
		t.Fatalf("expected 'No dividends' message for empty case; got:\n%s", out)
	}
}

func TestWriteStockDividendsCSV(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatCSV, div, true); err != nil {
		t.Fatalf("WriteStockDividends CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) < 3 {
		t.Fatalf("expected header + multiple rows, got %d", len(rows))
	}
	if rows[0][0] != "section" {
		t.Fatalf("expected first header col 'section', got %q", rows[0][0])
	}
}

func TestWriteStockDividendsJSON(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatJSON, div, true); err != nil {
		t.Fatalf("WriteStockDividends JSON error: %v", err)
	}
	var got domain.StockDividends
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, buf.String())
	}
	if got.ProductCode != div.ProductCode {
		t.Fatalf("product_code roundtrip mismatch: %q vs %q", got.ProductCode, div.ProductCode)
	}
	// Verify snake_case in raw JSON
	if !strings.Contains(buf.String(), `"ttm_dividend_total_count"`) {
		t.Fatalf("expected snake_case ttm_dividend_total_count in JSON")
	}
}
```

### Step 2: Run test (FAIL)

Run: `go test ./internal/output/ -run TestWriteStockDividends -v`

### Step 3: Implement output writer

Create `internal/output/dividends.go`:

```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockDividends renders the dividends snapshot. `allHistory` controls
// whether the full historical payout list is rendered in table mode (it's
// always emitted in JSON; CSV honors the flag).
func WriteStockDividends(w io.Writer, format Format, div domain.StockDividends, allHistory bool) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(div)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "ex_date", "payment_date", "currency", "cash", "cash_krw", "yield_ratio_pct", "ttm_yield_ratio_pct"}); err != nil {
			return err
		}
		// Summary card as a single row
		curr := div.Summary.Currency
		if curr == "" {
			curr = "-"
		}
		if err := cw.Write([]string{
			"summary", "", "", curr,
			strconv.FormatFloat(div.Summary.TTMDps, 'f', 4, 64),
			ptrFloatString(div.Summary.TTMDpsKrw),
			fmt.Sprintf("%.4f", div.Summary.TTMDividendYieldRatio*100),
			"",
		}); err != nil {
			return err
		}
		for _, p := range div.RecentYears.Payouts {
			if err := cw.Write([]string{
				"recent", p.ExDate, p.PaymentDate, p.Currency,
				strconv.FormatFloat(p.Cash, 'f', 4, 64),
				strconv.FormatFloat(p.CashKrw, 'f', 2, 64),
				fmt.Sprintf("%.4f", p.YieldRatio*100),
				fmt.Sprintf("%.4f", p.TTMYieldRatio*100),
			}); err != nil {
				return err
			}
		}
		if allHistory {
			for _, p := range div.FullHistory {
				if err := cw.Write([]string{
					"history", p.ExDate, p.PaymentDate, p.Currency,
					strconv.FormatFloat(p.Cash, 'f', 4, 64),
					strconv.FormatFloat(p.CashKrw, 'f', 2, 64),
					fmt.Sprintf("%.4f", p.YieldRatio*100),
					fmt.Sprintf("%.4f", p.TTMYieldRatio*100),
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — dividends\n", div.ProductCode); err != nil {
			return err
		}

		// Empty case: short-circuit
		if div.Summary.DividendCount == 0 && len(div.RecentYears.Payouts) == 0 {
			_, err := fmt.Fprintln(w, "  No dividends recorded for this stock.")
			return err
		}

		// Summary card
		fmt.Fprintf(w, "\n=== Summary (TTM, %s) ===\n", div.Summary.Currency)
		sumHeaders := []string{"FIELD", "VALUE"}
		sumRows := [][]string{
			{"TTM yield", fmt.Sprintf("%.2f%%", div.Summary.TTMDividendYieldRatio*100)},
			{"TTM dps", fmt.Sprintf("$%.2f", div.Summary.TTMDps)},
			{"TTM dividend count", strconv.Itoa(div.Summary.TTMDividendTotalCount)},
			{"Dividend cadence", fmt.Sprintf("%v", div.Summary.DividendMonths)},
			{"Growth", growthPct(div.Summary.DividendGrowthRatio)},
		}
		if err := renderTable(w, sumHeaders, sumRows); err != nil {
			return err
		}

		// Recent years
		fmt.Fprintf(w, "\n=== Recent payouts (%s, since %s) ===\n", div.RecentYears.RangeLabel, div.RecentYears.StartDate)
		if len(div.RecentYears.Payouts) == 0 {
			fmt.Fprintln(w, "  (no payouts in selected range)")
		} else {
			recHeaders := []string{"EX DATE", "PAYMENT", "CASH (USD)", "CASH (KRW)", "YIELD", "TTM YIELD"}
			recRows := make([][]string, len(div.RecentYears.Payouts))
			for i, p := range div.RecentYears.Payouts {
				recRows[i] = []string{
					p.ExDate, p.PaymentDate,
					fmt.Sprintf("$%.2f", p.Cash),
					formatWithCommas(int64(p.CashKrw)),
					fmt.Sprintf("%.2f%%", p.YieldRatio*100),
					fmt.Sprintf("%.2f%%", p.TTMYieldRatio*100),
				}
			}
			if err := renderTable(w, recHeaders, recRows); err != nil {
				return err
			}
			fmt.Fprintf(w, "Total: $%.2f  (₩%s)\n", div.RecentYears.TotalCash, formatWithCommas(int64(div.RecentYears.TotalCashKrw)))
		}

		// Full history (opt-in)
		if allHistory && len(div.FullHistory) > 0 {
			fmt.Fprintf(w, "\n=== 전체 히스토리 (%d 건) ===\n", len(div.FullHistory))
			hHeaders := []string{"EX DATE", "PAYMENT", "CASH (USD)", "YIELD"}
			hRows := make([][]string, len(div.FullHistory))
			for i, p := range div.FullHistory {
				hRows[i] = []string{
					p.ExDate, p.PaymentDate,
					fmt.Sprintf("$%.2f", p.Cash),
					fmt.Sprintf("%.2f%%", p.YieldRatio*100),
				}
			}
			if err := renderTable(w, hHeaders, hRows); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func growthPct(p *float64) string {
	if p == nil {
		return "—"
	}
	v := *p * 100
	sign := "+"
	if v < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.2f%%", sign, v)
}

func ptrFloatString(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', 2, 64)
}
```

### Step 4: Run test (PASS)

Run: `go test ./internal/output/ -run TestWriteStockDividends -v`

### Step 5: Register cobra subcommand

In `cmd/tossctl/stock.go`, after `financialsCmd`:

```go
var dividendsAllHistory bool
dividendsCmd := &cobra.Command{
    Use:   "dividends <symbol>",
    Short: "Dividend snapshot (TTM yield, recent payouts, optional full history)",
    Long: `Fetch dividend-related endpoints behind the 종목정보 DIVIDEND section.

Stitches three GET endpoints:
  - /api/v1/stock-infos/{code}/dividends/yield-ratio/histories — TTM summary card
  - /api/v1/stock-infos/dividend/{code}/years    — recent-range payouts (3-year default)
  - /api/v1/stock-infos/dividend/{code}/summary  — full historical payout list

For non-paying stocks all three return empty; the table mode prints
"No dividends recorded for this stock." JSON output always includes the
full history; CSV honors --all-history.

Examples:
  tossctl stock dividends AAPL
  tossctl stock dividends AAPL --all-history
  tossctl stock dividends NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        div, err := app.client.GetStockDividends(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockDividends(cmd.OutOrStdout(), app.format, div, dividendsAllHistory)
    },
}
dividendsCmd.Flags().BoolVar(&dividendsAllHistory, "all-history", false, "Include the full historical payout list in table/CSV output")
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd, dividendsCmd)
```

### Step 6: CHANGELOG entry

In `CHANGELOG.md`, after the `stock financials` line, add:

```markdown
- `tossctl stock dividends <symbol> [--all-history]` — dividend snapshot (TTM yield card + recent-range payouts + optional full history) via `/api/v1/stock-infos/{code}/dividends/yield-ratio/histories` + `/api/v1/stock-infos/dividend/{code}/years` + `/api/v1/stock-infos/dividend/{code}/summary`. Non-paying stocks render `"No dividends recorded for this stock."`
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`

### Step 8: Commit

```bash
git add internal/output/dividends.go internal/output/dividends_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock dividends (TTM summary + recent + full history)"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock dividends AAPL` prints populated 2-section output.
- [ ] `/tmp/tossctl stock dividends AAPL --all-history` adds the 전체 히스토리 section.
- [ ] `/tmp/tossctl stock dividends SNDK` prints "No dividends recorded for this stock."
- [ ] `/tmp/tossctl stock dividends AAPL --output json | head -30` confirms snake_case + non-nil growth_ratio.
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 3 endpoints from PR11 spec ✅ (yield-ratio/histories, dividend/years, dividend/summary)
- Empty-payer handled gracefully ✅
- `--all-history` flag for opt-in full history ✅
- snake_case JSON tags ✅
- `encoding/csv` for CSV ✅
- CHANGELOG entry ✅

**Placeholder scan:** none.

**Type consistency:** All field names use snake_case consistently. Wire (envelope) types use camelCase; domain types use snake_case via `json` tags.
