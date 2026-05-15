# PR10 — Stock financials (stability + revenue + operating-income)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `tossctl stock financials <sym>` covering three POST endpoints captured 2026-05-16: `/api/v2/stock-infos/stability/{code}`, `/api/v2/stock-infos/revenue-and-net-profit/{code}`, `/api/v2/stock-infos/operating-income/{code}`. All three use `{stockCode}` (productCode), no companyCode resolution needed. The two dense statements endpoints (`financial-statements/comprehensive`, `financial-statement-records`) are out of PR10 scope — they involve factor/period selectors and a 59KB raw payload that warrants its own command later.

**Architecture:** One client method (`GetStockFinancials`) that makes 3 sequential `postJSONEmpty` calls and stitches results into a single `domain.StockFinancials`. One output writer renders three sections in table mode. Reuses PR9's `postJSONEmpty` and existing patterns.

**Tech Stack:** Go, cobra, postJSONEmpty, fixture-driven httptest, encoding/csv.

**Branch:** continues on `feat/order-page-integration`. Base SHA: `6bcd533`.

---

## File Structure

**New:**
- `internal/client/financials.go` + `_test.go` — `GetStockFinancials`
- `internal/output/financials.go` + `_test.go` — `WriteStockFinancials`
- `fixtures/responses/public/stock-financials-sndk.json` — bundles all 3 responses

**Modified:**
- `internal/domain/models.go` — append `StockFinancials`, `StabilityRatios`, `RevenueSeries`, `RevenuePoint`, `OperatingIncomeSeries`, `OperatingIncomePoint`
- `cmd/tossctl/stock.go` — register `financialsCmd`
- `fixtures/responses/public/manifest.json` — append fixture entry
- `CHANGELOG.md` — add line under `## Unreleased`

---

## Task 1: Domain types + client + fixture + test

**Files:**
- Create: `internal/client/financials.go`
- Create: `internal/client/financials_test.go`
- Create: `fixtures/responses/public/stock-financials-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `fixtures/responses/public/manifest.json`

### Step 1: Save fixture

Create `fixtures/responses/public/stock-financials-sndk.json` (bundles 3 responses, trimmed to last 4 quarters for series):

```json
{
  "stability": {
    "result": {
      "liabilityRatio": 0.0,
      "currentRatio": 478.247,
      "interestCoverageRatio": 68516.667,
      "median": 32.87132985260365,
      "position": "LOW"
    }
  },
  "revenue": {
    "result": {
      "companyName": "샌디스크",
      "recentFiscalYear": 2026,
      "recentFiscalQuarter": 1,
      "recentNetProfit": 3615000000.0,
      "recentNetProfitKrw": 5490462000000.0,
      "fluctuationRate": 350.18,
      "position": "HIGH",
      "graph": [
        {"period": "2025-06", "revenue": 1923000000.0, "revenueKrw": 2607132000000.0, "netProfit": 252000000.0, "netProfitKrw": 341693200000.0, "netProfitRatio": 13.10452418096827},
        {"period": "2025-09", "revenue": 2308000000.0, "revenueKrw": 3243160800000.0, "netProfit": 803000000.0, "netProfitKrw": 1128616600000.0, "netProfitRatio": 34.79202772357020},
        {"period": "2025-12", "revenue": 2306000000.0, "revenueKrw": 3380840800000.0, "netProfit": 887000000.0, "netProfitKrw": 1300316600000.0, "netProfitRatio": 38.46487424934085},
        {"period": "2026-03", "revenue": 3092000000.0, "revenueKrw": 4699192000000.0, "netProfit": 3615000000.0, "netProfitKrw": 5490462000000.0, "netProfitRatio": 116.9133161709574}
      ]
    }
  },
  "operatingIncome": {
    "result": {
      "companyName": "샌디스크",
      "recentFiscalYear": 2026,
      "recentFiscalQuarter": 1,
      "recentOperatingIncome": 4111000000.0,
      "recentOperatingIncomeKrw": 6243786800000.0,
      "fluctuationRate": 286.0,
      "position": "HIGH",
      "graph": [
        {"period": "2025-06", "operatingIncome": 18000000.0, "operatingIncomeKrw": 24436800000.0, "operatingIncomeRatio": 0.9360374415990639},
        {"period": "2025-09", "operatingIncome": 879000000.0, "operatingIncomeKrw": 1234950600000.0, "operatingIncomeRatio": 38.085355285961},
        {"period": "2025-12", "operatingIncome": 1067000000.0, "operatingIncomeKrw": 1564265600000.0, "operatingIncomeRatio": 46.27059843885515},
        {"period": "2026-03", "operatingIncome": 4111000000.0, "operatingIncomeKrw": 6243786800000.0, "operatingIncomeRatio": 132.95019405562743}
      ]
    }
  }
}
```

### Step 2: Update manifest

Append (with comma):

```json
{
  "file": "stock-financials-sndk.json",
  "url": "(combined stability+revenue+operating-income fixture for testing only)",
  "method": "POST"
}
```

### Step 3: Add domain types

Append to `internal/domain/models.go`:

```go
type StockFinancials struct {
	ProductCode     string                 `json:"product_code"`
	Stability       StabilityRatios        `json:"stability"`
	Revenue         RevenueSeries          `json:"revenue"`
	OperatingIncome OperatingIncomeSeries  `json:"operating_income"`
	FetchedAt       time.Time              `json:"fetched_at"`
}

