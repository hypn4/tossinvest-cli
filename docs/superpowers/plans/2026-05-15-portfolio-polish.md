# Portfolio polish + orderable + my fills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Round out the read-only portfolio surface uncovered by the order-page deep dive: (1) expose the `SORTED_OVERVIEW` fields currently dropped on the floor (tradable/unsettled quantity, after-fees PnL, commission/tax estimate, notice flags, NXT support, after-hours close), (2) add `tossctl orderable` that fuses `cached-orderable-amount` with the per-market settlement overview, and (3) add `tossctl my fills <sym>` that wraps the `/trading/orders/histories/compact/executed` endpoint discovered in the capture.

**Architecture:** Extend the existing `domain.Position` struct rather than introducing a new type (keep `ListPositions` callers ergonomic). Add a thin `OrderableSummary` aggregator that just wires `GetCachedOrderableAmount` + `GetTransactionsOverview("kr"/"us")`. Add a new `Client.ListCompactExecutions` for the per-symbol fills bucketed by time. Each command stays single-shot; no streaming.

**Tech Stack:** Go 1.x, cobra CLI, existing helpers (`getJSON`, `postJSON`, `applySession`, `requireSession`). All endpoints under `wts-api`/`wts-cert-api`, all `auth`.

---

## Reference

- Capture write-up: [`docs/reverse-engineering/order-page-deep-dive.md`](../../reverse-engineering/order-page-deep-dive.md) sections `D 보유 주식 (Holdings / Portfolio)` and `4.2 사용자/거래 컨텍스트`
- Raw captures:
  - `.captures/2026-05-15/us-soxl/home-asset-sections-v2.network-response` (SORTED_OVERVIEW full payload)
  - `.captures/2026-05-15/us-soxl/executed-history-30m.network-response` (compact executed)
- Existing code touchpoints:
  - `internal/client/portfolio.go::ListPositions` decodes `SORTED_OVERVIEW`
  - `internal/client/account.go::orderableAmountEnvelope` already shapes the `cached-orderable-amount` body
  - `internal/client/transactions.go::GetTransactionsOverview` already wraps `transactions/markets/{m}/overview`

Fields we must surface (currently missing from `domain.Position`):

| API field | Domain field to add |
| --- | --- |
| `tradableQuantity` | `TradableQuantity` |
| `unsettledQuantity` | `UnsettledQuantity` |
| `evaluatedAmount.{krw,usd}` AfterFees | `MarketValueAfterFees`, `MarketValueAfterFeesUSD` |
| `profitLossAmountAfterFees.{krw,usd}` | `UnrealizedPnLAfterFees`, `UnrealizedPnLAfterFeesUSD` |
| `profitLossRateAfterFees.{krw,usd}` | `ProfitRateAfterFees`, `ProfitRateAfterFeesUSD` |
| `commission.{krw,usd}` | `EstimatedCommission`, `EstimatedCommissionUSD` |
| `commissionRate` | `CommissionRate` |
| `tax.{krw,usd}` | `EstimatedTax`, `EstimatedTaxUSD` |
| `taxRate` | `TaxRate` |
| `closeWithoutAfter.{krw,usd}` | `CloseWithoutAfter`, `CloseWithoutAfterUSD` |
| `delisting` | `Delisting` (bool) |
| `nxtSupported` | `NXTSupported` (bool) |
| `notice.splitMerge`/`earningsAnnouncement` | `NoticeSplitMerge`, `NoticeEarningsAnnouncement` |

## File Structure

| File | Status | Responsibility |
| --- | --- | --- |
| `internal/domain/models.go` | Modify | Extend `Position`. Add `OrderableSummary`, `CompactExecution` types |
| `internal/client/portfolio.go` | Modify | Decode new fields in `sortedOverviewData` |
| `internal/client/portfolio_test.go` | Modify | Cover new fields |
| `internal/client/account.go` | Modify | Add `GetCachedOrderableAmount` helper if not already public |
| `internal/client/orderable.go` | Create | `GetOrderableSummary` aggregator |
| `internal/client/orderable_test.go` | Create | Unit test using mocked endpoints |
| `internal/client/fills.go` | Create | `ListCompactExecutions` |
| `internal/client/fills_test.go` | Create | Unit test |
| `internal/output/orderable.go` | Create | Format `OrderableSummary` |
| `internal/output/orderable_test.go` | Create | |
| `internal/output/fills.go` | Create | Format `CompactExecution[]` |
| `internal/output/fills_test.go` | Create | |
| `internal/output/portfolio.go` | Modify | Add new column in table view (optional flag `--detail`) |
| `cmd/tossctl/orderable.go` | Create | `tossctl orderable` |
| `cmd/tossctl/fills.go` | Create | `tossctl my fills <sym> [--tf 30m]` |
| `cmd/tossctl/root.go` | Modify | Register two new commands (`newOrderableCmd`, `newMyCmd`) |
| `fixtures/responses/public/orderable-amount.json` | Create | Sanitized cached-orderable-amount |
| `fixtures/responses/public/transactions-overview-us.json` | Create | Sanitized US overview |
| `fixtures/responses/public/transactions-overview-kr.json` | Create | Sanitized KR overview |
| `fixtures/responses/public/compact-executed-us.json` | Create | Sanitized fills |
| `docs/reverse-engineering/rpc-catalog.md` | Modify | CLI mappings |
| `CHANGELOG.md` | Modify | Release notes |

---

## Task 1: Extend Position domain type

**Files:**
- Modify: `internal/domain/models.go`

- [ ] **Step 1: Add new fields to `Position`**

Locate the existing `Position` struct in `internal/domain/models.go`. Replace the entire struct with:

