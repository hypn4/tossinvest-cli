# PR14 — Stock ratios (debt-ratio time series with components)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `tossctl stock ratios <sym>` covering the deferred `financial-statements/comprehensive` endpoint. Returns 3 line items (총자본 + 총부채 + 부채비율) across 12 periods (3-year quarterly default), showing both the ratio and its components.

**Scope:** Empty-body POST only, returns DEBT_RATIO by default. Server supports CURRENT_RATIO and INTEREST_COVERAGE_RATIO via selector body (matching PR13 pattern), but those variants are NOT covered in PR14 — they need their own captures before we ship `--factor` and `--period` flags.

**Architecture:** One client method `GetStockRatios(ctx, symbol)` that calls `postJSONEmpty` on `/api/v2/companies/{code}/financial-statements/comprehensive`. Output writer pivots: rows = line items, columns = periods. Unit-aware formatting (AMOUNT items show `formatWithCommas`, PERCENT items show `%.2f%%`).

**Tech Stack:** Go, cobra, postJSONEmpty (reused), encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `7b54931`.

---

## File Structure

**New:**
- `internal/client/ratios.go` + `_test.go` — `GetStockRatios`
- `internal/output/ratios.go` + `_test.go` — `WriteStockRatios`
- `fixtures/responses/public/stock-ratios-sndk-debt-q.json` — fixture

**Modified:**
- `internal/domain/models.go` — append `StockRatios`, `RatioFactor`, `RatioLineItem`, `RatioValue` types
- `cmd/tossctl/stock.go` — register `ratiosCmd`
- `fixtures/responses/public/manifest.json` — append fixture entry
- `CHANGELOG.md` — add line

---

## Task 1: Domain + client + fixture + test

### Step 1: Save fixture

Create `fixtures/responses/public/stock-ratios-sndk-debt-q.json`. Trim values to 4 periods:

```json
{
  "result": {
    "selectedFactor": {"code": "DEBT_RATIO", "displayName": "부채비율"},
    "selectableFactors": [
      {"code": "DEBT_RATIO", "displayName": "부채비율"},
      {"code": "CURRENT_RATIO", "displayName": "유동비율"},
      {"code": "INTEREST_COVERAGE_RATIO", "displayName": "이자보상비율"}
    ],
    "selectedRange": {"code": 3, "displayName": "3년"},
    "selectableRanges": [
      {"code": 1, "displayName": "1년"},
      {"code": 3, "displayName": "3년"},
      {"code": 5, "displayName": "5년"},
      {"code": 2147483647, "displayName": "전체"}
    ],
    "selectedPeriod": {"code": "Q", "displayName": "분기"},
    "selectablePeriods": [
      {"code": "Q", "displayName": "분기"},
      {"code": "Y", "displayName": "연간"}
    ],
    "graph": [
      {
        "code": "TOTAL_SHAREHOLDERS_EQUITY",
        "unit": "AMOUNT",
        "name": "총자본",
        "values": [
          {"period": "2025-06", "value": 11082000000.0, "valueKrw": 15395114400000.0},
          {"period": "2025-09", "value": 12200000000.0, "valueKrw": 17142800000000.0},
          {"period": "2025-12", "value": 13100000000.0, "valueKrw": 19207320000000.0},
          {"period": "2026-03", "value": 16700000000.0, "valueKrw": 25382120000000.0}
        ]
      },
      {
        "code": "TOTAL_LIABILITIES",
        "unit": "AMOUNT",
        "name": "총부채",
        "values": [
          {"period": "2025-06", "value": 5200000000.0, "valueKrw": 7223840000000.0},
          {"period": "2025-09", "value": 4800000000.0, "valueKrw": 6745920000000.0},
          {"period": "2025-12", "value": 4500000000.0, "valueKrw": 6597000000000.0},
          {"period": "2026-03", "value": 0.0, "valueKrw": 0.0}
        ]
      },
      {
        "code": "DEBT_RATIO",
        "unit": "PERCENT",
        "name": "부채비율",
        "values": [
          {"period": "2025-06", "value": 46.92, "valueKrw": 0.0},
          {"period": "2025-09", "value": 39.34, "valueKrw": 0.0},
          {"period": "2025-12", "value": 34.35, "valueKrw": 0.0},
          {"period": "2026-03", "value": 0.0, "valueKrw": 0.0}
        ]
      }
    ],
    "table": []
  }
}
```

