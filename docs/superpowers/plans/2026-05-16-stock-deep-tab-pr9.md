# PR9 — Stock deep-tab CLI: indicators / valuation / revenue / peers / analyst

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship five new `tossctl stock <subcmd>` read-only commands that cover the captured deep-tab endpoints from 2026-05-16 (`investment-indicators`, `evaluation`+`evaluation-comparison`, `sales-compositions`, `tics`, `analyst-opinion`+`consensus`+`analyst-reports`).

**Architecture:** Each command follows the PR8 template: domain types in `internal/domain/models.go`, client method(s) in a new `internal/client/<name>.go` with httptest+fixture test, output writer in `internal/output/<name>.go`, cobra subcommand appended to `cmd/tossctl/stock.go`, fixture JSON committed to `fixtures/responses/public/` with manifest entry. Two shared helpers (`resolveCompanyCode`, `postJSONEmpty`) land first to keep individual commands small.

**Tech Stack:** Go 1.22+, cobra, standard library `net/http`/`net/http/httptest`, fixture-driven testing.

**Branch:** continues on `feat/order-page-integration`. No new branch; each task commits directly. All pushes are `origin` (fork) only — never `upstream`.

---

## File Structure

**New files:**
- `internal/client/companycode.go` — `(c *Client) resolveCompanyCode(ctx, sym) (string, error)` (reuses `GetCompanyOverview`).
- `internal/client/http_post.go` — `(c *Client) postJSONEmpty(ctx, url, dst) error` (shared by valuation/financials/estimates).
- `internal/client/indicators.go` + `_test.go` — `GetStockIndicators`.
- `internal/client/valuation.go` + `_test.go` — `GetStockValuation`.
- `internal/client/revenue.go` + `_test.go` — `GetSalesComposition`.
- `internal/client/peers.go` + `_test.go` — `GetTICSIndustry`.
- `internal/client/analyst.go` + `_test.go` — `GetAnalystSnapshot` (3 endpoints stitched).
- `internal/output/indicators.go` + `_test.go` — `WriteStockIndicators`.
- `internal/output/valuation.go` + `_test.go` — `WriteStockValuation`.
- `internal/output/revenue.go` + `_test.go` — `WriteSalesComposition`.
- `internal/output/peers.go` + `_test.go` — `WriteTICSIndustry`.
- `internal/output/analyst.go` + `_test.go` — `WriteAnalystSnapshot`.
- `fixtures/responses/public/stock-indicators-sndk.json`
- `fixtures/responses/public/stock-valuation-sndk.json` (combined `{evaluation: {…}, comparison: {…}}` for testing)
- `fixtures/responses/public/sales-compositions-sndk.json`
- `fixtures/responses/public/tics-sndk.json`
- `fixtures/responses/public/analyst-snapshot-sndk.json` (combined `{opinion: {…}, consensus: {…}, reports: {…}}`)

**Modified files:**
- `internal/domain/models.go` — append `StockIndicators`, `IndicatorSection`, `StockValuation`, `PeerValuation`, `SalesComposition`, `SalesCompositionItem`, `TICSIndustry`, `TICSEntry`, `TICSPeerRanking`, `AnalystSnapshot`, `AnalystOpinion`, `ConsensusTarget`, `ConsensusPastClose`, `AnalystReport`.
- `cmd/tossctl/stock.go` — append `indicatorsCmd`, `valuationCmd`, `revenueCmd`, `peersCmd`, `analystCmd` and register them.
- `fixtures/responses/public/manifest.json` — append five new fixture entries.
- `CHANGELOG.md` — add `### Added` lines under `## Unreleased`.

---

## Task 1: Shared client helpers

**Files:**
- Create: `internal/client/companycode.go`
- Create: `internal/client/http_post.go`
- Create: `internal/client/http_post_test.go`

Both helpers are used by multiple later tasks. Land them first.

- [ ] **Step 1: Write the failing test for `postJSONEmpty`**

Create `internal/client/http_post_test.go`:

```go
package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJSONEmptySendsEmptyBody(t *testing.T) {
	t.Parallel()
	var gotMethod, gotCT string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"result":{"ok":true}}`))
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	var env struct {
		Result struct {
			OK bool `json:"ok"`
		} `json:"result"`
	}
	if err := c.postJSONEmpty(context.Background(), server.URL+"/anything", &env); err != nil {
		t.Fatalf("postJSONEmpty error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotCT != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", gotCT)
	}
	if string(gotBody) != "{}" {
		t.Fatalf("expected body {}, got %q", string(gotBody))
	}
	if !env.Result.OK {
		t.Fatalf("expected env.Result.OK to be true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/client/ -run TestPostJSONEmpty -v`
Expected: FAIL with "undefined: postJSONEmpty" (compile error).

- [ ] **Step 3: Implement `postJSONEmpty`**

Create `internal/client/http_post.go`:

```go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// postJSONEmpty issues a POST request with body `{}` and decodes the JSON
// response into dst. Used by Toss endpoints that gate on a non-empty request
// body (e.g. /api/v2/stock-infos/evaluation/{code}).
func (c *Client) postJSONEmpty(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", DefaultBrowserUserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: %s — %s", endpoint, resp.Status, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/client/ -run TestPostJSONEmpty -v`
Expected: PASS.

- [ ] **Step 5: Implement `resolveCompanyCode`**

Create `internal/client/companycode.go`:

```go
package client

import (
	"context"
	"fmt"
)

// resolveCompanyCode maps a symbol or productCode to the company-level code
// returned by /api/v2/stock-infos/{code}/overview (e.g. NAS116LTR-E0). This
// company code is required by /api/v1/companies/{code}/... endpoints.
func (c *Client) resolveCompanyCode(ctx context.Context, symbolOrCode string) (string, error) {
	ov, err := c.GetCompanyOverview(ctx, symbolOrCode)
	if err != nil {
		return "", fmt.Errorf("resolveCompanyCode(%q): %w", symbolOrCode, err)
	}
	if ov.Company.Code == "" {
		return "", fmt.Errorf("resolveCompanyCode(%q): overview returned empty company code", symbolOrCode)
	}
	return ov.Company.Code, nil
}
```

(No separate test — exercised end-to-end by Task 4 + 5.)

- [ ] **Step 6: Run full client tests**

Run: `go test ./internal/client/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/client/http_post.go internal/client/http_post_test.go internal/client/companycode.go
git commit -m "feat(client): add postJSONEmpty + resolveCompanyCode helpers"
```

---

## Task 2: `tossctl stock indicators <sym>`

**Files:**
- Create: `internal/client/indicators.go`
- Create: `internal/client/indicators_test.go`
- Create: `internal/output/indicators.go`
- Create: `internal/output/indicators_test.go`
- Create: `fixtures/responses/public/stock-indicators-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `fixtures/responses/public/manifest.json`

Endpoint: `GET /api/v1/stock-detail/ui/wts/{code}/investment-indicators`.

- [ ] **Step 1: Save fixture**

Copy the captured JSON. Create `fixtures/responses/public/stock-indicators-sndk.json`:

```json
{
  "result": {
    "indicatorSections": [
      {
        "sectionName": "가치평가",
        "data": {
          "displayPer": "45.4배",
          "displayPbr": "14.9배",
          "displayPsr": "15.5배"
        }
      },
      {
        "sectionName": "수익",
        "data": {
          "eps": 28.76,
          "epsKrw": 42904,
          "bps": 93.08,
          "bpsKrw": 138856,
          "roe": "39.3%"
        }
      },
      {
        "sectionName": "배당",
        "data": {
          "dividendFrequency": null,
          "dividendYieldRatio": 0,
          "currency": "USD",
          "annualCash": null,
          "annualCashKrw": null,
          "months": [],
          "exDate": null,
          "lastYear": 2025,
          "lastYearDividendInfo": {
            "dividendCount": 0,
            "dividendMonths": [],
            "dividendCash": 0,
            "dividendCashKrw": null,
            "dividendYieldRatio": 0,
            "ttmDividendYieldRatio": 0,
            "ttmDividendMonths": [],
            "ttmDps": 0.00,
            "ttmDpsKrw": null,
            "ttmDividendTotalCount": 0,
            "dividendGrowthRatio": null,
            "currency": null
          }
        }
      }
    ]
  }
}
```

- [ ] **Step 2: Update manifest**

In `fixtures/responses/public/manifest.json`, append before the closing `]`:

```json
,
{
  "file": "stock-indicators-sndk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/stock-detail/ui/wts/NAS0250224006/investment-indicators",
  "method": "GET"
}
```

- [ ] **Step 3: Add domain types**

Append to `internal/domain/models.go`:

```go
// StockIndicators is the aggregated valuation/earnings/dividend/stability
// snapshot returned by /api/v1/stock-detail/ui/wts/{code}/investment-indicators.
// Each section is the raw map of keys returned by Toss for that block; callers
// route by section name.
type StockIndicators struct {
	ProductCode string                     `json:"productCode"`
	Sections    map[string]IndicatorFields `json:"sections"` // key: 가치평가|수익|배당|안정성
	FetchedAt   time.Time                  `json:"fetchedAt"`
}

// IndicatorFields holds the raw per-section payload as decoded JSON.
type IndicatorFields map[string]any
```

(`time` already imported in this file from PR8.)

- [ ] **Step 4: Write the failing client test**

Create `internal/client/indicators_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetStockIndicatorsFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-indicators-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stock-detail/ui/wts/NAS0250224006/investment-indicators" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ind, err := c.GetStockIndicators(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockIndicators error: %v", err)
	}
	val, ok := ind.Sections["가치평가"]
	if !ok {
		t.Fatalf("missing 가치평가 section")
	}
	if val["displayPer"] != "45.4배" {
		t.Fatalf("unexpected PER: %v", val["displayPer"])
	}
	earnings := ind.Sections["수익"]
	if earnings["roe"] != "39.3%" {
		t.Fatalf("unexpected ROE: %v", earnings["roe"])
	}
}
```

- [ ] **Step 5: Run test (fails — compile error)**

Run: `go test ./internal/client/ -run TestGetStockIndicators -v`
Expected: FAIL with "undefined: c.GetStockIndicators".

- [ ] **Step 6: Implement client method**

Create `internal/client/indicators.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type indicatorsEnvelope struct {
	Result struct {
		IndicatorSections []struct {
			SectionName string                  `json:"sectionName"`
			Data        domain.IndicatorFields  `json:"data"`
		} `json:"indicatorSections"`
	} `json:"result"`
}