type StabilityRatios struct {
	LiabilityRatio        float64 `json:"liability_ratio"`
	CurrentRatio          float64 `json:"current_ratio"`
	InterestCoverageRatio float64 `json:"interest_coverage_ratio"`
	IndustryMedian        float64 `json:"industry_median"`
	Position              string  `json:"position"` // HIGH|LOW|NORMAL
}

type RevenueSeries struct {
	CompanyName        string         `json:"company_name"`
	RecentFiscalYear   int            `json:"recent_fiscal_year"`
	RecentFiscalQuarter int           `json:"recent_fiscal_quarter"`
	RecentNetProfit    float64        `json:"recent_net_profit"`
	RecentNetProfitKrw float64        `json:"recent_net_profit_krw"`
	FluctuationRate    float64        `json:"fluctuation_rate"`
	Position           string         `json:"position"`
	Graph              []RevenuePoint `json:"graph"`
}

type RevenuePoint struct {
	Period         string  `json:"period"`
	Revenue        float64 `json:"revenue"`
	RevenueKrw     float64 `json:"revenue_krw"`
	NetProfit      float64 `json:"net_profit"`
	NetProfitKrw   float64 `json:"net_profit_krw"`
	NetProfitRatio float64 `json:"net_profit_ratio"`
}

type OperatingIncomeSeries struct {
	CompanyName               string                 `json:"company_name"`
	RecentFiscalYear          int                    `json:"recent_fiscal_year"`
	RecentFiscalQuarter       int                    `json:"recent_fiscal_quarter"`
	RecentOperatingIncome     float64                `json:"recent_operating_income"`
	RecentOperatingIncomeKrw  float64                `json:"recent_operating_income_krw"`
	FluctuationRate           float64                `json:"fluctuation_rate"`
	Position                  string                 `json:"position"`
	Graph                     []OperatingIncomePoint `json:"graph"`
}