```go
type Position struct {
	ProductCode     string  `json:"product_code,omitempty"`
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name,omitempty"`
	MarketType      string  `json:"market_type,omitempty"`
	MarketCode      string  `json:"market_code,omitempty"`
	Quantity        float64 `json:"quantity"`
	TradableQuantity   float64 `json:"tradable_quantity,omitempty"`
	UnsettledQuantity  float64 `json:"unsettled_quantity,omitempty"`
	AveragePrice    float64 `json:"average_price,omitempty"`
	CurrentPrice    float64 `json:"current_price,omitempty"`
	CloseWithoutAfter float64 `json:"close_without_after,omitempty"`
	MarketValue     float64 `json:"market_value,omitempty"`
	MarketValueAfterFees float64 `json:"market_value_after_fees,omitempty"`
	UnrealizedPnL   float64 `json:"unrealized_pnl,omitempty"`
	UnrealizedPnLAfterFees float64 `json:"unrealized_pnl_after_fees,omitempty"`
	ProfitRate      float64 `json:"profit_rate,omitempty"`
	ProfitRateAfterFees float64 `json:"profit_rate_after_fees,omitempty"`
	DailyProfitLoss float64 `json:"daily_profit_loss,omitempty"`
	DailyProfitRate float64 `json:"daily_profit_rate,omitempty"`
	EstimatedCommission float64 `json:"estimated_commission,omitempty"`
	CommissionRate     float64 `json:"commission_rate,omitempty"`
	EstimatedTax       float64 `json:"estimated_tax,omitempty"`
	TaxRate            float64 `json:"tax_rate,omitempty"`
	Delisting          bool    `json:"delisting,omitempty"`
	NXTSupported       bool    `json:"nxt_supported,omitempty"`
	NoticeSplitMerge          bool `json:"notice_split_merge,omitempty"`
	NoticeEarningsAnnouncement bool `json:"notice_earnings_announcement,omitempty"`

	AveragePriceUSD    float64 `json:"average_price_usd,omitempty"`
	CurrentPriceUSD    float64 `json:"current_price_usd,omitempty"`
	CloseWithoutAfterUSD float64 `json:"close_without_after_usd,omitempty"`
	MarketValueUSD     float64 `json:"market_value_usd,omitempty"`
	MarketValueAfterFeesUSD float64 `json:"market_value_after_fees_usd,omitempty"`
	UnrealizedPnLUSD   float64 `json:"unrealized_pnl_usd,omitempty"`
	UnrealizedPnLAfterFeesUSD float64 `json:"unrealized_pnl_after_fees_usd,omitempty"`
	ProfitRateUSD      float64 `json:"profit_rate_usd,omitempty"`
	ProfitRateAfterFeesUSD float64 `json:"profit_rate_after_fees_usd,omitempty"`
	DailyProfitLossUSD float64 `json:"daily_profit_loss_usd,omitempty"`
	DailyProfitRateUSD float64 `json:"daily_profit_rate_usd,omitempty"`
	EstimatedCommissionUSD float64 `json:"estimated_commission_usd,omitempty"`
	EstimatedTaxUSD        float64 `json:"estimated_tax_usd,omitempty"`
}
```

- [ ] **Step 2: Append the new types at the bottom**

```go
type OrderableSummary struct {
	OrderableKR domain_money `json:"orderable_kr"`
	OrderableUS domain_money `json:"orderable_us"`
	KR          TransactionOverview `json:"kr_overview"`
	US          TransactionOverview `json:"us_overview"`
	FetchedAt   time.Time `json:"fetched_at"`
}

type domain_money struct {
	KRW float64 `json:"krw,omitempty"`
	USD float64 `json:"usd,omitempty"`
}

type CompactExecution struct {
	ProductCode             string    `json:"product_code"`
	TradeType               string    `json:"trade_type"` // "buy" or "sell"
	ExecutionAvgKRWPrice    float64   `json:"execution_avg_krw_price"`
	ExecutionAvgLocalPrice  float64   `json:"execution_avg_local_price"`
	ExecutionTotalKRWAmount float64   `json:"execution_total_krw_amount"`
	ExecutionTotalLocal     float64   `json:"execution_total_local"`
	Quantity                float64   `json:"quantity"`
	BucketStart             time.Time `json:"bucket_start"`
}
```

(The `domain_money` type name is intentionally snake-cased to avoid clashing with the existing `Money` value pattern; we only use it inside this package as an aggregate alias.)

- [ ] **Step 3: Build**

Run: `go build ./internal/domain/...`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/models.go
git commit -m "feat(domain): extend Position; add OrderableSummary, CompactExecution"
```

---

## Task 2: Decode the new Position fields in portfolio client

**Files:**
- Modify: `internal/client/portfolio.go`
- Modify: `internal/client/portfolio_test.go`

- [ ] **Step 1: Write the failing test (or extend existing) for new fields**

Open `internal/client/portfolio_test.go` and locate the existing `TestListPositions` (or equivalent). Add at the bottom of that test, **after the existing positions assertion**, additional checks:

```go
	var soxl *domain.Position
	for i := range positions {
		if positions[i].ProductCode == "US20100311002" {
			soxl = &positions[i]
		}
	}
	if soxl == nil {
		t.Fatal("expected SOXL position in fixture")
	}
	if soxl.TradableQuantity != 20 {
		t.Fatalf("expected TradableQuantity=20, got %v", soxl.TradableQuantity)
	}
	if soxl.MarketValueAfterFees == 0 || soxl.MarketValueAfterFees >= soxl.MarketValue {
		t.Fatalf("after-fees value should be smaller and non-zero: %+v", soxl)
	}
	if soxl.CommissionRate != 0.001 {
		t.Fatalf("expected commissionRate=0.001, got %v", soxl.CommissionRate)
	}
```

The existing test fixture (the asset-sections fixture) must already contain SOXL with the new fields; if the file omits them, copy the relevant slice from `.captures/2026-05-15/us-soxl/home-asset-sections-v2.network-response` into the fixture (sanitize account-no by replacing it with `99999999999`).