// GetStockIndicators fetches the investment-indicators payload (가치평가/수익/
// 배당/안정성). Accepts symbol or productCode.
func (c *Client) GetStockIndicators(ctx context.Context, symbol string) (domain.StockIndicators, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockIndicators{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/investment-indicators", c.infoBaseURL, productCode)
	var env indicatorsEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.StockIndicators{}, err
	}
	sections := make(map[string]domain.IndicatorFields, len(env.Result.IndicatorSections))
	for _, s := range env.Result.IndicatorSections {
		sections[s.SectionName] = s.Data
	}
	return domain.StockIndicators{
		ProductCode: productCode,
		Sections:    sections,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 7: Run test (passes)**

Run: `go test ./internal/client/ -run TestGetStockIndicators -v`
Expected: PASS.

- [ ] **Step 8: Write the output writer test**

Create `internal/output/indicators_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockIndicatorsTable(t *testing.T) {
	t.Parallel()
	ind := domain.StockIndicators{
		ProductCode: "NAS0250224006",
		Sections: map[string]domain.IndicatorFields{
			"가치평가": {"displayPer": "45.4배", "displayPbr": "14.9배", "displayPsr": "15.5배"},
			"수익":     {"eps": 28.76, "epsKrw": float64(42904), "bps": 93.08, "bpsKrw": float64(138856), "roe": "39.3%"},
			"배당":     {"dividendYieldRatio": float64(0), "annualCash": nil},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockIndicators(&buf, FormatTable, ind); err != nil {
		t.Fatalf("WriteStockIndicators error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "가치평가", "PER", "45.4배", "수익", "EPS", "$28.76", "ROE", "39.3%", "배당", "0.00%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 9: Run output test (fails — compile)**

Run: `go test ./internal/output/ -run TestWriteStockIndicators -v`
Expected: FAIL with "undefined: WriteStockIndicators".

- [ ] **Step 10: Implement output writer**

Create `internal/output/indicators.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockIndicators renders the investment-indicators payload as a sectioned
// table (one block per sectionName), full JSON, or CSV (section,key,value).
func WriteStockIndicators(w io.Writer, format Format, ind domain.StockIndicators) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ind)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "section,key,value"); err != nil {
			return err
		}
		names := sortedSectionNames(ind.Sections)
		for _, name := range names {
			keys := sortedMapKeys(ind.Sections[name])
			for _, k := range keys {
				if _, err := fmt.Fprintf(w, "%s,%s,%v\n", name, k, ind.Sections[name][k]); err != nil {
					return err
				}
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — investment indicators\n", ind.ProductCode); err != nil {
			return err
		}
		order := []string{"가치평가", "수익", "배당", "안정성"}
		for _, name := range order {
			fields, ok := ind.Sections[name]
			if !ok {
				continue
			}
			if _, err := fmt.Fprintf(w, "\n=== %s ===\n", name); err != nil {
				return err
			}
			if err := renderIndicatorSection(w, name, fields); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func renderIndicatorSection(w io.Writer, name string, fields domain.IndicatorFields) error {
	type kv struct{ Label, Value string }
	var pairs []kv

	switch name {
	case "가치평가":
		pairs = []kv{
			{"PER", strVal(fields["displayPer"])},
			{"PBR", strVal(fields["displayPbr"])},
			{"PSR", strVal(fields["displayPsr"])},
		}
	case "수익":
		pairs = []kv{
			{"EPS", fmtUSDKRW(fields["eps"], fields["epsKrw"])},
			{"BPS", fmtUSDKRW(fields["bps"], fields["bpsKrw"])},
			{"ROE", strVal(fields["roe"])},
		}
	case "배당":
		pairs = []kv{
			{"배당 주기", strOrDash(fields["dividendFrequency"])},
			{"배당 수익률", fmtPctNum(fields["dividendYieldRatio"])},
			{"연간 배당금", fmtUSDKRWOrDash(fields["annualCash"], fields["annualCashKrw"])},
		}
	default:
		keys := sortedMapKeys(fields)
		for _, k := range keys {
			pairs = append(pairs, kv{k, strVal(fields[k])})
		}
	}

	headers := []string{"FIELD", "VALUE"}
	rows := make([][]string, len(pairs))
	for i, p := range pairs {
		rows[i] = []string{p.Label, p.Value}
	}
	return renderTable(w, headers, rows)
}

func strVal(v any) string {
	if v == nil {
		return "—"
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return fmt.Sprintf("%d", int64(s))
		}
		return fmt.Sprintf("%.2f", s)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func strOrDash(v any) string {
	if v == nil {
		return "—"
	}
	return strVal(v)
}

func fmtUSDKRW(usdAny, krwAny any) string {
	usd, _ := usdAny.(float64)
	krw, _ := krwAny.(float64)
	return fmt.Sprintf("%s (₩%s)", formatUSD(usd), formatWithCommas(int64(krw)))
}

func fmtUSDKRWOrDash(usdAny, krwAny any) string {
	if usdAny == nil && krwAny == nil {
		return "—"
	}
	return fmtUSDKRW(usdAny, krwAny)
}

func fmtPctNum(v any) string {
	switch f := v.(type) {
	case float64:
		return fmt.Sprintf("%.2f%%", f)
	default:
		return strVal(v)
	}
}

func sortedSectionNames(m map[string]domain.IndicatorFields) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys(m domain.IndicatorFields) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 11: Run output test (passes)**

Run: `go test ./internal/output/ -run TestWriteStockIndicators -v`
Expected: PASS.

- [ ] **Step 12: Register cobra subcommand**

In `cmd/tossctl/stock.go`, after the `overviewCmd` block (before `cmd.AddCommand`), add:

```go
indicatorsCmd := &cobra.Command{
    Use:   "indicators <symbol>",
    Short: "Show investment indicators (가치평가/수익/배당/안정성)",
    Long: `Fetch the investment indicators payload from
/api/v1/stock-detail/ui/wts/{code}/investment-indicators.

Returns four sectioned blocks: 가치평가 (PER/PBR/PSR), 수익 (EPS/BPS/ROE),
배당 (frequency, yield, annual cash), 안정성 (debt/current ratios).

Examples:
  tossctl stock indicators SNDK
  tossctl stock indicators NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        ind, err := app.client.GetStockIndicators(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockIndicators(cmd.OutOrStdout(), app.format, ind)
    },
}
```

Update the `cmd.AddCommand` line to include `indicatorsCmd`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd)
```

- [ ] **Step 13: Full build + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add internal/domain/models.go internal/client/indicators.go internal/client/indicators_test.go internal/output/indicators.go internal/output/indicators_test.go cmd/tossctl/stock.go fixtures/responses/public/stock-indicators-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(cli): add tossctl stock indicators (PER/PBR/PSR + EPS/BPS/ROE + dividend)"
```

---

## Task 3: `tossctl stock valuation <sym>`

**Files:**
- Create: `internal/client/valuation.go` + `_test.go`
- Create: `internal/output/valuation.go` + `_test.go`
- Create: `fixtures/responses/public/stock-valuation-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `fixtures/responses/public/manifest.json`

Two POST endpoints (`evaluation/{code}` + `evaluation-comparison/{code}`), both with body `{}`. Test fixture bundles both responses under `{"evaluation": {…}, "comparison": {…}}` so one file feeds both halves of the test.

- [ ] **Step 1: Save fixture**

Create `fixtures/responses/public/stock-valuation-sndk.json`:

```json
{
  "evaluation": {
    "result": {
      "per": 45.4,
      "pbr": 14.9,
      "psr": 15.5,
      "median": 26.52350207310445,
      "position": "HIGH"
    }
  },
  "comparison": {
    "result": {
      "selectedFactor": {"code": "PER", "displayName": "PER"},
      "selectedTics": {"code": 209, "displayName": "컴퓨터와 주변기기"},
      "ticsStocks": [
        {"code": "US19990122001", "name": "엔비디아", "isDefault": false},
        {"code": "US19801212001", "name": "애플", "isDefault": false},
        {"code": "US19890516001", "name": "마이크론 테크놀로지", "isDefault": false},
        {"code": "NAS0250224006", "name": "샌디스크", "isDefault": true},
        {"code": "US20021211002", "name": "씨게이트", "isDefault": false}
      ],
      "stockGraphs": [
        {"code": "US19990122001", "name": "엔비디아", "graph": [{"period": "26년 1분기", "value": 47.55}]},
        {"code": "US19801212001", "name": "애플", "graph": [{"period": "26년 1분기", "value": 35.73}]},
        {"code": "US19890516001", "name": "마이크론 테크놀로지", "graph": [{"period": "26년 1분기", "value": 36.30}]},
        {"code": "NAS0250224006", "name": "샌디스크", "graph": [{"period": "26년 1분기", "value": 45.4}]},
        {"code": "US20021211002", "name": "씨게이트", "graph": [{"period": "26년 1분기", "value": 22.1}]}
      ]
    }
  }
}
```

- [ ] **Step 2: Update manifest**

Append to `fixtures/responses/public/manifest.json`:

```json
,
{
  "file": "stock-valuation-sndk.json",
  "url": "(combined evaluation+comparison fixture for testing only)",
  "method": "POST"
}
```

- [ ] **Step 3: Add domain types**

Append to `internal/domain/models.go`:

```go
// StockValuation aggregates the per-stock valuation snapshot (evaluation) and
// peer comparison matrix (evaluation-comparison).
type StockValuation struct {
	ProductCode string          `json:"productCode"`
	PER, PBR, PSR float64       `json:"per,pbr,psr"`
	Median       float64        `json:"median"`        // industry median for SelectedFactor
	Position     string         `json:"position"`      // HIGH|LOW|NORMAL
	Factor       string         `json:"factor"`        // PER (default), PBR, PSR, …
	Industry     string         `json:"industry"`      // selectedTics displayName
	Peers        []PeerValuation `json:"peers"`
	FetchedAt    time.Time      `json:"fetchedAt"`
}

// PeerValuation is one row in the peer comparison table. `Value` is the most
// recent graph point for the selected factor.
type PeerValuation struct {
	ProductCode string  `json:"productCode"`
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Period      string  `json:"period"`
	IsSelf      bool    `json:"isSelf"`
}
```

- [ ] **Step 4: Write the failing client test**

Create `internal/client/valuation_test.go`:

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

func TestGetStockValuationFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Evaluation json.RawMessage `json:"evaluation"`
		Comparison json.RawMessage `json:"comparison"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-valuation-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/evaluation/NAS0250224006":
			w.Write(bundle.Evaluation)
		case "/api/v2/stock-infos/evaluation-comparison/NAS0250224006":
			w.Write(bundle.Comparison)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	val, err := c.GetStockValuation(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockValuation error: %v", err)
	}
	if val.PER != 45.4 {
		t.Fatalf("expected PER 45.4, got %v", val.PER)
	}
	if val.Position != "HIGH" {
		t.Fatalf("expected HIGH, got %q", val.Position)
	}
	if val.Industry != "컴퓨터와 주변기기" {
		t.Fatalf("unexpected industry: %q", val.Industry)
	}
	if len(val.Peers) != 5 {
		t.Fatalf("expected 5 peers, got %d", len(val.Peers))
	}
	var self *PeerValuationView
	for i := range val.Peers {
		if val.Peers[i].IsSelf {
			p := val.Peers[i]
			self = &PeerValuationView{Code: p.ProductCode, Value: p.Value}
		}
	}
	if self == nil || self.Code != "NAS0250224006" {
		t.Fatalf("expected self row marked; got %+v", self)
	}
}

type PeerValuationView struct {
	Code  string
	Value float64
}
```

- [ ] **Step 5: Run test (fails)**

Run: `go test ./internal/client/ -run TestGetStockValuation -v`
Expected: FAIL with "undefined: c.GetStockValuation".

- [ ] **Step 6: Implement client method**

Create `internal/client/valuation.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type evaluationEnvelope struct {
	Result struct {
		PER, PBR, PSR float64 `json:"per,pbr,psr"`
		Median        float64 `json:"median"`
		Position      string  `json:"position"`
	} `json:"result"`
}

type evaluationComparisonEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code string `json:"code"`
		} `json:"selectedFactor"`
		SelectedTics struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedTics"`
		TicsStocks []struct {
			Code      string `json:"code"`
			Name      string `json:"name"`
			IsDefault bool   `json:"isDefault"`
		} `json:"ticsStocks"`
		StockGraphs []struct {
			Code  string `json:"code"`
			Name  string `json:"name"`
			Graph []struct {
				Period string  `json:"period"`
				Value  float64 `json:"value"`
			} `json:"graph"`
		} `json:"stockGraphs"`
	} `json:"result"`
}