type OperatingIncomePoint struct {
	Period               string  `json:"period"`
	OperatingIncome      float64 `json:"operating_income"`
	OperatingIncomeKrw   float64 `json:"operating_income_krw"`
	OperatingIncomeRatio float64 `json:"operating_income_ratio"`
}
```

### Step 4: Write failing client test

Create `internal/client/financials_test.go`:

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

func TestGetStockFinancialsFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Stability       json.RawMessage `json:"stability"`
		Revenue         json.RawMessage `json:"revenue"`
		OperatingIncome json.RawMessage `json:"operatingIncome"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-financials-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/stability/NAS0250224006":
			w.Write(bundle.Stability)
		case "/api/v2/stock-infos/revenue-and-net-profit/NAS0250224006":
			w.Write(bundle.Revenue)
		case "/api/v2/stock-infos/operating-income/NAS0250224006":
			w.Write(bundle.OperatingIncome)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	fin, err := c.GetStockFinancials(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockFinancials error: %v", err)
	}
	if fin.Stability.Position != "LOW" {
		t.Fatalf("expected stability LOW, got %q", fin.Stability.Position)
	}
	if fin.Stability.CurrentRatio != 478.247 {
		t.Fatalf("unexpected currentRatio: %v", fin.Stability.CurrentRatio)
	}
	if fin.Revenue.RecentFiscalYear != 2026 || fin.Revenue.RecentFiscalQuarter != 1 {
		t.Fatalf("unexpected revenue period: %d Q%d", fin.Revenue.RecentFiscalYear, fin.Revenue.RecentFiscalQuarter)
	}
	if len(fin.Revenue.Graph) != 4 {
		t.Fatalf("expected 4 revenue points, got %d", len(fin.Revenue.Graph))
	}
	if fin.Revenue.Graph[3].Period != "2026-03" {
		t.Fatalf("unexpected last period: %q", fin.Revenue.Graph[3].Period)
	}
	if fin.OperatingIncome.Position != "HIGH" {
		t.Fatalf("expected op-income HIGH, got %q", fin.OperatingIncome.Position)
	}
	if len(fin.OperatingIncome.Graph) != 4 {
		t.Fatalf("expected 4 op-income points, got %d", len(fin.OperatingIncome.Graph))
	}
}
```

### Step 5: Run test (FAIL)

Run: `go test ./internal/client/ -run TestGetStockFinancials -v`
Expected: FAIL with "undefined: c.GetStockFinancials".

### Step 6: Implement client method

Create `internal/client/financials.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type stabilityEnvelope struct {
	Result struct {
		LiabilityRatio        float64 `json:"liabilityRatio"`
		CurrentRatio          float64 `json:"currentRatio"`
		InterestCoverageRatio float64 `json:"interestCoverageRatio"`
		Median                float64 `json:"median"`
		Position              string  `json:"position"`
	} `json:"result"`
}

type revenueEnvelope struct {
	Result struct {
		CompanyName         string  `json:"companyName"`
		RecentFiscalYear    int     `json:"recentFiscalYear"`
		RecentFiscalQuarter int     `json:"recentFiscalQuarter"`
		RecentNetProfit     float64 `json:"recentNetProfit"`
		RecentNetProfitKrw  float64 `json:"recentNetProfitKrw"`
		FluctuationRate     float64 `json:"fluctuationRate"`
		Position            string  `json:"position"`
		Graph               []struct {
			Period         string  `json:"period"`
			Revenue        float64 `json:"revenue"`
			RevenueKrw     float64 `json:"revenueKrw"`
			NetProfit      float64 `json:"netProfit"`
			NetProfitKrw   float64 `json:"netProfitKrw"`
			NetProfitRatio float64 `json:"netProfitRatio"`
		} `json:"graph"`
	} `json:"result"`
}

type operatingIncomeEnvelope struct {
	Result struct {
		CompanyName              string  `json:"companyName"`
		RecentFiscalYear         int     `json:"recentFiscalYear"`
		RecentFiscalQuarter      int     `json:"recentFiscalQuarter"`
		RecentOperatingIncome    float64 `json:"recentOperatingIncome"`
		RecentOperatingIncomeKrw float64 `json:"recentOperatingIncomeKrw"`
		FluctuationRate          float64 `json:"fluctuationRate"`
		Position                 string  `json:"position"`
		Graph                    []struct {
			Period               string  `json:"period"`
			OperatingIncome      float64 `json:"operatingIncome"`
			OperatingIncomeKrw   float64 `json:"operatingIncomeKrw"`
			OperatingIncomeRatio float64 `json:"operatingIncomeRatio"`
		} `json:"graph"`
	} `json:"result"`
}

