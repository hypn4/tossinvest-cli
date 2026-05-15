# PR13 — Stock statements (financial-statement-records, factor+period switchable)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `tossctl stock statements <sym> [--type BAL|INC|CAS] [--period Q|Y]` for the dense `/api/v2/companies/{code}/financial-statement-records` endpoint that was deferred from PR10. Returns 10 periods × ~34 line items per factor (BAL=재무상태표, INC=손익계산서, CAS=현금흐름표). The endpoint is POST with body `{"factorCode":"INC","period":"Q"}` (non-empty body, unlike PR10's three POSTs).

**Architecture:** One client method `GetStockStatements(ctx, symbol, factor, period)` that POSTs the selector body and parses the table. Output writer pivots rows to be line items and columns to be periods, indenting child items under parents.

**Tech Stack:** Go, cobra, new `postJSON` helper (POST with non-empty body), encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `4262cc1`.

---

## File Structure

**New:**
- `internal/client/statements.go` + `_test.go` — `GetStockStatements`
- `internal/output/statements.go` + `_test.go` — `WriteStockStatements`
- `fixtures/responses/public/stock-statements-sndk-inc-q.json` — INC/Q fixture
- `internal/client/http_post_json.go` — new helper `postJSON(ctx, url, body, dst)` (POST with arbitrary JSON body); reused for any future selector POST.

**Modified:**
- `internal/domain/models.go` — append `StockStatements`, `StatementPeriod`, `StatementLineItem`, `StatementFactor` types
- `cmd/tossctl/stock.go` — register `statementsCmd` with `--type` and `--period` flags
- `fixtures/responses/public/manifest.json` — append fixture entry
- `CHANGELOG.md` — add line

---

## Task 1: postJSON helper + client + fixture + test

### Step 1: postJSON helper

Create `internal/client/http_post_json.go`:

```go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// postJSON issues a POST with an arbitrary JSON body and decodes the response.
// Companion to postJSONEmpty for endpoints that gate on a selector body (e.g.
// /api/v2/companies/{code}/financial-statement-records).
func (c *Client) postJSON(ctx context.Context, endpoint string, body any, dst any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	c.applySession(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", DefaultBrowserUserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return newStatusError(resp.StatusCode, endpoint, respBody)
	}
	return json.Unmarshal(respBody, dst)
}
```

### Step 2: Save fixture

Trim the 59KB capture to just 4 periods + 12 line items for INC/Q. Create `fixtures/responses/public/stock-statements-sndk-inc-q.json`:

```json
{
  "result": {
    "selectedFactor": {"code": "INC", "displayName": "손익계산서"},
    "selectableFactors": [
      {"code": "BAL", "displayName": "재무상태표"},
      {"code": "INC", "displayName": "손익계산서"},
      {"code": "CAS", "displayName": "현금흐름표"}
    ],
    "selectedPeriod": {"code": "Q", "displayName": "분기"},
    "selectablePeriods": [
      {"code": "Q", "displayName": "분기"},
      {"code": "Y", "displayName": "연간"}
    ],
    "isKr": false,
    "table": [
      {
        "period": "2025-06",
        "value": [
          {"item": "RTLR", "parentItem": null, "itemNameKor": "매출액", "itemNameEng": "Total Revenue", "unitType": "USD", "value": 1923.0},
          {"item": "SREV", "parentItem": "RTLR", "itemNameKor": "매출", "itemNameEng": "Revenue", "unitType": "USD", "value": 1923.0},
          {"item": "SCOR", "parentItem": null, "itemNameKor": "매출원가", "itemNameEng": "Cost of Revenue", "unitType": "USD", "value": 1432.0},
          {"item": "SGRP", "parentItem": null, "itemNameKor": "매출총이익", "itemNameEng": "Gross Profit", "unitType": "USD", "value": 491.0},
          {"item": "SOOE", "parentItem": null, "itemNameKor": "영업이익", "itemNameEng": "Operating Income", "unitType": "USD", "value": 18.0},
          {"item": "TIAT", "parentItem": null, "itemNameKor": "당기순이익", "itemNameEng": "Net Income", "unitType": "USD", "value": 252.0}
        ]
      },
      {
        "period": "2025-09",
        "value": [
          {"item": "RTLR", "parentItem": null, "itemNameKor": "매출액", "itemNameEng": "Total Revenue", "unitType": "USD", "value": 2308.0},
          {"item": "SREV", "parentItem": "RTLR", "itemNameKor": "매출", "itemNameEng": "Revenue", "unitType": "USD", "value": 2308.0},
          {"item": "SCOR", "parentItem": null, "itemNameKor": "매출원가", "itemNameEng": "Cost of Revenue", "unitType": "USD", "value": 1542.0},
          {"item": "SGRP", "parentItem": null, "itemNameKor": "매출총이익", "itemNameEng": "Gross Profit", "unitType": "USD", "value": 766.0},
          {"item": "SOOE", "parentItem": null, "itemNameKor": "영업이익", "itemNameEng": "Operating Income", "unitType": "USD", "value": 879.0},
          {"item": "TIAT", "parentItem": null, "itemNameKor": "당기순이익", "itemNameEng": "Net Income", "unitType": "USD", "value": 803.0}
        ]
      },
      {
        "period": "2025-12",
        "value": [
          {"item": "RTLR", "parentItem": null, "itemNameKor": "매출액", "itemNameEng": "Total Revenue", "unitType": "USD", "value": 2306.0},
          {"item": "SREV", "parentItem": "RTLR", "itemNameKor": "매출", "itemNameEng": "Revenue", "unitType": "USD", "value": 2306.0},
          {"item": "SCOR", "parentItem": null, "itemNameKor": "매출원가", "itemNameEng": "Cost of Revenue", "unitType": "USD", "value": 1239.0},
          {"item": "SGRP", "parentItem": null, "itemNameKor": "매출총이익", "itemNameEng": "Gross Profit", "unitType": "USD", "value": 1067.0},
          {"item": "SOOE", "parentItem": null, "itemNameKor": "영업이익", "itemNameEng": "Operating Income", "unitType": "USD", "value": 1067.0},
          {"item": "TIAT", "parentItem": null, "itemNameKor": "당기순이익", "itemNameEng": "Net Income", "unitType": "USD", "value": 887.0}
        ]
      },
      {
        "period": "2026-03",
        "value": [
          {"item": "RTLR", "parentItem": null, "itemNameKor": "매출액", "itemNameEng": "Total Revenue", "unitType": "USD", "value": 3092.0},
          {"item": "SREV", "parentItem": "RTLR", "itemNameKor": "매출", "itemNameEng": "Revenue", "unitType": "USD", "value": 3092.0},
          {"item": "SCOR", "parentItem": null, "itemNameKor": "매출원가", "itemNameEng": "Cost of Revenue", "unitType": "USD", "value": 1581.0},
          {"item": "SGRP", "parentItem": null, "itemNameKor": "매출총이익", "itemNameEng": "Gross Profit", "unitType": "USD", "value": 1511.0},
          {"item": "SOOE", "parentItem": null, "itemNameKor": "영업이익", "itemNameEng": "Operating Income", "unitType": "USD", "value": 4111.0},
          {"item": "TIAT", "parentItem": null, "itemNameKor": "당기순이익", "itemNameEng": "Net Income", "unitType": "USD", "value": 3615.0}
        ]
      }
    ]
  }
}
```

### Step 3: Update manifest

Append:

```json
{
  "file": "stock-statements-sndk-inc-q.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/companies/NAS0250224006/financial-statement-records",
  "method": "POST"
}
```

### Step 4: Add domain types

Append to `internal/domain/models.go`:

```go
// StockStatements is the pivoted view of a financial-statement-records call.
// Periods are ordered oldest-first; line items follow the parent/child order
// returned by the API.
type StockStatements struct {
	ProductCode  string                `json:"product_code"`
	Factor       StatementFactor       `json:"factor"`        // BAL|INC|CAS
	Period       string                `json:"period"`        // Q|Y
	IsKr         bool                  `json:"is_kr"`
	Periods      []StatementPeriod     `json:"periods"`
	FetchedAt    time.Time             `json:"fetched_at"`
}

type StatementFactor struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type StatementPeriod struct {
	Period string              `json:"period"`
	Items  []StatementLineItem `json:"items"`
}

type StatementLineItem struct {
	Item        string   `json:"item"`         // RTLR, SREV, etc
	ParentItem  string   `json:"parent_item,omitempty"`
	NameKor     string   `json:"name_kor"`
	NameEng     string   `json:"name_eng,omitempty"`
	Unit        string   `json:"unit,omitempty"` // USD, KRW, etc
	Value       *float64 `json:"value"`          // null when not reported
}
```

### Step 5: Failing client test

Create `internal/client/statements_test.go`:

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

func TestGetStockStatementsIncQ(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-statements-sndk-inc-q.json"))

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/companies/NAS0250224006/financial-statement-records" {
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
	st, err := c.GetStockStatements(context.Background(), "NAS0250224006", "INC", "Q")
	if err != nil {
		t.Fatalf("GetStockStatements error: %v", err)
	}

	// Verify the request body was the JSON selector
	var sentBody struct {
		FactorCode string `json:"factorCode"`
		Period     string `json:"period"`
	}
	if err := json.Unmarshal(gotBody, &sentBody); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	if sentBody.FactorCode != "INC" || sentBody.Period != "Q" {
		t.Fatalf("unexpected request body: %+v", sentBody)
	}

	if st.Factor.Code != "INC" || st.Factor.DisplayName != "손익계산서" {
		t.Fatalf("unexpected factor: %+v", st.Factor)
	}
	if st.Period != "Q" {
		t.Fatalf("unexpected period: %q", st.Period)
	}
	if len(st.Periods) != 4 {
		t.Fatalf("expected 4 periods, got %d", len(st.Periods))
	}
	if st.Periods[0].Period != "2025-06" || st.Periods[3].Period != "2026-03" {
		t.Fatalf("unexpected period order: %s ... %s", st.Periods[0].Period, st.Periods[3].Period)
	}
	if len(st.Periods[0].Items) != 6 {
		t.Fatalf("expected 6 items in first period, got %d", len(st.Periods[0].Items))
	}
	// Verify parent-child link survives
	srev := st.Periods[3].Items[1]
	if srev.Item != "SREV" || srev.ParentItem != "RTLR" {
		t.Fatalf("unexpected SREV item: %+v", srev)
	}
	// Verify revenue grows
	rtlr := st.Periods[3].Items[0]
	if rtlr.Value == nil || *rtlr.Value != 3092.0 {
		t.Fatalf("unexpected RTLR Q1 2026 value: %v", rtlr.Value)
	}
}

func TestGetStockStatementsRejectsBadFactor(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockStatements(context.Background(), "NAS0250224006", "XYZ", "Q")
	if err == nil {
		t.Fatalf("expected error for bad factor")
	}
}

func TestGetStockStatementsRejectsBadPeriod(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.GetStockStatements(context.Background(), "NAS0250224006", "INC", "X")
	if err == nil {
		t.Fatalf("expected error for bad period")
	}
}
```

### Step 6: Run test (FAIL)

### Step 7: Implement client

Create `internal/client/statements.go`:

```go
package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type statementRecordsEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedFactor"`
		SelectedPeriod struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedPeriod"`
		IsKr  bool `json:"isKr"`
		Table []struct {
			Period string `json:"period"`
			Value  []struct {
				Item        string   `json:"item"`
				ParentItem  string   `json:"parentItem"`
				ItemNameKor string   `json:"itemNameKor"`
				ItemNameEng string   `json:"itemNameEng"`
				UnitType    string   `json:"unitType"`
				Value       *float64 `json:"value"`
			} `json:"value"`
		} `json:"table"`
	} `json:"result"`
}