// GetStockValuation fetches the per-stock valuation snapshot and peer
// comparison matrix. Two POST calls (both with empty body) are stitched into
// a single StockValuation result.
func (c *Client) GetStockValuation(ctx context.Context, symbol string) (domain.StockValuation, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockValuation{}, err
	}

	evalEndpoint := fmt.Sprintf("%s/api/v2/stock-infos/evaluation/%s", c.infoBaseURL, productCode)
	var evalEnv evaluationEnvelope
	if err := c.postJSONEmpty(ctx, evalEndpoint, &evalEnv); err != nil {
		return domain.StockValuation{}, err
	}

	cmpEndpoint := fmt.Sprintf("%s/api/v2/stock-infos/evaluation-comparison/%s", c.infoBaseURL, productCode)
	var cmpEnv evaluationComparisonEnvelope
	if err := c.postJSONEmpty(ctx, cmpEndpoint, &cmpEnv); err != nil {
		return domain.StockValuation{}, err
	}

	peers := make([]domain.PeerValuation, 0, len(cmpEnv.Result.StockGraphs))
	for _, g := range cmpEnv.Result.StockGraphs {
		var value float64
		var period string
		if n := len(g.Graph); n > 0 {
			value = g.Graph[n-1].Value
			period = g.Graph[n-1].Period
		}
		peers = append(peers, domain.PeerValuation{
			ProductCode: g.Code,
			Name:        g.Name,
			Value:       value,
			Period:      period,
			IsSelf:      g.Code == productCode,
		})
	}

	return domain.StockValuation{
		ProductCode: productCode,
		PER:         evalEnv.Result.PER,
		PBR:         evalEnv.Result.PBR,
		PSR:         evalEnv.Result.PSR,
		Median:      evalEnv.Result.Median,
		Position:    evalEnv.Result.Position,
		Factor:      cmpEnv.Result.SelectedFactor.Code,
		Industry:    cmpEnv.Result.SelectedTics.DisplayName,
		Peers:       peers,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 7: Run test (passes)**

Run: `go test ./internal/client/ -run TestGetStockValuation -v`
Expected: PASS.

- [ ] **Step 8: Write the output writer test**

Create `internal/output/valuation_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockValuationTable(t *testing.T) {
	t.Parallel()
	val := domain.StockValuation{
		ProductCode: "NAS0250224006",
		PER:         45.4, PBR: 14.9, PSR: 15.5,
		Median:   26.52,
		Position: "HIGH",
		Factor:   "PER",
		Industry: "컴퓨터와 주변기기",
		Peers: []domain.PeerValuation{
			{ProductCode: "US19990122001", Name: "엔비디아", Value: 47.55, Period: "26년 1분기"},
			{ProductCode: "NAS0250224006", Name: "샌디스크", Value: 45.4, Period: "26년 1분기", IsSelf: true},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockValuation(&buf, FormatTable, val); err != nil {
		t.Fatalf("WriteStockValuation error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"PER 45.40", "median 26.52", "HIGH", "PBR 14.90", "PSR 15.50", "컴퓨터와 주변기기", "엔비디아", "샌디스크", "← self"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 9: Run output test (fails)**

Run: `go test ./internal/output/ -run TestWriteStockValuation -v`
Expected: FAIL with "undefined: WriteStockValuation".

- [ ] **Step 10: Implement output writer**

Create `internal/output/valuation.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockValuation renders the per-stock valuation snapshot + peer table.
func WriteStockValuation(w io.Writer, format Format, val domain.StockValuation) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(val)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "product_code,name,factor,value,period,is_self"); err != nil {
			return err
		}
		for _, p := range val.Peers {
			if _, err := fmt.Fprintf(w, "%s,%s,%s,%.2f,%s,%t\n", p.ProductCode, p.Name, val.Factor, p.Value, p.Period, p.IsSelf); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — valuation snapshot\n", val.ProductCode); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "PER %.2f vs median %.2f  →  %s\n", val.PER, val.Median, val.Position); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "PBR %.2f   PSR %.2f\n", val.PBR, val.PSR); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "\n동종업계 비교 (%s 기준, %s)\n", val.Factor, val.Industry); err != nil {
			return err
		}
		headers := []string{"SYMBOL", "NAME", val.Factor, "PERIOD", ""}
		rows := make([][]string, len(val.Peers))
		for i, p := range val.Peers {
			marker := ""
			if p.IsSelf {
				marker = "← self"
			}
			rows[i] = []string{p.ProductCode, p.Name, fmt.Sprintf("%.2f", p.Value), p.Period, marker}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 11: Run output test (passes)**

Run: `go test ./internal/output/ -run TestWriteStockValuation -v`
Expected: PASS.

- [ ] **Step 12: Register cobra subcommand**

In `cmd/tossctl/stock.go`, after `indicatorsCmd`, append:

```go
valuationCmd := &cobra.Command{
    Use:   "valuation <symbol>",
    Short: "Per-stock PER/PBR/PSR vs industry median + peer table",
    Long: `Fetch the valuation snapshot and peer comparison from
/api/v2/stock-infos/evaluation/{code} and
/api/v2/stock-infos/evaluation-comparison/{code} (both POST {}).

Output shows PER/PBR/PSR plus the industry median and HIGH/LOW/NORMAL
position label, followed by the peer table (5 stocks) for the
selected factor.

Examples:
  tossctl stock valuation SNDK
  tossctl stock valuation NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        val, err := app.client.GetStockValuation(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockValuation(cmd.OutOrStdout(), app.format, val)
    },
}
```

Update `cmd.AddCommand` to include `valuationCmd`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd)
```