- [ ] **Step 2: Run; expect failure**

Run: `go test ./internal/client/ -run TestListPositions -v`
Expected: at least one of the new assertions fails (`TradableQuantity != 20` or undefined field).

- [ ] **Step 3: Extend `sortedOverviewData` in `portfolio.go`**

In `internal/client/portfolio.go`, edit the anonymous struct inside `sortedOverviewData.Products[].Items[]` to add the new fields:

```go
				TradableQuantity float64 `json:"tradableQuantity"`
				UnsettledQuantity float64 `json:"unsettledQuantity"`
				CloseWithoutAfter struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"closeWithoutAfter"`
				EvaluatedAmountAfterFees struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"evaluatedAmountAfterFees"`
				ProfitLossAmountAfterFees struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"profitLossAmountAfterFees"`
				ProfitLossRateAfterFees struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"profitLossRateAfterFees"`
				Commission struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"commission"`
				CommissionRate float64 `json:"commissionRate"`
				Tax struct {
					KRW *float64 `json:"krw"`
					USD *float64 `json:"usd"`
				} `json:"tax"`
				TaxRate    float64 `json:"taxRate"`
				Delisting  bool   `json:"delisting"`
				NXTSupported bool `json:"nxtSupported"`
				Notice struct {
					SplitMerge          bool `json:"splitMerge"`
					EarningsAnnouncement bool `json:"earningsAnnouncement"`
				} `json:"notice"`
```

(Append immediately after the existing `MarketCode string` field, inside the same anonymous item struct.)

- [ ] **Step 4: Map the new fields into `domain.Position` in the `for product, item` loop**

Right after the existing `DailyProfitRateUSD: derefFloat(item.DailyProfitLossRate.USD),` line (the last entry in the struct literal), insert the new fields:

```go
				TradableQuantity:           item.TradableQuantity,
				UnsettledQuantity:          item.UnsettledQuantity,
				CloseWithoutAfter:          coalesceMoney(item.CloseWithoutAfter.KRW, item.CloseWithoutAfter.USD),
				CloseWithoutAfterUSD:       derefFloat(item.CloseWithoutAfter.USD),
				MarketValueAfterFees:       coalesceMoney(item.EvaluatedAmountAfterFees.KRW, item.EvaluatedAmountAfterFees.USD),
				MarketValueAfterFeesUSD:    derefFloat(item.EvaluatedAmountAfterFees.USD),
				UnrealizedPnLAfterFees:     coalesceMoney(item.ProfitLossAmountAfterFees.KRW, item.ProfitLossAmountAfterFees.USD),
				UnrealizedPnLAfterFeesUSD:  derefFloat(item.ProfitLossAmountAfterFees.USD),
				ProfitRateAfterFees:        coalesceMoney(item.ProfitLossRateAfterFees.KRW, item.ProfitLossRateAfterFees.USD),
				ProfitRateAfterFeesUSD:     derefFloat(item.ProfitLossRateAfterFees.USD),
				EstimatedCommission:        coalesceMoney(item.Commission.KRW, item.Commission.USD),
				EstimatedCommissionUSD:     derefFloat(item.Commission.USD),
				CommissionRate:             item.CommissionRate,
				EstimatedTax:               coalesceMoney(item.Tax.KRW, item.Tax.USD),
				EstimatedTaxUSD:            derefFloat(item.Tax.USD),
				TaxRate:                    item.TaxRate,
				Delisting:                  item.Delisting,
				NXTSupported:               item.NXTSupported,
				NoticeSplitMerge:           item.Notice.SplitMerge,
				NoticeEarningsAnnouncement: item.Notice.EarningsAnnouncement,
```

- [ ] **Step 5: Run the test; PASS**

Run: `go test ./internal/client/ -run TestListPositions -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/client/portfolio.go internal/client/portfolio_test.go
git commit -m "feat(client): decode tradable/unsettled/afterFees/commission/notice on Position"
```

---

## Task 3: GetCachedOrderableAmount helper

**Files:**
- Modify: `internal/client/account.go`
- Create: `internal/client/orderable_test.go` (later — first the helper)

- [ ] **Step 1: Inspect the existing decoder**

The envelope `orderableAmountEnvelope` already exists in `account.go`. Check whether the package already exposes a method that returns just the parsed amounts. If yes, skip to Task 4. If only an internal helper exists, expose it:

```go
// CachedOrderableAmount is what /api/v1/dashboard/common/cached-orderable-amount returns.
type CachedOrderableAmount struct {
	KR domain.Money `json:"kr"`
	US domain.Money `json:"us"`
}
```

(`domain.Money` does not exist — use the snake-cased helper from Task 1: `domain.domain_money` — but that is unexported. Cleanest is to define a local pair here and convert at the boundary.)

To avoid leaking unexported types across packages, use:

```go
type cachedOrderableAmount struct {
	KRkrw float64
	KRusd float64
	USkrw float64
	USusd float64
}

func (c *Client) GetCachedOrderableAmount(ctx context.Context) (cachedOrderableAmount, error) {
	if err := c.requireSession(); err != nil {
		return cachedOrderableAmount{}, err
	}
	var envelope orderableAmountEnvelope
	if err := c.getJSON(ctx, c.certBaseURL+"/api/v1/dashboard/common/cached-orderable-amount", &envelope); err != nil {
		return cachedOrderableAmount{}, err
	}
	return cachedOrderableAmount{
		KRkrw: pointerFloat(envelope.Result.OrderableAmountKr.KRW),
		KRusd: pointerFloat(envelope.Result.OrderableAmountKr.USD),
		USkrw: pointerFloat(envelope.Result.OrderableAmountUs.KRW),
		USusd: pointerFloat(envelope.Result.OrderableAmountUs.USD),
	}, nil
}
```

Place this near the other `account.go` helpers (after `orderableAmountEnvelope` is defined).

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/client/account.go
git commit -m "feat(client): expose GetCachedOrderableAmount helper"
```

