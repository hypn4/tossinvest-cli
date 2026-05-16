# PR15 — `stock ratios` --factor + --period flags

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Extend PR14's `tossctl stock ratios` with `--factor DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO` and `--period Q|Y` flags. The endpoint accepts the same selector body shape as PR13's statements endpoint: `{factorCode, period}`. Verified live: all 3 factors × 2 periods return the same response structure (3 line items × N periods).

**Architecture:** Refactor `GetStockRatios(ctx, symbol)` → `GetStockRatios(ctx, symbol, factorCode, period)`. Reuse the existing `postJSON` helper in `portfolio.go` (same as PR13 did) via `json.RawMessage`. Output writer unchanged — already pivots correctly. Cobra subcommand gains `--factor` and `--period` flags with validation.

**Tech Stack:** Go, cobra, existing `postJSON` (no new helpers), encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `924e684`.

---

## File Structure

**Modified:**
- `internal/client/ratios.go` — accept `factorCode` and `period` params, switch from `postJSONEmpty` to body-based `postJSON`, add validation
- `internal/client/ratios_test.go` — extend tests for non-default factor + bad-input rejection
- `cmd/tossctl/stock.go` — add `--factor` and `--period` flags to `ratiosCmd`
- `CHANGELOG.md` — note new flags

**No new files.** Existing PR14 fixture stays as-is (DEBT_RATIO/Q happy path).

---

## Task 1: Client refactor + new test cases

### Step 1: Update tests first (TDD)

Edit `internal/client/ratios_test.go`. Replace the existing `TestGetStockRatiosFromFixture` to pass `"DEBT_RATIO", "Q"` arguments, and add two new tests:

```go
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockRatiosFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statements/comprehensive" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ra, err := c.GetStockRatios(context.Background(), "NAS0250224006", "DEBT_RATIO", "Q")
	if err != nil {
		t.Fatalf("GetStockRatios error: %v", err)
	}

	// Verify the selector body was sent
	var sent struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("request body not JSON: %v (body=%q)", err, string(gotBody))
	}
	if sent.FactorCode != "DEBT_RATIO" || sent.Period != "Q" {
		t.Fatalf("unexpected request body: %+v", sent)
	}

	if ra.Factor.Code != "DEBT_RATIO" || ra.Factor.DisplayName != "부채비율" {
		t.Fatalf("unexpected factor: %+v", ra.Factor)
	}
	if ra.Period != "Q" {
		t.Fatalf("unexpected period: %q", ra.Period)
	}
	if len(ra.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(ra.Items))
	}
	if ra.Items[2].Code != "DEBT_RATIO" || ra.Items[2].Unit != "PERCENT" {
		t.Fatalf("unexpected last item: %+v", ra.Items[2])
	}
}

func TestGetStockRatiosLowercaseInputsNormalize(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-ratios-sndk-debt-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "current_ratio", "q")
	if err != nil {
		t.Fatalf("expected lowercase inputs to be normalized; got error: %v", err)
	}
	var sent struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("body parse: %v", err)
	}
	if sent.FactorCode != "CURRENT_RATIO" || sent.Period != "Q" {
		t.Fatalf("expected uppercase in body; got %+v", sent)
	}
}

func TestGetStockRatiosRejectsBadFactor(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "BAD_FACTOR", "Q")
	if err == nil {
		t.Fatalf("expected error for bad factor")
	}
}

func TestGetStockRatiosRejectsBadPeriod(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockRatios(context.Background(), "NAS0250224006", "DEBT_RATIO", "X")
	if err == nil {
		t.Fatalf("expected error for bad period")
	}
}
```

### Step 2: Run tests (FAIL — signature mismatch on existing test)

Run: `go test ./internal/client/ -run TestGetStockRatios -v`
Expected: compile error because client method still has 2-arg signature.

### Step 3: Refactor client method

Edit `internal/client/ratios.go`. Replace the existing implementation:

```go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// GetStockRatios fetches the financial-statements/comprehensive endpoint for
// a selected factor + period. The Toss API accepts a JSON body
// {factorCode, period} and returns 3 line items (two components + the
// derived ratio) across N periods (3-year default; the 1년/3년/5년/전체
// range is server-controlled and not exposed via this call).
//
// Validated factor codes: DEBT_RATIO, CURRENT_RATIO, INTEREST_COVERAGE_RATIO.
// Validated periods: Q (quarterly), Y (annual). Inputs are case-insensitive;
// they are normalized to uppercase before the wire call.
func (c *Client) GetStockRatios(ctx context.Context, symbol, factorCode, period string) (domain.StockRatios, error) {
	factorCode = strings.ToUpper(factorCode)
	switch factorCode {
	case "DEBT_RATIO", "CURRENT_RATIO", "INTEREST_COVERAGE_RATIO":
	default:
		return domain.StockRatios{}, fmt.Errorf("GetStockRatios: factorCode must be one of DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO (got %q)", factorCode)
	}
	period = strings.ToUpper(period)
	if period != "Q" && period != "Y" {
		return domain.StockRatios{}, fmt.Errorf("GetStockRatios: period must be Q or Y (got %q)", period)
	}

	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockRatios{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statements/comprehensive", c.infoBaseURL, productCode)

	body, err := json.Marshal(map[string]string{"factorCode": factorCode, "period": period})
	if err != nil {
		return domain.StockRatios{}, err
	}

	var env ratiosEnvelope
	if err := c.postJSON(ctx, endpoint, body, &env); err != nil {
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

Note: `postJSON` in `portfolio.go:187` accepts `body json.RawMessage` (which is `[]byte` under the hood). We pass `body []byte` from `json.Marshal` directly — Go's type system accepts this because `json.RawMessage` is defined as `[]byte`.

### Step 4: Run tests (PASS)

Run: `go test ./internal/client/ -run TestGetStockRatios -v`
Expected: all 4 subtests PASS.

### Step 5: Commit

```bash
git add internal/client/ratios.go internal/client/ratios_test.go
git commit -m "feat(client): accept factor+period in GetStockRatios (selector body)"
```

---

## Task 2: Cobra flags + CHANGELOG

### Step 1: Update cobra command

Edit `cmd/tossctl/stock.go`. Replace the `ratiosCmd` block with:

```go
var ratiosFactor, ratiosPeriod string
ratiosCmd := &cobra.Command{
    Use:   "ratios <symbol>",
    Short: "Solvency-ratio time series with components (debt / current / interest-coverage × Q|Y)",
    Long: `Fetch the financial-statements/comprehensive endpoint (POST {factorCode, period}).

Each factor returns 3 line items × N periods:
  DEBT_RATIO               총자본 + 총부채 + 부채비율
  CURRENT_RATIO            유동자산 + 유동부채 + 유동비율
  INTEREST_COVERAGE_RATIO  영업이익 + 이자비용 + 이자보상비율

--factor selects the factor (default DEBT_RATIO).
--period selects Q (quarterly, 12 points, default) or Y (annual, 3 points).

The 1년 / 3년 / 5년 / 전체 range is server-controlled (3년 default) and not
exposed via this call.

Examples:
  tossctl stock ratios SNDK
  tossctl stock ratios SNDK --factor CURRENT_RATIO
  tossctl stock ratios SNDK --factor INTEREST_COVERAGE_RATIO --period Y
  tossctl stock ratios NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        ra, err := app.client.GetStockRatios(cmd.Context(), args[0], ratiosFactor, ratiosPeriod)
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockRatios(cmd.OutOrStdout(), app.format, ra)
    },
}
ratiosCmd.Flags().StringVar(&ratiosFactor, "factor", "DEBT_RATIO", "Factor: DEBT_RATIO, CURRENT_RATIO, or INTEREST_COVERAGE_RATIO")
ratiosCmd.Flags().StringVar(&ratiosPeriod, "period", "Q", "Period granularity: Q (quarterly) or Y (annual)")
```

(The `cmd.AddCommand(...)` line is unchanged — `ratiosCmd` is still registered.)

### Step 2: Update CHANGELOG

In `CHANGELOG.md`, find the existing `tossctl stock ratios` line under `## [Unreleased]`. Replace it with:

```markdown
- `tossctl stock ratios <symbol> [--factor DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO] [--period Q|Y]` — solvency-ratio time series with components (e.g. 총자본 + 총부채 + 부채비율, or 유동자산 + 유동부채 + 유동비율) via POST `/api/v2/companies/{code}/financial-statements/comprehensive` with `{factorCode, period}` body. Default DEBT_RATIO/Q.
```

### Step 3: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`
Expected: green.

### Step 4: Commit

```bash
git add cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add --factor and --period flags to stock ratios"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock ratios SNDK` — DEBT_RATIO/Q, 12 quarters (default).
- [ ] `/tmp/tossctl stock ratios SNDK --factor CURRENT_RATIO` — 유동비율 with 유동자산 + 유동부채 components.
- [ ] `/tmp/tossctl stock ratios SNDK --factor INTEREST_COVERAGE_RATIO --period Y` — 이자보상비율 annual.
- [ ] `/tmp/tossctl stock ratios SNDK --factor XYZ` — clear error.
- [ ] `/tmp/tossctl stock ratios SNDK --period X` — clear error.
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 3 factors supported ✅
- 2 periods supported ✅
- Case-insensitive input normalization ✅
- Body shape verified live in pre-implementation capture (`{factorCode, period}`) ✅
- Validation rejects bad inputs ✅
- Help text + examples for all 3 factors ✅
- CHANGELOG updated to reflect new flags ✅

**Placeholder scan:** none.

**Type consistency:** Method signature `(ctx, symbol, factorCode, period)` matches the PR13 `GetStockStatements` precedent exactly. Body marshaling follows the same `json.Marshal → []byte → postJSON` pattern.