// GetStockStatements fetches the financial-statement-records payload for a
// chosen factor (BAL|INC|CAS) and period (Q|Y). Returns the pivoted view.
func (c *Client) GetStockStatements(ctx context.Context, symbol, factorCode, period string) (domain.StockStatements, error) {
	factorCode = strings.ToUpper(factorCode)
	if factorCode != "BAL" && factorCode != "INC" && factorCode != "CAS" {
		return domain.StockStatements{}, fmt.Errorf("GetStockStatements: factorCode must be one of BAL|INC|CAS (got %q)", factorCode)
	}
	period = strings.ToUpper(period)
	if period != "Q" && period != "Y" {
		return domain.StockStatements{}, fmt.Errorf("GetStockStatements: period must be Q or Y (got %q)", period)
	}

	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockStatements{}, err
	}

	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statement-records", c.infoBaseURL, productCode)
	body := map[string]string{"factorCode": factorCode, "period": period}
	var env statementRecordsEnvelope
	if err := c.postJSON(ctx, endpoint, body, &env); err != nil {
		return domain.StockStatements{}, err
	}

	periods := make([]domain.StatementPeriod, len(env.Result.Table))
	for i, p := range env.Result.Table {
		items := make([]domain.StatementLineItem, len(p.Value))
		for j, v := range p.Value {
			items[j] = domain.StatementLineItem{
				Item:       v.Item,
				ParentItem: v.ParentItem,
				NameKor:    v.ItemNameKor,
				NameEng:    v.ItemNameEng,
				Unit:       v.UnitType,
				Value:      v.Value,
			}
		}
		periods[i] = domain.StatementPeriod{Period: p.Period, Items: items}
	}

	return domain.StockStatements{
		ProductCode: productCode,
		Factor: domain.StatementFactor{
			Code:        env.Result.SelectedFactor.Code,
			DisplayName: env.Result.SelectedFactor.DisplayName,
		},
		Period:    env.Result.SelectedPeriod.Code,
		IsKr:      env.Result.IsKr,
		Periods:   periods,
		FetchedAt: time.Now().UTC(),
	}, nil
}
```

### Step 8: Run test (PASS — all 3 subtests)

### Step 9: Commit

```bash
git add internal/domain/models.go internal/client/http_post_json.go internal/client/statements.go internal/client/statements_test.go fixtures/responses/public/stock-statements-sndk-inc-q.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockStatements (financial-statement-records, factor+period switchable)"
```

---

## Task 2: Output writer + cobra + CHANGELOG

### Step 1: Failing output test

Create `internal/output/statements_test.go`:

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

func statementsTestFixture() domain.StockStatements {
	v := func(f float64) *float64 { return &f }
	return domain.StockStatements{
		ProductCode: "NAS0250224006",
		Factor:      domain.StatementFactor{Code: "INC", DisplayName: "손익계산서"},
		Period:      "Q",
		IsKr:        false,
		Periods: []domain.StatementPeriod{
			{
				Period: "2025-12",
				Items: []domain.StatementLineItem{
					{Item: "RTLR", NameKor: "매출액", NameEng: "Total Revenue", Unit: "USD", Value: v(2306)},
					{Item: "SREV", ParentItem: "RTLR", NameKor: "매출", NameEng: "Revenue", Unit: "USD", Value: v(2306)},
					{Item: "TIAT", NameKor: "당기순이익", NameEng: "Net Income", Unit: "USD", Value: v(887)},
				},
			},
			{
				Period: "2026-03",
				Items: []domain.StatementLineItem{
					{Item: "RTLR", NameKor: "매출액", NameEng: "Total Revenue", Unit: "USD", Value: v(3092)},
					{Item: "SREV", ParentItem: "RTLR", NameKor: "매출", NameEng: "Revenue", Unit: "USD", Value: v(3092)},
					{Item: "TIAT", NameKor: "당기순이익", NameEng: "Net Income", Unit: "USD", Value: v(3615)},
				},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockStatementsTable(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatTable, st); err != nil {
		t.Fatalf("WriteStockStatements error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"NAS0250224006",
		"손익계산서", "(Q)",
		"ITEM", "NAME", "2025-12", "2026-03",
		"RTLR", "매출액", "2,306", "3,092",
		"  SREV", // child indent
		"TIAT", "당기순이익", "3,615",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockStatementsTableNilValues(t *testing.T) {
	t.Parallel()
	v := func(f float64) *float64 { return &f }
	st := domain.StockStatements{
		ProductCode: "X",
		Factor:      domain.StatementFactor{Code: "INC", DisplayName: "손익계산서"},
		Period:      "Q",
		Periods: []domain.StatementPeriod{
			{Period: "2026-03", Items: []domain.StatementLineItem{
				{Item: "RTLR", NameKor: "매출액", Unit: "USD", Value: v(100)},
				{Item: "SORE", ParentItem: "RTLR", NameKor: "기타 수익", Unit: "", Value: nil},
			}},
		},
	}
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatTable, st); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "—") {
		t.Fatalf("expected — for nil value; got:\n%s", buf.String())
	}
}

func TestWriteStockStatementsCSV(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatCSV, st); err != nil {
		t.Fatalf("CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if rows[0][0] != "item" || rows[0][1] != "parent_item" || rows[0][2] != "name_kor" {
		t.Fatalf("unexpected CSV header: %v", rows[0])
	}
	// Header + 3 items (we dedupe — each line item appears once even though we have 2 periods)
	// Actually CSV pivots periods as columns too, so we'll have item rows with period columns
	if len(rows) < 4 {
		t.Fatalf("expected at least 4 rows, got %d", len(rows))
	}
}

func TestWriteStockStatementsJSON(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatJSON, st); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	var got domain.StockStatements
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if got.Factor.Code != "INC" {
		t.Fatalf("factor roundtrip mismatch")
	}
	if !strings.Contains(buf.String(), `"name_kor"`) {
		t.Fatalf("expected snake_case name_kor in JSON")
	}
}
```