---

## Task 4: GetOrderableSummary aggregator + fixtures + test

**Files:**
- Create: `internal/client/orderable.go`
- Create: `internal/client/orderable_test.go`
- Create: `fixtures/responses/public/orderable-amount.json`
- Create: `fixtures/responses/public/transactions-overview-us.json`
- Create: `fixtures/responses/public/transactions-overview-kr.json`

- [ ] **Step 1: Drop the fixtures**

```bash
cat > fixtures/responses/public/orderable-amount.json <<'JSON'
{
  "result": {
    "orderableAmountKr": {"krw": 110421, "usd": null},
    "orderableAmountUs": {"krw": 758893, "usd": 508.71}
  }
}
JSON

cat > fixtures/responses/public/transactions-overview-us.json <<'JSON'
{
  "result": {
    "orderableAmount": {"krw": 758893, "usd": 508.71},
    "withdrawableAmount": {
      "amount0": {"krw": 0, "usd": 508.71},
      "date0": "2026-05-15",
      "amount1": {"krw": 0, "usd": 508.71},
      "date1": "2026-05-18"
    },
    "displayWithdrawableAmount": {
      "amount0": {"krw": 0, "usd": 508.71},
      "date0": "2026-05-15"
    },
    "depositAmount": {
      "amount0": {"krw": 0, "usd": 0},
      "date0": "2026-05-15"
    },
    "estimateSettlementAmount": {}
  }
}
JSON

cat > fixtures/responses/public/transactions-overview-kr.json <<'JSON'
{
  "result": {
    "orderableAmount": {"krw": 110421, "usd": null},
    "withdrawableAmount": {
      "amount0": {"krw": 110421, "usd": null},
      "date0": "2026-05-15"
    },
    "displayWithdrawableAmount": {
      "amount0": {"krw": 110421, "usd": null},
      "date0": "2026-05-15"
    },
    "depositAmount": {},
    "estimateSettlementAmount": {}
  }
}
JSON
```

- [ ] **Step 2: Write the failing aggregator test**

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
)

func orderableFixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func TestGetOrderableSummary(t *testing.T) {
	t.Parallel()

	root := orderableFixtureRoot(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/dashboard/common/cached-orderable-amount", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "orderable-amount.json"))
	})
	mux.HandleFunc("/api/v3/my-assets/transactions/markets/us/overview", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "transactions-overview-us.json"))
	})
	mux.HandleFunc("/api/v3/my-assets/transactions/markets/kr/overview", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "transactions-overview-kr.json"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		CertBaseURL: server.URL,
		Session:     &testSession,
	})
	summary, err := c.GetOrderableSummary(context.Background())
	if err != nil {
		t.Fatalf("GetOrderableSummary returned error: %v", err)
	}
	if summary.OrderableKR.KRW != 110421 {
		t.Fatalf("unexpected KR orderable KRW: %v", summary.OrderableKR.KRW)
	}
	if summary.OrderableUS.USD != 508.71 {
		t.Fatalf("unexpected US orderable USD: %v", summary.OrderableUS.USD)
	}
	if summary.US.Market != "us" {
		t.Fatalf("US overview missing")
	}
	if summary.KR.Market != "kr" {
		t.Fatalf("KR overview missing")
	}
}
```

`testSession` is the session-fixture helper used by other client tests. Reference how `internal/client/transactions_test.go` constructs its `Client` and reuse the same pattern (e.g. `&session.Session{Cookies: map[string]string{"SESSION":"x"}}`).

- [ ] **Step 3: Run; expect failure**

Run: `go test ./internal/client/ -run TestGetOrderableSummary -v`
Expected: `undefined: Client.GetOrderableSummary` or compile error from `testSession`.

If `testSession` is undefined here, define one at the top of the new test file:

```go
import "github.com/junghoonkye/tossinvest-cli/internal/session"

var testSession = session.Session{Cookies: map[string]string{"SESSION": "test"}}
```

Then re-run; expect `undefined: Client.GetOrderableSummary`.

- [ ] **Step 4: Implement GetOrderableSummary**

```go
package client