- [ ] **Step 13: Full build + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add internal/domain/models.go internal/client/valuation.go internal/client/valuation_test.go internal/output/valuation.go internal/output/valuation_test.go cmd/tossctl/stock.go fixtures/responses/public/stock-valuation-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(cli): add tossctl stock valuation (PER/PBR/PSR vs median + peer table)"
```

---

## Task 4: `tossctl stock revenue <sym>`

**Files:**
- Create: `internal/client/revenue.go` + `_test.go`
- Create: `internal/output/revenue.go` + `_test.go`
- Create: `fixtures/responses/public/sales-compositions-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `fixtures/responses/public/manifest.json`

Endpoint: `GET /api/v1/companies/{companyCode}/sales-compositions`. Uses companyCode (e.g. `NAS116LTR-E0`), resolved via overview.

- [ ] **Step 1: Save fixture**

Create `fixtures/responses/public/sales-compositions-sndk.json`:

```json
{
  "result": {
    "code": "NAS116LTR-E0",
    "fiscalYear": 2025,
    "endDate": "2025-06-30",
    "compositions": [
      {"business": "클라이언트", "jpBusiness": null, "product": null, "ratio": 56.11},
      {"business": "소비자", "jpBusiness": null, "product": null, "ratio": 30.84},
      {"business": "클라우드 서비스", "jpBusiness": null, "product": null, "ratio": 13.05}
    ],
    "dataSource": "출처: 연합인포맥스 및 기업 IR자료"
  }
}
```

- [ ] **Step 2: Update manifest**

```json
,
{
  "file": "sales-compositions-sndk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/companies/NAS116LTR-E0/sales-compositions",
  "method": "GET"
}
```

- [ ] **Step 3: Add domain types**

Append to `internal/domain/models.go`:

```go
type SalesComposition struct {
	ProductCode string                  `json:"productCode"`
	CompanyCode string                  `json:"companyCode"`
	FiscalYear  int                     `json:"fiscalYear"`
	EndDate     string                  `json:"endDate"`
	Items       []SalesCompositionItem  `json:"items"`
	DataSource  string                  `json:"dataSource"`
	FetchedAt   time.Time               `json:"fetchedAt"`
}

type SalesCompositionItem struct {
	Business string  `json:"business"`
	Product  string  `json:"product,omitempty"`
	Ratio    float64 `json:"ratio"`
}
```

- [ ] **Step 4: Write the failing client test**

Create `internal/client/revenue_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetSalesCompositionFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	sales := mustReadFile(t, filepath.Join(root, "sales-compositions-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v1/companies/NAS116LTR-E0/sales-compositions":
			w.Write(sales)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	sc, err := c.GetSalesComposition(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetSalesComposition error: %v", err)
	}
	if sc.CompanyCode != "NAS116LTR-E0" {
		t.Fatalf("expected companyCode NAS116LTR-E0, got %q", sc.CompanyCode)
	}
	if sc.FiscalYear != 2025 {
		t.Fatalf("expected FY2025, got %d", sc.FiscalYear)
	}
	if len(sc.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sc.Items))
	}
	if sc.Items[0].Business != "클라이언트" || sc.Items[0].Ratio != 56.11 {
		t.Fatalf("unexpected first item: %+v", sc.Items[0])
	}
}
```

This depends on `stock-overview-sndk.json` already existing (PR8). Verify the fixture has `result.company.code == "NAS116LTR-E0"`. If different, update the company-code mapping above.

- [ ] **Step 5: Run test (fails)**

Run: `go test ./internal/client/ -run TestGetSalesComposition -v`
Expected: FAIL with "undefined: c.GetSalesComposition".

- [ ] **Step 6: Implement client method**