### Step 2: Run test (FAIL)

### Step 3: Implement output writer

Create `internal/output/statements.go`:

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

// WriteStockStatements renders the pivoted financial-statement-records table:
// rows = line items (in document order, children indented under parents),
// columns = periods (oldest-first). Values display in millions when the unit
// is USD, raw otherwise.
func WriteStockStatements(w io.Writer, format Format, st domain.StockStatements) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(st)

	case FormatCSV:
		// Pivot: one row per (item, period); columns = item, parent_item, name_kor, name_eng, unit, period, value
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"item", "parent_item", "name_kor", "name_eng", "unit", "period", "value"}); err != nil {
			return err
		}
		for _, p := range st.Periods {
			for _, it := range p.Items {
				val := ""
				if it.Value != nil {
					val = strconv.FormatFloat(*it.Value, 'f', 2, 64)
				}
				if err := cw.Write([]string{
					it.Item, it.ParentItem, it.NameKor, it.NameEng, it.Unit, p.Period, val,
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()

	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — %s (%s)\n", st.ProductCode, st.Factor.DisplayName, st.Period); err != nil {
			return err
		}

		// Collect line items in document order from the LAST period (most
		// complete). Use a stable order taken from the first non-empty period.
		var refItems []domain.StatementLineItem
		for _, p := range st.Periods {
			if len(p.Items) > 0 {
				refItems = p.Items
				break
			}
		}
		if len(refItems) == 0 {
			fmt.Fprintln(w, "  (no rows returned)")
			return nil
		}

		// Build a value lookup: map[period][item] -> value
		valuesByPeriod := make(map[string]map[string]*float64, len(st.Periods))
		for _, p := range st.Periods {
			m := make(map[string]*float64, len(p.Items))
			for _, it := range p.Items {
				m[it.Item] = it.Value
			}
			valuesByPeriod[p.Period] = m
		}

		// Header row
		periodCols := make([]string, len(st.Periods))
		for i, p := range st.Periods {
			periodCols[i] = p.Period
		}
		headers := append([]string{"ITEM", "NAME"}, periodCols...)

		// Data rows
		rows := make([][]string, len(refItems))
		for i, ref := range refItems {
			label := ref.Item
			if ref.ParentItem != "" {
				label = "  " + ref.Item
			}
			name := ref.NameKor
			if name == "" {
				name = ref.NameEng
			}
			row := make([]string, 0, 2+len(periodCols))
			row = append(row, label, name)
			for _, p := range st.Periods {
				v := valuesByPeriod[p.Period][ref.Item]
				row = append(row, formatStatementValue(v, ref.Unit))
			}
			rows[i] = row
		}

		if err := renderTable(w, headers, rows); err != nil {
			return err
		}

		// Footer unit hint
		unit := ""
		if len(refItems) > 0 && refItems[0].Unit != "" {
			unit = refItems[0].Unit
		}
		if unit != "" {
			fmt.Fprintf(w, "(values in millions of %s; — = not reported)\n", unit)
		}
		return nil

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatStatementValue(v *float64, unit string) string {
	if v == nil {
		return "—"
	}
	// Values come in as raw units (e.g. 1665.0 for $1.665B). Toss's web UI
	// displays these as-is with comma separators — they are already in
	// millions per the unitType (USD=USD millions).
	return formatWithCommas(int64(*v))
}
```

### Step 4: Run test (PASS — all 4 subtests)

### Step 5: Register cobra subcommand

In `cmd/tossctl/stock.go`, after `estimatesCmd`:

```go
var statementsFactor, statementsPeriod string
statementsCmd := &cobra.Command{
    Use:   "statements <symbol>",
    Short: "Financial-statement records (BAL/INC/CAS × Q/Y, pivoted by period)",
    Long: `Fetch the full financial-statement-records table from
/api/v2/companies/{code}/financial-statement-records (POST {factorCode, period}).

--type selects the statement:
  BAL  재무상태표 (balance sheet)
  INC  손익계산서 (income statement, default)
  CAS  현금흐름표 (cash flow statement)

--period selects:
  Q  분기 (quarterly, default)
  Y  연간 (annual)

Output pivots the table: rows are line items (children indented under parents),
columns are periods (oldest-first). Values are in millions of the unit
reported by Toss (typically USD for US stocks, KRW for KR stocks).

Examples:
  tossctl stock statements SNDK
  tossctl stock statements SNDK --type BAL --period Y
  tossctl stock statements NAS0250224006 --type CAS --output csv`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        st, err := app.client.GetStockStatements(cmd.Context(), args[0], statementsFactor, statementsPeriod)
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockStatements(cmd.OutOrStdout(), app.format, st)
    },
}
statementsCmd.Flags().StringVar(&statementsFactor, "type", "INC", "Statement type: BAL (balance sheet), INC (income statement), CAS (cash flow)")
statementsCmd.Flags().StringVar(&statementsPeriod, "period", "Q", "Period granularity: Q (quarterly) or Y (annual)")
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd, dividendsCmd, estimatesCmd, statementsCmd)
```

### Step 6: CHANGELOG entry

In `CHANGELOG.md`, after the `stock estimates` line, add:

```markdown
- `tossctl stock statements <symbol> [--type BAL|INC|CAS] [--period Q|Y]` — full financial-statement records (BS/IS/CF line items, pivoted by period) via POST `/api/v2/companies/{code}/financial-statement-records` with `{factorCode, period}` body. Default INC/Q. Closes the dense-endpoint gap deferred from PR10.
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`

### Step 8: Commit

```bash
git add internal/output/statements.go internal/output/statements_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock statements (BAL/INC/CAS × Q/Y line items)"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock statements SNDK` prints income-statement Q pivot (~10 periods × ~34 line items).
- [ ] `/tmp/tossctl stock statements SNDK --type BAL --period Y` prints annual balance sheet.
- [ ] `/tmp/tossctl stock statements SNDK --type CAS` prints quarterly cash-flow statement.
- [ ] `/tmp/tossctl stock statements SNDK --output json | head -30` confirms snake_case + nested period structure.
- [ ] `/tmp/tossctl stock statements SNDK --type XYZ` returns a clear error.
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 1 dense POST endpoint with selector body ✅
- `--type` and `--period` flags ✅
- Pivoted output (items × periods) ✅
- Nil-value handling (`—`) ✅
- snake_case JSON tags ✅
- `encoding/csv` for CSV ✅
- Factor/period validation at client level ✅
- CHANGELOG ✅

**Placeholder scan:** none.

**Type consistency:** `StockStatements` uses snake_case JSON tags; the wire envelope uses camelCase. Tests cover happy path + nil values + 3 formats + invalid flags.