import (
	"context"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// GetOrderableSummary fans out to cached-orderable-amount and per-market
// transaction overview, returning a single struct for the CLI.
func (c *Client) GetOrderableSummary(ctx context.Context) (domain.OrderableSummary, error) {
	if err := c.requireSession(); err != nil {
		return domain.OrderableSummary{}, err
	}

	cached, err := c.GetCachedOrderableAmount(ctx)
	if err != nil {
		return domain.OrderableSummary{}, err
	}
	kr, err := c.GetTransactionsOverview(ctx, "kr")
	if err != nil {
		return domain.OrderableSummary{}, err
	}
	us, err := c.GetTransactionsOverview(ctx, "us")
	if err != nil {
		return domain.OrderableSummary{}, err
	}

	return domain.OrderableSummary{
		OrderableKR: moneyOf(cached.KRkrw, cached.KRusd),
		OrderableUS: moneyOf(cached.USkrw, cached.USusd),
		KR:          kr,
		US:          us,
		FetchedAt:   time.Now().UTC(),
	}, nil
}

func moneyOf(krw, usd float64) (m struct {
	KRW float64 `json:"krw,omitempty"`
	USD float64 `json:"usd,omitempty"`
}) {
	m.KRW = krw
	m.USD = usd
	return
}
```

Note: `domain.OrderableSummary.OrderableKR` is the unexported helper type `domain_money` from Task 1. To make the assignment compile cleanly, switch the field types in `domain/models.go` from `domain_money` to `struct{KRW, USD float64 …}` inline, or export the helper:

```go
type Money struct {
	KRW float64 `json:"krw,omitempty"`
	USD float64 `json:"usd,omitempty"`
}

type OrderableSummary struct {
	OrderableKR Money               `json:"orderable_kr"`
	OrderableUS Money               `json:"orderable_us"`
	KR          TransactionOverview `json:"kr_overview"`
	US          TransactionOverview `json:"us_overview"`
	FetchedAt   time.Time           `json:"fetched_at"`
}
```

Then drop the `domain_money` type from Task 1. Simplify `moneyOf` in `orderable.go`:

```go
func moneyOf(krw, usd float64) domain.Money {
	return domain.Money{KRW: krw, USD: usd}
}
```

- [ ] **Step 5: Run; expect PASS**

Run: `go test ./internal/client/ -run TestGetOrderableSummary -v`
Expected: PASS.

- [ ] **Step 6: Update the manifest**

```bash
python3 - <<'PY'
import json, pathlib
p = pathlib.Path("fixtures/responses/public/manifest.json")
data = json.loads(p.read_text())
new = [
    {"file": "orderable-amount.json", "url": "https://wts-cert-api.tossinvest.com/api/v1/dashboard/common/cached-orderable-amount", "method": "GET"},
    {"file": "transactions-overview-us.json", "url": "https://wts-api.tossinvest.com/api/v3/my-assets/transactions/markets/us/overview", "method": "GET"},
    {"file": "transactions-overview-kr.json", "url": "https://wts-api.tossinvest.com/api/v3/my-assets/transactions/markets/kr/overview", "method": "GET"},
]
existing = {e["file"] for e in data.get("fixtures", [])}
for entry in new:
    if entry["file"] not in existing:
        data["fixtures"].append(entry)
p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")
print("manifest updated")
PY
```

- [ ] **Step 7: Commit**

```bash
git add internal/client/orderable.go internal/client/orderable_test.go internal/domain/models.go fixtures/responses/public/
git commit -m "feat(client): add GetOrderableSummary aggregator"
```

---

## Task 5: ListCompactExecutions client + fixture

**Files:**
- Create: `internal/client/fills.go`
- Create: `internal/client/fills_test.go`
- Create: `fixtures/responses/public/compact-executed-us.json`

- [ ] **Step 1: Drop the fixture**

```bash
cat > fixtures/responses/public/compact-executed-us.json <<'JSON'
{
  "result": {
    "pagingParam": {
      "from": "2021-01-01T00:00:00+09:00",
      "to":   "2026-05-15T21:09:41+09:00",
      "cursorKey": null
    },
    "body": [
      {
        "productCode": "US20100311002",
        "tradeType": "buy",
        "executionAvgKrwPrice": 277505.50,
        "executionTotalKrwPrice": 5550110,
        "executionTotalKrwAmount": 5550110,
        "executionAvgLocalPrice": 185.61,
        "executionTotalLocalPrice": 3712.20,
        "executionTotalLocalAmount": 3712.20,
        "quantity": 20.0,
        "executedTimeBucket": "2026-05-15T02:00:00+09:00"
      },
      {
        "productCode": "US20100311002",
        "tradeType": "sell",
        "executionAvgKrwPrice": 282992.49,
        "executionTotalKrwPrice": 9904737,
        "executionTotalKrwAmount": 9904737,
        "executionAvgLocalPrice": 189.28,
        "executionTotalLocalPrice": 6624.80,
        "executionTotalLocalAmount": 6624.80,
        "quantity": 35.0,
        "executedTimeBucket": "2026-05-15T00:00:00+09:00"
      }
    ],
    "lastPage": true
  }
}
JSON
```

- [ ] **Step 2: Write the failing test**

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
)

func fillsFixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func TestListCompactExecutions(t *testing.T) {
	t.Parallel()

	root := fillsFixtureRoot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/trading/orders/histories/compact/executed" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("productCode"); got != "US20100311002" {
			t.Fatalf("unexpected productCode: %s", got)
		}
		if got := r.URL.Query().Get("timeUnit"); got != "thirty_minute" {
			t.Fatalf("unexpected timeUnit: %s", got)
		}
		http.ServeFile(w, r, filepath.Join(root, "compact-executed-us.json"))
	}))
	defer server.Close()

	sess := session.Session{Cookies: map[string]string{"SESSION": "test"}}
	c := New(Config{HTTPClient: server.Client(), CertBaseURL: server.URL, Session: &sess})
	fills, err := c.ListCompactExecutions(context.Background(), "US20100311002", "thirty_minute")
	if err != nil {
		t.Fatalf("ListCompactExecutions returned error: %v", err)
	}
	if len(fills) != 2 {
		t.Fatalf("expected 2 fills, got %d", len(fills))
	}
	if fills[0].TradeType != "buy" {
		t.Fatalf("unexpected first trade type: %s", fills[0].TradeType)
	}
	if fills[1].Quantity != 35 {
		t.Fatalf("unexpected second quantity: %v", fills[1].Quantity)
	}
}
```

Add `import "github.com/junghoonkye/tossinvest-cli/internal/session"` at the top.

- [ ] **Step 3: Run; expect failure**

Run: `go test ./internal/client/ -run TestListCompactExecutions -v`
Expected: `undefined: Client.ListCompactExecutions`.

- [ ] **Step 4: Implement ListCompactExecutions**

Create `internal/client/fills.go`:

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

type compactExecutedEnvelope struct {
	Result struct {
		Body []struct {
			ProductCode               string  `json:"productCode"`
			TradeType                 string  `json:"tradeType"`
			ExecutionAvgKrwPrice      float64 `json:"executionAvgKrwPrice"`
			ExecutionTotalKrwAmount   float64 `json:"executionTotalKrwAmount"`
			ExecutionAvgLocalPrice    float64 `json:"executionAvgLocalPrice"`
			ExecutionTotalLocalAmount float64 `json:"executionTotalLocalAmount"`
			Quantity                  float64 `json:"quantity"`
			ExecutedTimeBucket        time.Time `json:"executedTimeBucket"`
		} `json:"body"`
	} `json:"result"`
}

// ListCompactExecutions wraps /api/v3/trading/orders/histories/compact/executed.
// timeUnit accepted by Toss includes "thirty_minute" (verified in capture);
// other granularities are observed but not validated here — we forward as-is.
func (c *Client) ListCompactExecutions(ctx context.Context, productCode, timeUnit string) ([]domain.CompactExecution, error) {
	if err := c.requireSession(); err != nil {
		return nil, err
	}
	productCode = strings.TrimSpace(productCode)
	if productCode == "" {
		return nil, fmt.Errorf("productCode is required")
	}
	timeUnit = strings.TrimSpace(timeUnit)
	if timeUnit == "" {
		timeUnit = "thirty_minute"
	}

	endpoint, err := url.Parse(c.certBaseURL + "/api/v3/trading/orders/histories/compact/executed")
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("productCode", productCode)
	q.Set("timeUnit", timeUnit)
	q.Set("excludeSavings", "false")
	endpoint.RawQuery = q.Encode()

	var envelope compactExecutedEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.CompactExecution, 0, len(envelope.Result.Body))
	for _, raw := range envelope.Result.Body {
		out = append(out, domain.CompactExecution{
			ProductCode:             raw.ProductCode,
			TradeType:               raw.TradeType,
			ExecutionAvgKRWPrice:    raw.ExecutionAvgKrwPrice,
			ExecutionAvgLocalPrice:  raw.ExecutionAvgLocalPrice,
			ExecutionTotalKRWAmount: raw.ExecutionTotalKrwAmount,
			ExecutionTotalLocal:     raw.ExecutionTotalLocalAmount,
			Quantity:                raw.Quantity,
			BucketStart:             raw.ExecutedTimeBucket,
		})
	}
	return out, nil
}
```

- [ ] **Step 5: Run the test**

Run: `go test ./internal/client/ -run TestListCompactExecutions -v`
Expected: PASS.

- [ ] **Step 6: Update manifest, commit**

```bash
python3 - <<'PY'
import json, pathlib
p = pathlib.Path("fixtures/responses/public/manifest.json")
data = json.loads(p.read_text())
entry = {"file": "compact-executed-us.json", "url": "https://wts-cert-api.tossinvest.com/api/v3/trading/orders/histories/compact/executed?productCode=US20100311002&timeUnit=thirty_minute&excludeSavings=false", "method": "GET"}
if entry["file"] not in {e["file"] for e in data["fixtures"]}:
    data["fixtures"].append(entry)