// GetStockFinancials fetches three financial-snapshot endpoints (stability,
// revenue-and-net-profit, operating-income) for a stock and stitches them
// into a single StockFinancials view. All three are POST with empty body.
func (c *Client) GetStockFinancials(ctx context.Context, symbol string) (domain.StockFinancials, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockFinancials{}, err
	}

	var stab stabilityEnvelope
	stabURL := fmt.Sprintf("%s/api/v2/stock-infos/stability/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, stabURL, &stab); err != nil {
		return domain.StockFinancials{}, err
	}

	var rev revenueEnvelope
	revURL := fmt.Sprintf("%s/api/v2/stock-infos/revenue-and-net-profit/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, revURL, &rev); err != nil {
		return domain.StockFinancials{}, err
	}

	var op operatingIncomeEnvelope
	opURL := fmt.Sprintf("%s/api/v2/stock-infos/operating-income/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, opURL, &op); err != nil {
		return domain.StockFinancials{}, err
	}

	revGraph := make([]domain.RevenuePoint, len(rev.Result.Graph))
	for i, g := range rev.Result.Graph {
		revGraph[i] = domain.RevenuePoint{
			Period:         g.Period,
			Revenue:        g.Revenue,
			RevenueKrw:     g.RevenueKrw,
			NetProfit:      g.NetProfit,
			NetProfitKrw:   g.NetProfitKrw,
			NetProfitRatio: g.NetProfitRatio,
		}
	}

	opGraph := make([]domain.OperatingIncomePoint, len(op.Result.Graph))
	for i, g := range op.Result.Graph {
		opGraph[i] = domain.OperatingIncomePoint{
			Period:               g.Period,
			OperatingIncome:      g.OperatingIncome,
			OperatingIncomeKrw:   g.OperatingIncomeKrw,
			OperatingIncomeRatio: g.OperatingIncomeRatio,
		}
	}

	return domain.StockFinancials{
		ProductCode: productCode,
		Stability: domain.StabilityRatios{
			LiabilityRatio:        stab.Result.LiabilityRatio,
			CurrentRatio:          stab.Result.CurrentRatio,
			InterestCoverageRatio: stab.Result.InterestCoverageRatio,
			IndustryMedian:        stab.Result.Median,
			Position:              stab.Result.Position,
		},
		Revenue: domain.RevenueSeries{
			CompanyName:         rev.Result.CompanyName,
			RecentFiscalYear:    rev.Result.RecentFiscalYear,
			RecentFiscalQuarter: rev.Result.RecentFiscalQuarter,
			RecentNetProfit:     rev.Result.RecentNetProfit,
			RecentNetProfitKrw:  rev.Result.RecentNetProfitKrw,
			FluctuationRate:     rev.Result.FluctuationRate,
			Position:            rev.Result.Position,
			Graph:               revGraph,
		},
		OperatingIncome: domain.OperatingIncomeSeries{
			CompanyName:              op.Result.CompanyName,
			RecentFiscalYear:         op.Result.RecentFiscalYear,
			RecentFiscalQuarter:      op.Result.RecentFiscalQuarter,
			RecentOperatingIncome:    op.Result.RecentOperatingIncome,
			RecentOperatingIncomeKrw: op.Result.RecentOperatingIncomeKrw,
			FluctuationRate:          op.Result.FluctuationRate,
			Position:                 op.Result.Position,
			Graph:                    opGraph,
		},
		FetchedAt: time.Now().UTC(),
	}, nil
}
```

### Step 7: Run test (PASS)

Run: `go test ./internal/client/ -run TestGetStockFinancials -v`

### Step 8: Commit

```bash
git add internal/domain/models.go internal/client/financials.go internal/client/financials_test.go fixtures/responses/public/stock-financials-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockFinancials (stability + revenue + operating-income)"
```

---

## Task 2: Output writer + cobra wiring + CHANGELOG

**Files:**
- Create: `internal/output/financials.go`
- Create: `internal/output/financials_test.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `CHANGELOG.md`

### Step 1: Write failing output test