Create `internal/client/revenue.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type salesCompositionEnvelope struct {
	Result struct {
		Code       string `json:"code"`
		FiscalYear int    `json:"fiscalYear"`
		EndDate    string `json:"endDate"`
		Compositions []struct {
			Business string  `json:"business"`
			Product  string  `json:"product"`
			Ratio    float64 `json:"ratio"`
		} `json:"compositions"`
		DataSource string `json:"dataSource"`
	} `json:"result"`
}

// GetSalesComposition fetches the revenue-composition breakdown. The endpoint
// uses companyCode (NAS116LTR-E0 form) which is resolved via the overview call.
func (c *Client) GetSalesComposition(ctx context.Context, symbol string) (domain.SalesComposition, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.SalesComposition{}, err
	}
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return domain.SalesComposition{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/companies/%s/sales-compositions", c.infoBaseURL, companyCode)
	var env salesCompositionEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.SalesComposition{}, err
	}
	items := make([]domain.SalesCompositionItem, len(env.Result.Compositions))
	for i, comp := range env.Result.Compositions {
		items[i] = domain.SalesCompositionItem{
			Business: comp.Business,
			Product:  comp.Product,
			Ratio:    comp.Ratio,
		}
	}
	return domain.SalesComposition{
		ProductCode: productCode,
		CompanyCode: env.Result.Code,
		FiscalYear:  env.Result.FiscalYear,
		EndDate:     env.Result.EndDate,
		Items:       items,
		DataSource:  env.Result.DataSource,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 7: Run test (passes)**

Run: `go test ./internal/client/ -run TestGetSalesComposition -v`
Expected: PASS.

- [ ] **Step 8: Write output writer test**

Create `internal/output/revenue_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteSalesCompositionTable(t *testing.T) {
	t.Parallel()
	sc := domain.SalesComposition{
		ProductCode: "NAS0250224006",
		CompanyCode: "NAS116LTR-E0",
		FiscalYear:  2025,
		EndDate:     "2025-06-30",
		Items: []domain.SalesCompositionItem{
			{Business: "클라이언트", Ratio: 56.11},
			{Business: "소비자", Ratio: 30.84},
			{Business: "클라우드 서비스", Ratio: 13.05},
		},
		DataSource: "출처: 연합인포맥스 및 기업 IR자료",
		FetchedAt:  time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteSalesComposition(&buf, FormatTable, sc); err != nil {
		t.Fatalf("WriteSalesComposition error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "FY2025", "2025-06-30", "클라이언트", "56.11", "소비자", "30.84", "클라우드 서비스", "13.05", "연합인포맥스"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 9: Run output test (fails)**

Run: `go test ./internal/output/ -run TestWriteSalesComposition -v`
Expected: FAIL.

- [ ] **Step 10: Implement output writer**

Create `internal/output/revenue.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteSalesComposition renders the revenue composition payload.
func WriteSalesComposition(w io.Writer, format Format, sc domain.SalesComposition) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(sc)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "business,product,ratio"); err != nil {
			return err
		}
		for _, it := range sc.Items {
			if _, err := fmt.Fprintf(w, "%s,%s,%.2f\n", it.Business, it.Product, it.Ratio); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — 매출 구성 (FY%d, ending %s)\n", sc.ProductCode, sc.FiscalYear, sc.EndDate); err != nil {
			return err
		}
		headers := []string{"BUSINESS", "PRODUCT", "RATIO"}
		rows := make([][]string, len(sc.Items))
		for i, it := range sc.Items {
			rows[i] = []string{it.Business, it.Product, fmt.Sprintf("%.2f%%", it.Ratio)}
		}
		if err := renderTable(w, headers, rows); err != nil {
			return err
		}
		if sc.DataSource != "" {
			_, err := fmt.Fprintf(w, "%s\n", sc.DataSource)
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 11: Run output test (passes)**

Run: `go test ./internal/output/ -run TestWriteSalesComposition -v`
Expected: PASS.

- [ ] **Step 12: Register cobra subcommand**

In `cmd/tossctl/stock.go`, after `valuationCmd`:

```go
revenueCmd := &cobra.Command{
    Use:   "revenue <symbol>",
    Short: "Revenue composition by business segment (latest fiscal period)",
    Long: `Fetch revenue composition from
/api/v1/companies/{companyCode}/sales-compositions.

The companyCode (e.g. NAS116LTR-E0) is resolved internally via the
company-overview call.

Examples:
  tossctl stock revenue SNDK
  tossctl stock revenue NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        sc, err := app.client.GetSalesComposition(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteSalesComposition(cmd.OutOrStdout(), app.format, sc)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd)
```

- [ ] **Step 13: Full build + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add internal/domain/models.go internal/client/revenue.go internal/client/revenue_test.go internal/output/revenue.go internal/output/revenue_test.go cmd/tossctl/stock.go fixtures/responses/public/sales-compositions-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(cli): add tossctl stock revenue (sales composition by business)"
```

---

## Task 5: `tossctl stock peers <sym>`

**Files:**
- Create: `internal/client/peers.go` + `_test.go`
- Create: `internal/output/peers.go` + `_test.go`
- Create: `fixtures/responses/public/tics-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `fixtures/responses/public/manifest.json`

Endpoint: `GET /api/v2/companies/{companyCode}/tics`.

- [ ] **Step 1: Save fixture**

Create `fixtures/responses/public/tics-sndk.json`:

```json
{
  "result": {
    "baseDate": "2020-12-01T00:00:00",
    "majorList": [
      {
        "id": 209,
        "title": "컴퓨터와 주변기기",
        "imageUrl": "https://static.toss.im/ml-product/computer-speaker-monitor-area.png",
        "ordering": 1,
        "description": "sd카드, usb 메모리 등 판매",
        "representative": true,
        "companyCount": 85,
        "rankings": [
          {
            "baseDate": "2026-05-16",
            "baseDateTime": "2026-05-16T03:50:46",
            "fiscalPeriod": "Q",
            "type": {"code": 1, "displayName": "시가총액"},
            "ranking": 5,
            "companyCount": 45,
            "ticsId": 209,
            "displayValue": "310조 6,978억",
            "value": 3.1069789289844044E14
          },
          {
            "baseDate": "2026-04-03",
            "baseDateTime": "2026-04-03T08:00:00",
            "fiscalPeriod": "Q",
            "type": {"code": 3, "displayName": "매출"},
            "ranking": 8,
            "companyCount": 45,
            "ticsId": 209,
            "displayValue": "20조 238억",
            "value": 2.00238592E13
          },
          {
            "baseDate": "2026-04-03",
            "baseDateTime": "2026-04-03T08:00:00",
            "fiscalPeriod": "Q",
            "type": {"code": 4, "displayName": "영업이익률"},
            "ranking": 3,
            "companyCount": 45,
            "ticsId": 209,
            "displayValue": "40.7%",
            "value": 40.731189
          }
        ]
      }
    ],
    "minorList": []
  }
}
```

- [ ] **Step 2: Update manifest**

```json
,
{
  "file": "tics-sndk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/companies/NAS116LTR-E0/tics",
  "method": "GET"
}
```

- [ ] **Step 3: Add domain types**

Append to `internal/domain/models.go`:

```go
type TICSIndustry struct {
	ProductCode string       `json:"productCode"`
	CompanyCode string       `json:"companyCode"`
	BaseDate    string       `json:"baseDate"`
	Major       []TICSEntry  `json:"major"`
	Minor       []TICSEntry  `json:"minor"`
	FetchedAt   time.Time    `json:"fetchedAt"`
}

type TICSEntry struct {
	ID             int               `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	CompanyCount   int               `json:"companyCount"`
	Representative bool              `json:"representative"`
	Rankings       []TICSRanking     `json:"rankings"`
}

type TICSRanking struct {
	BaseDate     string  `json:"baseDate"`
	FiscalPeriod string  `json:"fiscalPeriod"`
	TypeName     string  `json:"typeName"`     // 시가총액|매출|영업이익률
	Ranking      int     `json:"ranking"`
	CompanyCount int     `json:"companyCount"`
	DisplayValue string  `json:"displayValue"`
	Value        float64 `json:"value"`
}
```

- [ ] **Step 4: Write the failing client test**

Create `internal/client/peers_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetTICSIndustryFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	tics := mustReadFile(t, filepath.Join(root, "tics-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v2/companies/NAS116LTR-E0/tics":
			w.Write(tics)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	ind, err := c.GetTICSIndustry(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetTICSIndustry error: %v", err)
	}
	if len(ind.Major) != 1 {
		t.Fatalf("expected 1 major entry, got %d", len(ind.Major))
	}
	if ind.Major[0].Title != "컴퓨터와 주변기기" {
		t.Fatalf("unexpected title: %q", ind.Major[0].Title)
	}
	if ind.Major[0].CompanyCount != 85 {
		t.Fatalf("expected companyCount 85, got %d", ind.Major[0].CompanyCount)
	}
	if len(ind.Major[0].Rankings) != 3 {
		t.Fatalf("expected 3 rankings, got %d", len(ind.Major[0].Rankings))
	}
	if ind.Major[0].Rankings[0].TypeName != "시가총액" {
		t.Fatalf("unexpected first ranking type: %q", ind.Major[0].Rankings[0].TypeName)
	}
}
```

- [ ] **Step 5: Run test (fails)**

Run: `go test ./internal/client/ -run TestGetTICSIndustry -v`
Expected: FAIL.

- [ ] **Step 6: Implement client method**

Create `internal/client/peers.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type ticsEnvelope struct {
	Result struct {
		BaseDate  string       `json:"baseDate"`
		MajorList []ticsEntry  `json:"majorList"`
		MinorList []ticsEntry  `json:"minorList"`
	} `json:"result"`
}

type ticsEntry struct {
	ID             int            `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	CompanyCount   int            `json:"companyCount"`
	Representative bool           `json:"representative"`
	Rankings       []ticsRanking  `json:"rankings"`
}

type ticsRanking struct {
	BaseDate     string  `json:"baseDate"`
	FiscalPeriod string  `json:"fiscalPeriod"`
	Type         struct {
		DisplayName string `json:"displayName"`
	} `json:"type"`
	Ranking      int     `json:"ranking"`
	CompanyCount int     `json:"companyCount"`
	DisplayValue string  `json:"displayValue"`
	Value        float64 `json:"value"`
}

// GetTICSIndustry fetches the TICS industry taxonomy and peer rankings.
func (c *Client) GetTICSIndustry(ctx context.Context, symbol string) (domain.TICSIndustry, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.TICSIndustry{}, err
	}
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return domain.TICSIndustry{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/tics", c.infoBaseURL, companyCode)
	var env ticsEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.TICSIndustry{}, err
	}
	return domain.TICSIndustry{
		ProductCode: productCode,
		CompanyCode: companyCode,
		BaseDate:    env.Result.BaseDate,
		Major:       convertTICSEntries(env.Result.MajorList),
		Minor:       convertTICSEntries(env.Result.MinorList),
		FetchedAt:   time.Now().UTC(),
	}, nil
}

func convertTICSEntries(src []ticsEntry) []domain.TICSEntry {
	out := make([]domain.TICSEntry, len(src))
	for i, e := range src {
		ranks := make([]domain.TICSRanking, len(e.Rankings))
		for j, r := range e.Rankings {
			ranks[j] = domain.TICSRanking{
				BaseDate:     r.BaseDate,
				FiscalPeriod: r.FiscalPeriod,
				TypeName:     r.Type.DisplayName,
				Ranking:      r.Ranking,
				CompanyCount: r.CompanyCount,
				DisplayValue: r.DisplayValue,
				Value:        r.Value,
			}
		}
		out[i] = domain.TICSEntry{
			ID:             e.ID,
			Title:          e.Title,
			Description:    e.Description,
			CompanyCount:   e.CompanyCount,
			Representative: e.Representative,
			Rankings:       ranks,
		}
	}
	return out
}
```

- [ ] **Step 7: Run test (passes)**

Run: `go test ./internal/client/ -run TestGetTICSIndustry -v`
Expected: PASS.

- [ ] **Step 8: Write output writer test**

Create `internal/output/peers_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteTICSIndustryTable(t *testing.T) {
	t.Parallel()
	ind := domain.TICSIndustry{
		ProductCode: "NAS0250224006",
		CompanyCode: "NAS116LTR-E0",
		Major: []domain.TICSEntry{
			{
				ID: 209, Title: "컴퓨터와 주변기기", Description: "sd카드, usb 메모리 등 판매",
				CompanyCount: 85, Representative: true,
				Rankings: []domain.TICSRanking{
					{BaseDate: "2026-05-16", TypeName: "시가총액", Ranking: 5, CompanyCount: 45, DisplayValue: "310조 6,978억"},
					{BaseDate: "2026-04-03", TypeName: "매출", Ranking: 8, CompanyCount: 45, DisplayValue: "20조 238억"},
					{BaseDate: "2026-04-03", TypeName: "영업이익률", Ranking: 3, CompanyCount: 45, DisplayValue: "40.7%"},
				},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteTICSIndustry(&buf, FormatTable, ind); err != nil {
		t.Fatalf("WriteTICSIndustry error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "컴퓨터와 주변기기", "85개사", "시가총액", "5", "310조 6,978억", "매출", "8", "영업이익률", "3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 9: Run output test (fails)**

Run: `go test ./internal/output/ -run TestWriteTICSIndustry -v`
Expected: FAIL.

- [ ] **Step 10: Implement output writer**

Create `internal/output/peers.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteTICSIndustry renders the TICS industry taxonomy + peer rankings.
func WriteTICSIndustry(w io.Writer, format Format, ind domain.TICSIndustry) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ind)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "industry_id,industry,base_date,metric,ranking,company_count,display_value"); err != nil {
			return err
		}
		for _, e := range ind.Major {
			for _, r := range e.Rankings {
				if _, err := fmt.Fprintf(w, "%d,%s,%s,%s,%d,%d,%s\n", e.ID, e.Title, r.BaseDate, r.TypeName, r.Ranking, r.CompanyCount, r.DisplayValue); err != nil {
					return err
				}
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — TICS industry & peer ranking\n", ind.ProductCode); err != nil {
			return err
		}
		for _, e := range ind.Major {
			if _, err := fmt.Fprintf(w, "\n%s (id %d, %d개사)\n", e.Title, e.ID, e.CompanyCount); err != nil {
				return err
			}
			if e.Description != "" {
				if _, err := fmt.Fprintf(w, "  \"%s\"\n", e.Description); err != nil {
					return err
				}
			}
			headers := []string{"METRIC", "RANK", "OUT OF", "VALUE", "AS OF"}
			rows := make([][]string, len(e.Rankings))
			for i, r := range e.Rankings {
				rows[i] = []string{r.TypeName, fmt.Sprintf("%d", r.Ranking), fmt.Sprintf("%d", r.CompanyCount), r.DisplayValue, r.BaseDate}
			}
			if err := renderTable(w, headers, rows); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 11: Run output test (passes)**

Run: `go test ./internal/output/ -run TestWriteTICSIndustry -v`
Expected: PASS.

- [ ] **Step 12: Register cobra subcommand**

In `cmd/tossctl/stock.go`, after `revenueCmd`:

```go
peersCmd := &cobra.Command{
    Use:   "peers <symbol>",
    Short: "TICS industry classification + peer rankings within industry",
    Long: `Fetch the TICS industry taxonomy and per-metric peer rankings
from /api/v2/companies/{companyCode}/tics.

Each industry block lists the company's rank within that industry for
시가총액 / 매출 / 영업이익률 (most recent fiscal period).

Examples:
  tossctl stock peers SNDK
  tossctl stock peers NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        ind, err := app.client.GetTICSIndustry(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteTICSIndustry(cmd.OutOrStdout(), app.format, ind)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd)
```

- [ ] **Step 13: Full build + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add internal/domain/models.go internal/client/peers.go internal/client/peers_test.go internal/output/peers.go internal/output/peers_test.go cmd/tossctl/stock.go fixtures/responses/public/tics-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(cli): add tossctl stock peers (TICS industry + peer rankings)"
```

---

## Task 6: `tossctl stock analyst <sym>`

**Files:**
- Create: `internal/client/analyst.go` + `_test.go`
- Create: `internal/output/analyst.go` + `_test.go`
- Create: `fixtures/responses/public/analyst-snapshot-sndk.json`
- Modify: `internal/domain/models.go`
- Modify: `cmd/tossctl/stock.go`
- Modify: `fixtures/responses/public/manifest.json`

Three endpoints stitched into one snapshot:
- `GET /api/v1/stock-detail/ui/wts/{code}/analyst-opinion`
- `GET /api/v2/stock-infos/consensus/{code}`
- `GET /api/v1/stock-detail/ui/wts/{code}/analyst-reports`

- [ ] **Step 1: Save fixture**

Create `fixtures/responses/public/analyst-snapshot-sndk.json`:

```json
{
  "opinion": {
    "result": {
      "type": "BUY",
      "strongSell": 0, "sell": 0, "hold": 4, "buy": 12, "strongBuy": 6,
      "targetPrice": {"USD": 1224.42, "KRW": 1794999.72},
      "description": "애널리스트 22명 중 18명이 구매 의견을 냈어요."
    }
  },
  "consensus": {
    "result": {
      "targetPrice": {
        "stockCode": "NAS0250224006",
        "mean": 1224.42, "meanKrw": 1794999.72,
        "high": 2000.0, "highKrw": 2932000.0,
        "low": 250.0, "lowKrw": 366500.0,
        "currency": "USD"
      },
      "pointDate": "2026-05-15",
      "pastClosePrices": [
        {"price": 1405.85, "priceKrw": 2097247, "date": "2026-05-15"},
        {"price": 1096.51, "priceKrw": 1635773, "date": "2026-04-30"},
        {"price": 635.34, "priceKrw": 947800, "date": "2026-03-31"}
      ]
    }
  },
  "reports": {
    "result": {"analystReportGroups": []}
  }
}
```

- [ ] **Step 2: Update manifest**

```json
,
{
  "file": "analyst-snapshot-sndk.json",
  "url": "(combined opinion+consensus+reports fixture for testing only)",
  "method": "GET"
}
```

- [ ] **Step 3: Add domain types**

Append to `internal/domain/models.go`:

```go
type AnalystSnapshot struct {
	ProductCode string             `json:"productCode"`
	Opinion     AnalystOpinion     `json:"opinion"`
	Consensus   ConsensusTarget    `json:"consensus"`
	Reports     []AnalystReport    `json:"reports"`
	FetchedAt   time.Time          `json:"fetchedAt"`
}

type AnalystOpinion struct {
	Type        string  `json:"type"`        // BUY|HOLD|SELL
	StrongBuy   int     `json:"strongBuy"`
	Buy         int     `json:"buy"`
	Hold        int     `json:"hold"`
	Sell        int     `json:"sell"`
	StrongSell  int     `json:"strongSell"`
	TargetUSD   float64 `json:"targetUSD"`
	TargetKRW   float64 `json:"targetKRW"`
	Description string  `json:"description"`
}

type ConsensusTarget struct {
	Mean, High, Low                float64                `json:"mean,high,low"`
	MeanKRW, HighKRW, LowKRW       float64                `json:"meanKRW,highKRW,lowKRW"`
	Currency                       string                 `json:"currency"`
	PointDate                      string                 `json:"pointDate"`
	PastCloses                     []ConsensusPastClose   `json:"pastCloses"`
}

type ConsensusPastClose struct {
	Date     string  `json:"date"`
	Price    float64 `json:"price"`
	PriceKRW float64 `json:"priceKRW"`
}

type AnalystReport struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Date   string `json:"date"`
	URL    string `json:"url,omitempty"`
}
```

- [ ] **Step 4: Write the failing client test**

Create `internal/client/analyst_test.go`:

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

func TestGetAnalystSnapshotFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Opinion   json.RawMessage `json:"opinion"`
		Consensus json.RawMessage `json:"consensus"`
		Reports   json.RawMessage `json:"reports"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "analyst-snapshot-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-detail/ui/wts/NAS0250224006/analyst-opinion":
			w.Write(bundle.Opinion)
		case "/api/v2/stock-infos/consensus/NAS0250224006":
			w.Write(bundle.Consensus)
		case "/api/v1/stock-detail/ui/wts/NAS0250224006/analyst-reports":
			w.Write(bundle.Reports)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	snap, err := c.GetAnalystSnapshot(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetAnalystSnapshot error: %v", err)
	}
	if snap.Opinion.Type != "BUY" {
		t.Fatalf("expected BUY, got %q", snap.Opinion.Type)
	}
	if snap.Opinion.Buy != 12 || snap.Opinion.StrongBuy != 6 {
		t.Fatalf("unexpected opinion counts: %+v", snap.Opinion)
	}
	if snap.Consensus.Mean != 1224.42 {
		t.Fatalf("expected mean 1224.42, got %v", snap.Consensus.Mean)
	}
	if len(snap.Consensus.PastCloses) != 3 {
		t.Fatalf("expected 3 past closes, got %d", len(snap.Consensus.PastCloses))
	}
	if len(snap.Reports) != 0 {
		t.Fatalf("expected 0 reports, got %d", len(snap.Reports))
	}
}
```

- [ ] **Step 5: Run test (fails)**

Run: `go test ./internal/client/ -run TestGetAnalystSnapshot -v`
Expected: FAIL.

- [ ] **Step 6: Implement client method**

Create `internal/client/analyst.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type analystOpinionEnvelope struct {
	Result struct {
		Type        string `json:"type"`
		StrongSell  int    `json:"strongSell"`
		Sell        int    `json:"sell"`
		Hold        int    `json:"hold"`
		Buy         int    `json:"buy"`
		StrongBuy   int    `json:"strongBuy"`
		TargetPrice struct {
			USD float64 `json:"USD"`
			KRW float64 `json:"KRW"`
		} `json:"targetPrice"`
		Description string `json:"description"`
	} `json:"result"`
}

type consensusEnvelope struct {
	Result struct {
		TargetPrice struct {
			Mean, High, Low                  float64 `json:"mean,high,low"`
			MeanKrw, HighKrw, LowKrw         float64 `json:"meanKrw,highKrw,lowKrw"`
			Currency                         string  `json:"currency"`
		} `json:"targetPrice"`
		PointDate       string `json:"pointDate"`
		PastClosePrices []struct {
			Price    float64 `json:"price"`
			PriceKrw float64 `json:"priceKrw"`
			Date     string  `json:"date"`
		} `json:"pastClosePrices"`
	} `json:"result"`
}

type analystReportsEnvelope struct {
	Result struct {
		AnalystReportGroups []struct {
			Date    string `json:"date"`
			Reports []struct {
				Title  string `json:"title"`
				Source string `json:"source"`
				URL    string `json:"url"`
			} `json:"reports"`
		} `json:"analystReportGroups"`
	} `json:"result"`
}

// GetAnalystSnapshot stitches opinion + consensus + reports into one bundle.
func (c *Client) GetAnalystSnapshot(ctx context.Context, symbol string) (domain.AnalystSnapshot, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var opEnv analystOpinionEnvelope
	opURL := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/analyst-opinion", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, opURL, &opEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var conEnv consensusEnvelope
	conURL := fmt.Sprintf("%s/api/v2/stock-infos/consensus/%s", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, conURL, &conEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var repEnv analystReportsEnvelope
	repURL := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/analyst-reports", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, repURL, &repEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	pastCloses := make([]domain.ConsensusPastClose, len(conEnv.Result.PastClosePrices))
	for i, p := range conEnv.Result.PastClosePrices {
		pastCloses[i] = domain.ConsensusPastClose{Date: p.Date, Price: p.Price, PriceKRW: p.PriceKrw}
	}

	var reports []domain.AnalystReport
	for _, group := range repEnv.Result.AnalystReportGroups {
		for _, r := range group.Reports {
			reports = append(reports, domain.AnalystReport{
				Title: r.Title, Source: r.Source, Date: group.Date, URL: r.URL,
			})
		}
	}

	return domain.AnalystSnapshot{
		ProductCode: productCode,
		Opinion: domain.AnalystOpinion{
			Type:        opEnv.Result.Type,
			StrongBuy:   opEnv.Result.StrongBuy,
			Buy:         opEnv.Result.Buy,
			Hold:        opEnv.Result.Hold,
			Sell:        opEnv.Result.Sell,
			StrongSell:  opEnv.Result.StrongSell,
			TargetUSD:   opEnv.Result.TargetPrice.USD,
			TargetKRW:   opEnv.Result.TargetPrice.KRW,
			Description: opEnv.Result.Description,
		},
		Consensus: domain.ConsensusTarget{
			Mean: conEnv.Result.TargetPrice.Mean,
			High: conEnv.Result.TargetPrice.High,
			Low:  conEnv.Result.TargetPrice.Low,
			MeanKRW: conEnv.Result.TargetPrice.MeanKrw,
			HighKRW: conEnv.Result.TargetPrice.HighKrw,
			LowKRW:  conEnv.Result.TargetPrice.LowKrw,
			Currency: conEnv.Result.TargetPrice.Currency,
			PointDate: conEnv.Result.PointDate,
			PastCloses: pastCloses,
		},
		Reports:   reports,
		FetchedAt: time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 7: Run test (passes)**

Run: `go test ./internal/client/ -run TestGetAnalystSnapshot -v`
Expected: PASS.

- [ ] **Step 8: Write output writer test**

Create `internal/output/analyst_test.go`:

```go
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteAnalystSnapshotTable(t *testing.T) {
	t.Parallel()
	snap := domain.AnalystSnapshot{
		ProductCode: "NAS0250224006",
		Opinion: domain.AnalystOpinion{
			Type: "BUY", StrongBuy: 6, Buy: 12, Hold: 4, Sell: 0, StrongSell: 0,
			TargetUSD: 1224.42, TargetKRW: 1794999.72,
			Description: "애널리스트 22명 중 18명이 구매 의견을 냈어요.",
		},
		Consensus: domain.ConsensusTarget{
			Mean: 1224.42, High: 2000.0, Low: 250.0,
			MeanKRW: 1794999.72, HighKRW: 2932000.0, LowKRW: 366500.0,
			Currency: "USD", PointDate: "2026-05-15",
			PastCloses: []domain.ConsensusPastClose{
				{Date: "2026-05-15", Price: 1405.85, PriceKRW: 2097247},
				{Date: "2026-04-30", Price: 1096.51, PriceKRW: 1635773},
			},
		},
		Reports:   nil,
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteAnalystSnapshot(&buf, FormatTable, snap); err != nil {
		t.Fatalf("WriteAnalystSnapshot error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "BUY", "strongBuy 6", "buy 12", "hold 4", "$1224.42", "$2000.00", "$250.00", "2026-05-15", "$1405.85", "0건"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 9: Run output test (fails)**

Run: `go test ./internal/output/ -run TestWriteAnalystSnapshot -v`
Expected: FAIL.

- [ ] **Step 10: Implement output writer**

Create `internal/output/analyst.go`:

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteAnalystSnapshot renders the analyst opinion + consensus + reports
// bundle as a sectioned table.
func WriteAnalystSnapshot(w io.Writer, format Format, snap domain.AnalystSnapshot) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(snap)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "section,key,value"); err != nil {
			return err
		}
		fmt.Fprintf(w, "opinion,type,%s\n", snap.Opinion.Type)
		fmt.Fprintf(w, "opinion,strongBuy,%d\n", snap.Opinion.StrongBuy)
		fmt.Fprintf(w, "opinion,buy,%d\n", snap.Opinion.Buy)
		fmt.Fprintf(w, "opinion,hold,%d\n", snap.Opinion.Hold)
		fmt.Fprintf(w, "opinion,sell,%d\n", snap.Opinion.Sell)
		fmt.Fprintf(w, "opinion,strongSell,%d\n", snap.Opinion.StrongSell)
		fmt.Fprintf(w, "consensus,mean,%.2f\n", snap.Consensus.Mean)
		fmt.Fprintf(w, "consensus,high,%.2f\n", snap.Consensus.High)
		fmt.Fprintf(w, "consensus,low,%.2f\n", snap.Consensus.Low)
		for _, p := range snap.Consensus.PastCloses {
			fmt.Fprintf(w, "pastClose,%s,%.2f\n", p.Date, p.Price)
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — analyst snapshot\n", snap.ProductCode); err != nil {
			return err
		}
		op := snap.Opinion
		fmt.Fprintf(w, "\n=== 의견 ===\n")
		fmt.Fprintf(w, "Type: %s\n", op.Type)
		fmt.Fprintf(w, "strongBuy %d  buy %d  hold %d  sell %d  strongSell %d\n", op.StrongBuy, op.Buy, op.Hold, op.Sell, op.StrongSell)
		if op.Description != "" {
			fmt.Fprintf(w, "%s\n", op.Description)
		}

		fmt.Fprintf(w, "\n=== 컨센서스 목표가 (%s 기준) ===\n", snap.Consensus.PointDate)
		headers := []string{"FIELD", "USD", "KRW"}
		rows := [][]string{
			{"mean", fmt.Sprintf("$%.2f", snap.Consensus.Mean), formatWithCommas(int64(snap.Consensus.MeanKRW))},
			{"high", fmt.Sprintf("$%.2f", snap.Consensus.High), formatWithCommas(int64(snap.Consensus.HighKRW))},
			{"low", fmt.Sprintf("$%.2f", snap.Consensus.Low), formatWithCommas(int64(snap.Consensus.LowKRW))},
		}
		if err := renderTable(w, headers, rows); err != nil {
			return err
		}

		if len(snap.Consensus.PastCloses) > 0 {
			fmt.Fprintf(w, "\n=== 과거 종가 ===\n")
			headers = []string{"DATE", "PRICE (USD)", "PRICE (KRW)"}
			rows = make([][]string, len(snap.Consensus.PastCloses))
			for i, p := range snap.Consensus.PastCloses {
				rows[i] = []string{p.Date, fmt.Sprintf("$%.2f", p.Price), formatWithCommas(int64(p.PriceKRW))}
			}
			if err := renderTable(w, headers, rows); err != nil {
				return err
			}
		}

		fmt.Fprintf(w, "\n애널리스트 보고서: %d건\n", len(snap.Reports))
		if len(snap.Reports) > 0 {
			headers = []string{"DATE", "SOURCE", "TITLE"}
			rows = make([][]string, len(snap.Reports))
			for i, r := range snap.Reports {
				rows[i] = []string{r.Date, r.Source, r.Title}
			}
			if err := renderTable(w, headers, rows); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 11: Run output test (passes)**

Run: `go test ./internal/output/ -run TestWriteAnalystSnapshot -v`
Expected: PASS.

- [ ] **Step 12: Register cobra subcommand**

In `cmd/tossctl/stock.go`, after `peersCmd`:

```go
analystCmd := &cobra.Command{
    Use:   "analyst <symbol>",
    Short: "Analyst BUY/HOLD/SELL counts + consensus target + reports",
    Long: `Fetch the analyst opinion + consensus target price + report list.

Stitches three endpoints:
  - /api/v1/stock-detail/ui/wts/{code}/analyst-opinion
  - /api/v2/stock-infos/consensus/{code}
  - /api/v1/stock-detail/ui/wts/{code}/analyst-reports

Examples:
  tossctl stock analyst SNDK
  tossctl stock analyst NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        snap, err := app.client.GetAnalystSnapshot(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteAnalystSnapshot(cmd.OutOrStdout(), app.format, snap)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd)
```

- [ ] **Step 13: Full build + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 14: Commit**

```bash
git add internal/domain/models.go internal/client/analyst.go internal/client/analyst_test.go internal/output/analyst.go internal/output/analyst_test.go cmd/tossctl/stock.go fixtures/responses/public/analyst-snapshot-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(cli): add tossctl stock analyst (opinion + consensus + reports)"
```

---

## Task 7: CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add changelog entries**

Locate the `## Unreleased` block (or create it if missing). Append under `### Added`:

```markdown
- `tossctl stock indicators <sym>` — investment indicators (PER/PBR/PSR, EPS/BPS/ROE, dividend summary, stability).
- `tossctl stock valuation <sym>` — PER/PBR/PSR vs industry median + peer comparison table.
- `tossctl stock revenue <sym>` — revenue composition by business segment (latest fiscal period).
- `tossctl stock peers <sym>` — TICS industry classification + per-metric peer rankings.
- `tossctl stock analyst <sym>` — analyst opinion + consensus target + report list.
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): PR9 stock deep-tab commands (indicators/valuation/revenue/peers/analyst)"
```

---

## Task 8: Live verification + final merge

**Files:** none modified; just verification + merge metadata.

- [ ] **Step 1: Build binary**

```bash
go build -o /tmp/tossctl ./cmd/tossctl
```

Expected: binary built without errors.

- [ ] **Step 2: Live verify each command**

Run each against the real API (requires public endpoints to be reachable):

```bash
/tmp/tossctl stock indicators SNDK
/tmp/tossctl stock valuation SNDK
/tmp/tossctl stock revenue SNDK
/tmp/tossctl stock peers SNDK
/tmp/tossctl stock analyst SNDK
```

Expected: each command prints a populated table; no 4xx/5xx errors. If any
fails:
- 400/404 → re-verify endpoint path against `docs/reverse-engineering/rpc-catalog.md`.
- empty output → check fixture vs live response shape; some fields may have
  changed since 2026-05-16 capture.

Capture the table output in a scratch buffer; you'll paste a trimmed sample
into the merge commit body.

- [ ] **Step 3: Cross-check with JSON output**

```bash
/tmp/tossctl stock indicators SNDK --output json | head -30
/tmp/tossctl stock valuation SNDK --output json | head -30
```

Expected: well-formed JSON, no null surprise fields.

- [ ] **Step 4: Tag PR9 with a single squash-style merge commit**

Since each task already committed atomically, no merge needed — they all
live on `feat/order-page-integration`. Push:

```bash
git push origin feat/order-page-integration
```

Expected: pushes 8+ commits to fork. Never push to `upstream`.

- [ ] **Step 5: Update memory if any new convention surfaced**

If during live verification you discover a new constraint not yet in
`/Users/whales/.ccs/instances/work/projects/-Users-whales-projects-trading-tossinvest-cli/memory/`,
save it (e.g. an endpoint shape detail, a CSV ordering, a peer count quirk).

Otherwise: skip. Stop here.

---

## Self-Review

**Spec coverage:**
- PR9 5 commands → Tasks 2–6 ✅
- Shared helpers (`resolveCompanyCode`, `postJSONEmpty`) → Task 1 ✅
- `--output json/csv` for all → present in every output writer ✅
- Fixtures + manifest → present in every command task ✅
- CHANGELOG → Task 7 ✅
- Live verification → Task 8 ✅
- PR10–12 explicitly OUT of this plan; covered by spec, will get their own
  plans after fresh captures.

**Placeholder scan:** none. Every code block is concrete and complete.

**Type consistency:** `StockIndicators` / `IndicatorFields` / `StockValuation`
/ `PeerValuation` / `SalesComposition` / `SalesCompositionItem` / `TICSIndustry`
/ `TICSEntry` / `TICSRanking` / `AnalystSnapshot` / `AnalystOpinion` /
`ConsensusTarget` / `ConsensusPastClose` / `AnalystReport` — used identically
across tasks. Client methods named consistently (`Get<Thing>`). Output
writers named consistently (`Write<Thing>`).