p.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")
PY

git add internal/client/fills.go internal/client/fills_test.go fixtures/responses/public/compact-executed-us.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add ListCompactExecutions (30m own-fill buckets)"
```

---

## Task 6: Output formatter — OrderableSummary

**Files:**
- Create: `internal/output/orderable.go`
- Create: `internal/output/orderable_test.go`

- [ ] **Step 1: Write the failing tests**

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

var sampleOrderable = domain.OrderableSummary{
	OrderableKR: domain.Money{KRW: 110421},
	OrderableUS: domain.Money{KRW: 758893, USD: 508.71},
	KR: domain.TransactionOverview{Market: "kr", OrderableKRW: 110421, Withdrawable: []domain.SettlementBucket{{Date: "2026-05-15", KRW: 110421}}},
	US: domain.TransactionOverview{Market: "us", OrderableUSD: 508.71, Withdrawable: []domain.SettlementBucket{{Date: "2026-05-15", USD: 508.71}}},
	FetchedAt: time.Now(),
}

func TestWriteOrderableJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderable(&buf, FormatJSON, sampleOrderable); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OrderableSummary
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.OrderableUS.USD != 508.71 {
		t.Fatalf("USD round-trip failed: %+v", parsed.OrderableUS)
	}
}

func TestWriteOrderableTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderable(&buf, FormatTable, sampleOrderable); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "110,421") || !strings.Contains(out, "$508.71") {
		t.Fatalf("expected KR/US amounts in output: %s", out)
	}
}
```

- [ ] **Step 2: Run; expect failure**

Run: `go test ./internal/output/ -run TestWriteOrderable -v`
Expected: `undefined: WriteOrderable`.

- [ ] **Step 3: Implement WriteOrderable**

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOrderable renders the orderable / withdrawable summary.
func WriteOrderable(w io.Writer, format Format, s domain.OrderableSummary) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(s)
	case FormatCSV:
		return fmt.Errorf("csv output is not supported for orderable summary")
	case FormatTable:
		if _, err := fmt.Fprintf(w, "Orderable KR: ₩%s\n", formatKRW(s.OrderableKR.KRW)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Orderable US: %s (≈ ₩%s)\n", formatUSD(s.OrderableUS.USD), formatKRW(s.OrderableUS.KRW)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "\nKR overview:"); err != nil {
			return err
		}
		if err := WriteTransactionsOverview(w, FormatTable, s.KR); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "\nUS overview:"); err != nil {
			return err
		}
		return WriteTransactionsOverview(w, FormatTable, s.US)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/output/ -run TestWriteOrderable -v`
Expected: two PASS lines.

- [ ] **Step 5: Commit**

```bash
git add internal/output/orderable.go internal/output/orderable_test.go
git commit -m "feat(output): add WriteOrderable formatter"
```

---

## Task 7: Output formatter — CompactExecution

**Files:**
- Create: `internal/output/fills.go`
- Create: `internal/output/fills_test.go`

- [ ] **Step 1: Write the failing tests**

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

var sampleFills = []domain.CompactExecution{
	{
		ProductCode:             "US20100311002",
		TradeType:               "buy",
		ExecutionAvgKRWPrice:    277505.50,
		ExecutionAvgLocalPrice:  185.61,
		ExecutionTotalKRWAmount: 5550110,
		ExecutionTotalLocal:     3712.20,
		Quantity:                20.0,
		BucketStart:             time.Date(2026, 5, 15, 2, 0, 0, 0, time.UTC),
	},
}

func TestWriteFillsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatJSON, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.CompactExecution
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed[0].Quantity != 20 {
		t.Fatalf("unexpected qty: %v", parsed[0].Quantity)
	}
}

func TestWriteFillsTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatTable, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"buy", "20", "185.61"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output: %s", want, out)
		}
	}
}

func TestWriteFillsCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatCSV, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d", len(lines))
	}
}
```