### Step 2: Update manifest

Append:

```json
{
  "file": "stock-ratios-sndk-debt-q.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/companies/NAS0250224006/financial-statements/comprehensive",
  "method": "POST"
}
```

### Step 3: Add domain types

Append to `internal/domain/models.go`:

```go
type StockRatios struct {
	ProductCode string          `json:"product_code"`
	Factor      RatioFactor     `json:"factor"`   // DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO
	Period      string          `json:"period"`   // Q|Y
	RangeLabel  string          `json:"range_label"` // 1년|3년|5년|전체
	Items       []RatioLineItem `json:"items"`
	FetchedAt   time.Time       `json:"fetched_at"`
}

type RatioFactor struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type RatioLineItem struct {
	Code   string       `json:"code"`           // e.g. TOTAL_SHAREHOLDERS_EQUITY
	Unit   string       `json:"unit"`           // AMOUNT|PERCENT
	Name   string       `json:"name"`           // 총자본
	Values []RatioValue `json:"values"`
}

type RatioValue struct {
	Period   string  `json:"period"`
	Value    float64 `json:"value"`
	ValueKrw float64 `json:"value_krw,omitempty"`
}
```

### Step 4: Failing client test

Create `internal/client/ratios_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockRatiosFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statements/comprehensive" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ra, err := c.GetStockRatios(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockRatios error: %v", err)
	}
	if ra.Factor.Code != "DEBT_RATIO" || ra.Factor.DisplayName != "부채비율" {
		t.Fatalf("unexpected factor: %+v", ra.Factor)
	}
	if ra.Period != "Q" {
		t.Fatalf("unexpected period: %q", ra.Period)
	}
	if ra.RangeLabel != "3년" {
		t.Fatalf("unexpected range: %q", ra.RangeLabel)
	}
	if len(ra.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(ra.Items))
	}
	if ra.Items[0].Code != "TOTAL_SHAREHOLDERS_EQUITY" {
		t.Fatalf("unexpected first item: %+v", ra.Items[0])
	}
	if ra.Items[2].Code != "DEBT_RATIO" || ra.Items[2].Unit != "PERCENT" {
		t.Fatalf("unexpected last item: %+v", ra.Items[2])
	}
	if len(ra.Items[0].Values) != 4 {
		t.Fatalf("expected 4 values per item, got %d", len(ra.Items[0].Values))
	}
	if ra.Items[2].Values[2].Value != 34.35 {
		t.Fatalf("unexpected debt ratio Q4-2025: %v", ra.Items[2].Values[2].Value)
	}
}
```

### Step 5: Run test (FAIL)

### Step 6: Implement client

Create `internal/client/ratios.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type ratiosEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedFactor"`
		SelectedRange struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedRange"`
		SelectedPeriod struct {
			Code string `json:"code"`
		} `json:"selectedPeriod"`
		Graph []struct {
			Code   string `json:"code"`
			Unit   string `json:"unit"`
			Name   string `json:"name"`
			Values []struct {
				Period   string  `json:"period"`
				Value    float64 `json:"value"`
				ValueKrw float64 `json:"valueKrw"`
			} `json:"values"`
		} `json:"graph"`
	} `json:"result"`
}