Create `internal/output/financials_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockFinancialsTable(t *testing.T) {
	t.Parallel()
	fin := domain.StockFinancials{
		ProductCode: "NAS0250224006",
		Stability: domain.StabilityRatios{
			LiabilityRatio:        0.0,
			CurrentRatio:          478.247,
			InterestCoverageRatio: 68516.667,
			IndustryMedian:        32.87,
			Position:              "LOW",
		},
		Revenue: domain.RevenueSeries{
			CompanyName: "샌디스크",
			RecentFiscalYear: 2026, RecentFiscalQuarter: 1,
			RecentNetProfit: 3615000000.0, RecentNetProfitKrw: 5490462000000.0,
			FluctuationRate: 350.18, Position: "HIGH",
			Graph: []domain.RevenuePoint{
				{Period: "2025-12", Revenue: 2306000000.0, NetProfit: 887000000.0, NetProfitRatio: 38.46},
				{Period: "2026-03", Revenue: 3092000000.0, NetProfit: 3615000000.0, NetProfitRatio: 116.91},
			},
		},
		OperatingIncome: domain.OperatingIncomeSeries{
			CompanyName: "샌디스크",
			RecentFiscalYear: 2026, RecentFiscalQuarter: 1,
			RecentOperatingIncome: 4111000000.0, RecentOperatingIncomeKrw: 6243786800000.0,
			FluctuationRate: 286.0, Position: "HIGH",
			Graph: []domain.OperatingIncomePoint{
				{Period: "2025-12", OperatingIncome: 1067000000.0, OperatingIncomeRatio: 46.27},
				{Period: "2026-03", OperatingIncome: 4111000000.0, OperatingIncomeRatio: 132.95},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockFinancials(&buf, FormatTable, fin); err != nil {
		t.Fatalf("WriteStockFinancials error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"NAS0250224006", "안정성", "LOW", "478.25",
		"매출 & 순이익", "2026 Q1", "+350.18%",
		"영업이익", "+286.00%",
		"2026-03", "116.91%", "132.95%",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

### Step 2: Run test (FAIL)

Run: `go test ./internal/output/ -run TestWriteStockFinancials -v`

### Step 3: Implement output writer

Create `internal/output/financials.go`:

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

// WriteStockFinancials renders stability + revenue/net-profit series + operating
// income series in a sectioned table; JSON dumps full struct; CSV emits the
// time series.
func WriteStockFinancials(w io.Writer, format Format, fin domain.StockFinancials) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(fin)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "period", "metric", "value_usd", "value_krw", "ratio_pct"}); err != nil {
			return err
		}
		for _, p := range fin.Revenue.Graph {
			if err := cw.Write([]string{
				"revenue", p.Period, "revenue",
				fmt.Sprintf("%.0f", p.Revenue), fmt.Sprintf("%.0f", p.RevenueKrw), "",
			}); err != nil {
				return err
			}
			if err := cw.Write([]string{
				"revenue", p.Period, "net_profit",
				fmt.Sprintf("%.0f", p.NetProfit), fmt.Sprintf("%.0f", p.NetProfitKrw),
				fmt.Sprintf("%.2f", p.NetProfitRatio),
			}); err != nil {
				return err
			}
		}
		for _, p := range fin.OperatingIncome.Graph {
			if err := cw.Write([]string{
				"operating_income", p.Period, "operating_income",
				fmt.Sprintf("%.0f", p.OperatingIncome), fmt.Sprintf("%.0f", p.OperatingIncomeKrw),
				fmt.Sprintf("%.2f", p.OperatingIncomeRatio),
			}); err != nil {
				return err
			}
		}
		if err := cw.Write([]string{
			"stability", "", "liability_ratio",
			"", "", strconv.FormatFloat(fin.Stability.LiabilityRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		if err := cw.Write([]string{
			"stability", "", "current_ratio",
			"", "", strconv.FormatFloat(fin.Stability.CurrentRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		if err := cw.Write([]string{
			"stability", "", "interest_coverage_ratio",
			"", "", strconv.FormatFloat(fin.Stability.InterestCoverageRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — financials\n", fin.ProductCode); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 안정성 (vs 업종 중앙값 %.2f%%) ===\n", fin.Stability.IndustryMedian)
		stHeaders := []string{"METRIC", "VALUE", "POSITION"}
		stRows := [][]string{
			{"부채비율", fmt.Sprintf("%.2f%%", fin.Stability.LiabilityRatio), fin.Stability.Position},
			{"유동비율", fmt.Sprintf("%.2f%%", fin.Stability.CurrentRatio), ""},
			{"이자보상비율", fmt.Sprintf("%.2f%%", fin.Stability.InterestCoverageRatio), ""},
		}
		if err := renderTable(w, stHeaders, stRows); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 매출 & 순이익 — 최근 %d Q%d ===\n", fin.Revenue.RecentFiscalYear, fin.Revenue.RecentFiscalQuarter)
		fmt.Fprintf(w, "Recent net profit: %s (%s KRW)  Fluctuation: %+.2f%%  Position: %s\n",
			formatUSDLarge(fin.Revenue.RecentNetProfit),
			formatWithCommas(int64(fin.Revenue.RecentNetProfitKrw)),
			fin.Revenue.FluctuationRate,
			fin.Revenue.Position,
		)
		revHeaders := []string{"PERIOD", "REVENUE (USD)", "NET PROFIT (USD)", "NET PROFIT MARGIN"}
		revRows := make([][]string, len(fin.Revenue.Graph))
		for i, p := range fin.Revenue.Graph {
			revRows[i] = []string{
				p.Period,
				formatUSDLarge(p.Revenue),
				formatUSDLarge(p.NetProfit),
				fmt.Sprintf("%.2f%%", p.NetProfitRatio),
			}
		}
		if err := renderTable(w, revHeaders, revRows); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 영업이익 — 최근 %d Q%d ===\n", fin.OperatingIncome.RecentFiscalYear, fin.OperatingIncome.RecentFiscalQuarter)
		fmt.Fprintf(w, "Recent operating income: %s (%s KRW)  Fluctuation: %+.2f%%  Position: %s\n",
			formatUSDLarge(fin.OperatingIncome.RecentOperatingIncome),
			formatWithCommas(int64(fin.OperatingIncome.RecentOperatingIncomeKrw)),
			fin.OperatingIncome.FluctuationRate,
			fin.OperatingIncome.Position,
		)
		opHeaders := []string{"PERIOD", "OPERATING INCOME (USD)", "OPERATING MARGIN"}
		opRows := make([][]string, len(fin.OperatingIncome.Graph))
		for i, p := range fin.OperatingIncome.Graph {
			opRows[i] = []string{
				p.Period,
				formatUSDLarge(p.OperatingIncome),
				fmt.Sprintf("%.2f%%", p.OperatingIncomeRatio),
			}
		}
		return renderTable(w, opHeaders, opRows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatUSDLarge formats large USD values with thousand separators, e.g.
// 3092000000 → "$3,092,000,000".
func formatUSDLarge(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := "$" + formatWithCommas(int64(v))
	if neg {
		return "-" + s
	}
	return s
}
```