- [ ] **Step 2: Run; expect failure**

Run: `go test ./internal/output/ -run TestWriteFills -v`
Expected: `undefined: WriteFills`.

- [ ] **Step 3: Implement WriteFills**

```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteFills(w io.Writer, format Format, fills []domain.CompactExecution) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(fills)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"product_code", "trade_type", "bucket_start",
			"quantity", "execution_avg_local_price", "execution_avg_krw_price",
			"execution_total_local", "execution_total_krw",
		}); err != nil {
			return err
		}
		for _, f := range fills {
			if err := writer.Write([]string{
				f.ProductCode,
				f.TradeType,
				f.BucketStart.Format("2006-01-02T15:04:05Z07:00"),
				formatFloat(f.Quantity),
				formatFloat(f.ExecutionAvgLocalPrice),
				formatFloat(f.ExecutionAvgKRWPrice),
				formatFloat(f.ExecutionTotalLocal),
				formatFloat(f.ExecutionTotalKRWAmount),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"BUCKET", "SIDE", "QTY", "AVG (local)", "AVG (KRW)", "TOTAL (local)"}
		rows := make([][]string, 0, len(fills))
		for _, f := range fills {
			rows = append(rows, []string{
				f.BucketStart.Format("2006-01-02 15:04"),
				f.TradeType,
				formatQty(f.Quantity),
				formatFloat(f.ExecutionAvgLocalPrice),
				formatKRW(f.ExecutionAvgKRWPrice),
				formatFloat(f.ExecutionTotalLocal),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/output/ -run TestWriteFills -v`
Expected: three PASS lines.

- [ ] **Step 5: Commit**

```bash
git add internal/output/fills.go internal/output/fills_test.go
git commit -m "feat(output): add WriteFills formatter"
```

---

## Task 8: CLI command — tossctl orderable

**Files:**
- Create: `cmd/tossctl/orderable.go`
- Modify: `cmd/tossctl/root.go`

- [ ] **Step 1: Create the command**

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newOrderableCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "orderable",
		Short: "Show orderable + withdrawable summary across KR and US markets",
		Long: `Show the combined orderable cash + withdrawable amounts per
settlement date for both KR and US markets.

The output bundles three Toss endpoints:
  - /api/v1/dashboard/common/cached-orderable-amount
  - /api/v3/my-assets/transactions/markets/kr/overview
  - /api/v3/my-assets/transactions/markets/us/overview

Use --output json to get a single structured payload for LLM/agent pipelines.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			summary, err := app.client.GetOrderableSummary(cmd.Context())
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOrderable(cmd.OutOrStdout(), app.format, summary)
		},
	}
}
```

- [ ] **Step 2: Register in root.go**

In `cmd/tossctl/root.go`, after `newSignalsCmd(opts),` insert:

```go
		newOrderableCmd(opts),
```

- [ ] **Step 3: Build and smoke help**

Run: `go build ./...`
Expected: no output.

Run: `go run ./cmd/tossctl orderable --help`
Expected output begins with `Show orderable + withdrawable summary across KR and US markets`.

- [ ] **Step 4: Commit**

```bash
git add cmd/tossctl/orderable.go cmd/tossctl/root.go
git commit -m "feat(cli): add tossctl orderable"
```

---

## Task 9: CLI command — tossctl my fills

**Files:**
- Create: `cmd/tossctl/my.go`
- Modify: `cmd/tossctl/root.go`