// GetStockRatios fetches the financial-statements/comprehensive payload —
// the time series for the default factor (DEBT_RATIO) and its component
// line items, across the default range (3년) and period (Q). Toss's server
// supports other factor/range/period selections via body, but PR14 ships
// only the default case; future PRs may add --factor / --period flags
// after fresh captures verify those variants.
func (c *Client) GetStockRatios(ctx context.Context, symbol string) (domain.StockRatios, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockRatios{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statements/comprehensive", c.infoBaseURL, productCode)
	var env ratiosEnvelope
	if err := c.postJSONEmpty(ctx, endpoint, &env); err != nil {
		return domain.StockRatios{}, err
	}

	items := make([]domain.RatioLineItem, len(env.Result.Graph))
	for i, g := range env.Result.Graph {
		vals := make([]domain.RatioValue, len(g.Values))
		for j, v := range g.Values {
			vals[j] = domain.RatioValue{Period: v.Period, Value: v.Value, ValueKrw: v.ValueKrw}
		}
		items[i] = domain.RatioLineItem{
			Code:   g.Code,
			Unit:   g.Unit,
			Name:   g.Name,
			Values: vals,
		}
	}

	return domain.StockRatios{
		ProductCode: productCode,
		Factor: domain.RatioFactor{
			Code:        env.Result.SelectedFactor.Code,
			DisplayName: env.Result.SelectedFactor.DisplayName,
		},
		Period:     env.Result.SelectedPeriod.Code,
		RangeLabel: env.Result.SelectedRange.DisplayName,
		Items:      items,
		FetchedAt:  time.Now().UTC(),
	}, nil
}
```

### Step 7: Run test (PASS)

### Step 8: Commit

```bash
git add internal/domain/models.go internal/client/ratios.go internal/client/ratios_test.go fixtures/responses/public/stock-ratios-sndk-debt-q.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockRatios (debt-ratio time series with components)"
```

---

## Task 2: Output writer + cobra + CHANGELOG

### Step 1: Output test

Create `internal/output/ratios_test.go`:

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

func ratiosTestFixture() domain.StockRatios {
	return domain.StockRatios{
		ProductCode: "NAS0250224006",
		Factor:      domain.RatioFactor{Code: "DEBT_RATIO", DisplayName: "부채비율"},
		Period:      "Q",
		RangeLabel:  "3년",
		Items: []domain.RatioLineItem{
			{
				Code: "TOTAL_SHAREHOLDERS_EQUITY", Unit: "AMOUNT", Name: "총자본",
				Values: []domain.RatioValue{
					{Period: "2025-12", Value: 13100000000.0, ValueKrw: 19207320000000.0},
					{Period: "2026-03", Value: 16700000000.0, ValueKrw: 25382120000000.0},
				},
			},
			{
				Code: "TOTAL_LIABILITIES", Unit: "AMOUNT", Name: "총부채",
				Values: []domain.RatioValue{
					{Period: "2025-12", Value: 4500000000.0, ValueKrw: 6597000000000.0},
					{Period: "2026-03", Value: 0.0, ValueKrw: 0.0},
				},
			},
			{
				Code: "DEBT_RATIO", Unit: "PERCENT", Name: "부채비율",
				Values: []domain.RatioValue{
					{Period: "2025-12", Value: 34.35},
					{Period: "2026-03", Value: 0.0},
				},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockRatiosTable(t *testing.T) {
	t.Parallel()
	ra := ratiosTestFixture()
	var buf bytes.Buffer
	if err := WriteStockRatios(&buf, FormatTable, ra); err != nil {
		t.Fatalf("WriteStockRatios error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"NAS0250224006", "부채비율", "(Q, 3년)",
		"ITEM", "2025-12", "2026-03",
		"총자본",
		"총부채",
		"34.35%",
		"13,100,000,000",  // amount comma-formatted
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockRatiosCSV(t *testing.T) {
	t.Parallel()
	ra := ratiosTestFixture()
	var buf bytes.Buffer
	if err := WriteStockRatios(&buf, FormatCSV, ra); err != nil {
		t.Fatalf("CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if rows[0][0] != "code" || rows[0][1] != "name" || rows[0][2] != "unit" || rows[0][3] != "period" || rows[0][4] != "value" {
		t.Fatalf("unexpected CSV header: %v", rows[0])
	}
	// 3 items × 2 periods = 6 data rows + 1 header
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}
}

func TestWriteStockRatiosJSON(t *testing.T) {
	t.Parallel()
	ra := ratiosTestFixture()
	var buf bytes.Buffer
	if err := WriteStockRatios(&buf, FormatJSON, ra); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	var got domain.StockRatios
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if got.Factor.Code != "DEBT_RATIO" {
		t.Fatalf("factor roundtrip mismatch")
	}
	if !strings.Contains(buf.String(), `"display_name"`) {
		t.Fatalf("expected snake_case display_name in JSON")
	}
	if !strings.Contains(buf.String(), `"range_label"`) {
		t.Fatalf("expected snake_case range_label in JSON")
	}
}
```

### Step 2: Run test (FAIL)

### Step 3: Implement output writer

Create `internal/output/ratios.go`:

```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockRatios renders the factor time series — typically 3 line items
// (two components + the derived ratio) across N periods. Rows are line items,
// columns are periods. AMOUNT-unit values use comma-separated millions;
// PERCENT-unit values use %.2f%%.
func WriteStockRatios(w io.Writer, format Format, ra domain.StockRatios) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ra)

	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"code", "name", "unit", "period", "value", "value_krw"}); err != nil {
			return err
		}
		for _, it := range ra.Items {
			for _, v := range it.Values {
				if err := cw.Write([]string{
					it.Code, it.Name, it.Unit, v.Period,
					strconv.FormatFloat(v.Value, 'f', 4, 64),
					strconv.FormatFloat(v.ValueKrw, 'f', 2, 64),
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()

	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — %s (%s, %s)\n", ra.ProductCode, ra.Factor.DisplayName, ra.Period, ra.RangeLabel); err != nil {
			return err
		}
		if len(ra.Items) == 0 {
			fmt.Fprintln(w, "  (no rows returned)")
			return nil
		}

		// Periods come from the first item; the API always returns the same
		// period grid across items.
		periodCols := make([]string, len(ra.Items[0].Values))
		for i, v := range ra.Items[0].Values {
			periodCols[i] = v.Period
		}
		headers := append([]string{"ITEM"}, periodCols...)

		rows := make([][]string, len(ra.Items))
		for i, it := range ra.Items {
			row := make([]string, 0, 1+len(periodCols))
			row = append(row, it.Name)
			for _, v := range it.Values {
				row = append(row, formatRatioValue(v.Value, it.Unit))
			}
			rows[i] = row
		}

		if err := renderTable(w, headers, rows); err != nil {
			return err
		}
		fmt.Fprintln(w, "(AMOUNT values are raw units; PERCENT values are formatted as percentages)")
		return nil

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatRatioValue(v float64, unit string) string {
	switch unit {
	case "PERCENT":
		return fmt.Sprintf("%.2f%%", v)
	case "AMOUNT":
		return formatWithCommas(int64(math.Round(v)))
	default:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
}
```

### Step 4: Run test (PASS)

### Step 5: Register cobra subcommand

In `cmd/tossctl/stock.go`, after `statementsCmd`:

```go
ratiosCmd := &cobra.Command{
    Use:   "ratios <symbol>",
    Short: "Debt-ratio time series with components (총자본 + 총부채 + 부채비율 across 12 quarters)",
    Long: `Fetch the financial-statements/comprehensive endpoint (POST {}).

Default response: DEBT_RATIO factor, 분기 (Q) period, 3년 range — 3 line items
(총자본 + 총부채 + 부채비율) × 12 periods.

Toss's server also supports CURRENT_RATIO and INTEREST_COVERAGE_RATIO via
selector body, plus 연간 (Y) period and 1년/5년/전체 ranges. PR14 ships only
the empty-body default; --factor/--period flags will be added in a future PR
after fresh captures verify those variants.

Examples:
  tossctl stock ratios SNDK
  tossctl stock ratios NAS0250224006 --output json
  tossctl stock ratios SNDK --output csv`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        ra, err := app.client.GetStockRatios(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockRatios(cmd.OutOrStdout(), app.format, ra)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd, dividendsCmd, estimatesCmd, statementsCmd, ratiosCmd)
```

### Step 6: CHANGELOG entry

In `CHANGELOG.md`, after the `stock statements` line, add:

```markdown
- `tossctl stock ratios <symbol>` — debt-ratio time series with components (총자본 + 총부채 + 부채비율 × 12 quarters) via POST `/api/v2/companies/{code}/financial-statements/comprehensive`. Empty-body call, default DEBT_RATIO/Q/3년. CURRENT_RATIO / INTEREST_COVERAGE_RATIO selectors and 연간 period deferred to a future PR.
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`

### Step 8: Commit

```bash
git add internal/output/ratios.go internal/output/ratios_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock ratios (debt-ratio time series + components)"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock ratios SNDK` prints 3 line items × 12 periods.
- [ ] `/tmp/tossctl stock ratios SNDK --output json | head -20` confirms snake_case keys.
- [ ] `/tmp/tossctl stock ratios SNDK --output csv | head -5` shows the 6-column header.
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 1 endpoint (empty-body POST) ✅
- DEBT_RATIO default ✅
- Pivoted output ✅
- Unit-aware formatting (AMOUNT vs PERCENT) ✅
- snake_case JSON tags ✅
- `encoding/csv` for CSV ✅
- CHANGELOG entry ✅
- Future-PR deferral notes in code comments + help text ✅

**Placeholder scan:** none.

**Type consistency:** All field names use snake_case. Wire (envelope) types use camelCase. Tests cover happy path + 3 output formats.