### Step 4: Run test (PASS)

Run: `go test ./internal/output/ -run TestWriteStockFinancials -v`

### Step 5: Register cobra subcommand

In `cmd/tossctl/stock.go`, after `analystCmd`:

```go
financialsCmd := &cobra.Command{
    Use:   "financials <symbol>",
    Short: "Financial snapshot (stability + revenue/net-profit + operating-income)",
    Long: `Fetch financial-snapshot endpoints behind the 종목정보 FINANCES section.

Stitches three POST endpoints (each with empty body):
  - /api/v2/stock-infos/stability/{code}        — 부채/유동/이자보상비율 + 업종 중앙값
  - /api/v2/stock-infos/revenue-and-net-profit/{code} — 분기별 매출/순이익 시계열
  - /api/v2/stock-infos/operating-income/{code} — 분기별 영업이익 시계열

The graph arrays contain 12 quarters by default (Toss web fixed range).

Examples:
  tossctl stock financials SNDK
  tossctl stock financials NAS0250224006 --output json
  tossctl stock financials SNDK --output csv | column -t -s,`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        fin, err := app.client.GetStockFinancials(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockFinancials(cmd.OutOrStdout(), app.format, fin)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd)
```

### Step 6: Add CHANGELOG line

In `CHANGELOG.md`, under `## [Unreleased]` `### Added`, after the `stock analyst` line, append:

```markdown
- `tossctl stock financials <symbol>` — financial snapshot (안정성 + 매출/순이익 시계열 + 영업이익 시계열) via `/api/v2/stock-infos/stability|revenue-and-net-profit|operating-income/{code}` (all POST `{}`).
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

### Step 8: Commit

```bash
git add internal/output/financials.go internal/output/financials_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock financials (stability + revenue + operating-income)"
```

---

## Task 3: Live verification + push

- [ ] **Step 1:** `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] **Step 2:** Run `/tmp/tossctl stock financials SNDK` and confirm 3 sections populated.
- [ ] **Step 3:** Run `/tmp/tossctl stock financials SNDK --output json | head -30` and confirm well-formed snake_case JSON.
- [ ] **Step 4:** `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 3 endpoints from PR10 spec ✅ (stability, revenue-and-net-profit, operating-income)
- 2 dense endpoints (financial-statements/comprehensive, financial-statement-records) — explicitly deferred to a future PR; large payloads + selector UI need their own design.
- CHANGELOG entry ✅
- Live verify ✅

**Placeholder scan:** none.

**Type consistency:** All field names and JSON tags use snake_case consistently with PR9 final fixes.