- [ ] **Step 1: Create the parent `my` command with `fills` subcommand**

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newMyCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "my",
		Short: "Personal data (own fills, recent activity)",
	}

	var timeUnit string
	fillsCmd := &cobra.Command{
		Use:   "fills <symbol>",
		Short: "Show own executions bucketed by time (chart overlay)",
		Long: `List the executions of the current account for a given symbol,
bucketed by --tf granularity (default thirty_minute).

This wraps /api/v3/trading/orders/histories/compact/executed.

Examples:
  tossctl my fills SOXL                       # 30-minute buckets
  tossctl my fills SOXL --tf one_hour         # 1-hour buckets (if supported)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			productCode, err := app.client.ResolveProductCode(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			fills, err := app.client.ListCompactExecutions(cmd.Context(), productCode, timeUnit)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteFills(cmd.OutOrStdout(), app.format, fills)
		},
	}
	fillsCmd.Flags().StringVar(&timeUnit, "tf", "thirty_minute", "Bucket size: thirty_minute (default) or other Toss-supported timeUnit string")

	cmd.AddCommand(fillsCmd)
	return cmd
}
```

- [ ] **Step 2: Register in root.go**

After `newOrderableCmd(opts),` insert:

```go
		newMyCmd(opts),
```

- [ ] **Step 3: Build and smoke**

Run: `go build ./...`
Expected: no output.

Run: `go run ./cmd/tossctl my fills --help`
Expected output begins with `List the executions of the current account ...` and includes the `--tf` flag.

- [ ] **Step 4: Commit**

```bash
git add cmd/tossctl/my.go cmd/tossctl/root.go
git commit -m "feat(cli): add tossctl my fills"
```

---

## Task 10: Surface new Position fields in portfolio table view

**Files:**
- Modify: `internal/output/portfolio.go`

- [ ] **Step 1: Add a `--detail` aware column block**

The existing `writePositionsTable` adds a column for unrealized PnL. We want to:
- Show `TradableQuantity` if it differs from `Quantity` (`tradable/total`)
- Show after-fees PnL in tooltip-style line after each market section (optional)

Locate `writePositionsTable` in `internal/output/portfolio.go`. After the row-building loop, before `return renderTable(...)`, append a follow-up loop that prints commission/tax footnotes:

```go
	// Footer notes for fees and notices
	for _, p := range positions {
		if p.EstimatedCommission == 0 && p.EstimatedTax == 0 && !p.Delisting && !p.NoticeSplitMerge && !p.NoticeEarningsAnnouncement {
			continue
		}
		notes := []string{}
		if p.EstimatedCommission != 0 {
			notes = append(notes, fmt.Sprintf("수수료 추정 %s (%.3f%%)", formatKRW(p.EstimatedCommission), p.CommissionRate*100))
		}
		if p.EstimatedTax != 0 {
			notes = append(notes, fmt.Sprintf("세금 추정 %s (%.3f%%)", formatKRW(p.EstimatedTax), p.TaxRate*100))
		}
		if p.Delisting {
			notes = append(notes, "상장폐지 예정")
		}
		if p.NoticeSplitMerge {
			notes = append(notes, "액면분할/합병 공지")
		}
		if p.NoticeEarningsAnnouncement {
			notes = append(notes, "실적 발표 예정")
		}
		fmt.Fprintf(w, "  · %s: %s\n", p.Symbol, strings.Join(notes, " · "))
	}
```

Add `"strings"` to the imports if not already there.

- [ ] **Step 2: Run existing portfolio output tests**

Run: `go test ./internal/output/ -run Portfolio -v`
Expected: all existing tests still pass.

- [ ] **Step 3: Commit**

```bash
git add internal/output/portfolio.go
git commit -m "feat(output): surface commission/tax/notice notes in portfolio table"
```

---

## Task 11: RPC catalog mapping update

**Files:**
- Modify: `docs/reverse-engineering/rpc-catalog.md`

- [ ] **Step 1: Update CLI mapping columns**

For the following rows, change the `CLI mapping` cell text:

| Endpoint | New CLI mapping |
| --- | --- |
| `/api/v2/dashboard/asset/sections/all` `{"types":["SORTED_OVERVIEW"]}` | `portfolio positions` (extended in v0.5.x) |
| `/api/v1/dashboard/common/cached-orderable-amount` | `orderable` (v0.5.x) |
| `/api/v3/my-assets/transactions/markets/{market}/overview` | `orderable` / `transactions overview` |
| `/api/v3/trading/orders/histories/compact/executed` | `my fills <sym>` (v0.5.x) |

- [ ] **Step 2: Commit**

```bash
git add docs/reverse-engineering/rpc-catalog.md
git commit -m "docs(catalog): wire CLI mappings for orderable, my fills, portfolio fields"
```

---

## Task 12: CHANGELOG entry

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Append release notes**

```markdown
### Added
- `tossctl orderable` — 한·미 주문가능금액 + 출금가능액 (결제일별) + 정산일 묶음을 한 번에. JSON 출력은 LLM 컨텍스트로 그대로 흘리기 좋게 단일 구조체.
- `tossctl my fills <sym> [--tf thirty_minute]` — 본인 체결 내역을 30분 (또는 토스 지원 timeUnit) 버킷으로 출력. 차트 오버레이 용도.

### Changed
- `tossctl portfolio` 가 `tradableQuantity`, `unsettledQuantity`, `evaluatedAmountAfterFees`, `profitLossAmountAfterFees`, `commission` (+ `commissionRate`), `tax` (+ `taxRate`), `delisting`, `nxtSupported`, `notice.{splitMerge,earningsAnnouncement}` 까지 노출. JSON 키는 모두 신규 — 기존 키는 그대로 유지되어 backward compatible.
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): note orderable, my fills, portfolio polish"
```

---

## Task 13: Full integration sweep

**Files:** none

- [ ] **Step 1: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Tests**

Run: `go test ./...`
Expected: every package ok.

- [ ] **Step 4: Smoke commands**

Run: `go run ./cmd/tossctl orderable --help`
Expected: shows the long description and no errors.

Run: `go run ./cmd/tossctl my fills --help`
Expected: shows `--tf` flag.

Run: `go run ./cmd/tossctl portfolio positions --help` (or whatever the existing subcommand is)
Expected: existing flags unchanged.

- [ ] **Step 5: git status check**

Run: `git status`
Expected: `nothing to commit, working tree clean`.

---

## Self-review notes

- `Position` is widened, not replaced; existing JSON output stays compatible because new fields use `omitempty`. CSV header in the existing `WriteOrders` / `WriteTransactions` test fixtures is unchanged.
- `OrderableSummary` includes both the `cached-orderable-amount` quick read (millisecond freshness) and the per-market settlement detail. They can disagree if the settlement window flipped right between calls — that's acceptable for a snapshot summary and matches what the Toss web UI shows.
- `my fills` uses Toss's `timeUnit` string directly so future-Toss-added granularities work without code changes. Currently only `thirty_minute` is verified; document this fact in the long help.
- `tossctl my` is a parent command with one subcommand (`fills`) for now; future expansion (`my pending`, `my completed`, `my analysis`) plugs in without command renames.
- Tests use `httptest` mock servers per file; no live network. The session fixture is a minimal `session.Session{Cookies:{"SESSION":"test"}}` because the only requirement is that `requireSession` doesn't error.
- `WriteOrderable` reuses the existing `WriteTransactionsOverview` to render KR/US sections — DRY.
