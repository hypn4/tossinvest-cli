# Options Chain Read-Only Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Plan tasks use checkbox (`- [ ]`) syntax.

**Goal:** Expose the option chain enumeration endpoints captured via chrome-devtools on 2026-05-16. Read-only only — option ordering remains out of scope (see memory `options-scope-read-only`).

**Architecture:** Three new client methods + three new output writers + three new CLI subcommands under `tossctl options`. Reuses existing helpers (`resolveProductCode`, `getJSON`, `domain` types).

**Tech Stack:** Go, cobra (no new dependencies).

---

## Reference

- RE doc: [`docs/reverse-engineering/options.md`](../../reverse-engineering/options.md) (Chain Enumeration section)
- RPC catalog rows: `/api/v1/option-maturity-date/get-all`, `/api/v1/option-both-chain/get-all`, `/api/v2/stock-prices` (bulk)
- Raw captures: `.captures/2026-05-16/sndk-option-chain/`
- Existing patterns: `internal/client/options.go` (PR6 `GetOptionInstrument` / `GetNearestATMOption`)
- Fork-only branch: `feat/pr7-options-chain` (already created off `feat/order-page-integration`)

Key constraints:

1. All three endpoints are **public GET** — no auth.
2. `option-both-chain/get-all`'s `result` is a **flat array**, not nested in `items`. Don't confuse with `option-maturity-date/get-all` whose `result.items` IS nested.
3. Bulk price endpoint takes comma-separated `codes` (URL-encoded by `url.Values.Encode`). Chain UI batches ~58 codes/call.
4. `with-prices` join for `options chain` is implemented client-side: fetch the chain (one call) then bulk-fetch prices for the visible codes (one call), join by `code == callGuid|putGuid`.

---

## Task 1: Domain types — `OptionExpiry`, `OptionChainRow`, `OptionPrice`

**Files:**
- Modify: `internal/domain/models.go`

- [ ] **Step 1: Append at end of file**

```go
// OptionExpiry is one expiry-ladder entry returned by
// /api/v1/option-maturity-date/get-all.
type OptionExpiry struct {
	MaturityDate                string `json:"maturity_date"`
	MaturityDateTime            string `json:"maturity_date_time,omitempty"`
	LiquidationDateTime         string `json:"liquidation_date_time,omitempty"`
	DisplayLiquidationDateTime  string `json:"display_liquidation_date_time,omitempty"`
	CorporateActionDateTime     string `json:"corporate_action_date_time,omitempty"`
	CorporateActionName         string `json:"corporate_action_name,omitempty"`
	DisplayCorporateActionName  string `json:"display_corporate_action_name,omitempty"`
}

// OptionChainRow is one strike row returned by /api/v1/option-both-chain/get-all.
// CallPrice/PutPrice are populated only when callers fetch prices separately and
// join by callGuid/putGuid (see GetOptionPrices).
type OptionChainRow struct {
	StrikePrice      float64      `json:"strike_price"`
	CallGuid         string       `json:"call_guid,omitempty"`
	PutGuid          string       `json:"put_guid,omitempty"`
	CallOpenInterest int          `json:"call_open_interest,omitempty"`
	PutOpenInterest  int          `json:"put_open_interest,omitempty"`
	CallPrice        *OptionPrice `json:"call_price,omitempty"`
	PutPrice         *OptionPrice `json:"put_price,omitempty"`
}

// OptionPrice is one price row returned by /api/v2/stock-prices (bulk).
type OptionPrice struct {
	Code             string  `json:"code"`
	Base             float64 `json:"base,omitempty"`
	Close            float64 `json:"close,omitempty"`
	ChangeType       string  `json:"change_type,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Volume           float64 `json:"volume,omitempty"`
	BaseKrw          float64 `json:"base_krw,omitempty"`
	CloseKrw         float64 `json:"close_krw,omitempty"`
	BaseKrwDecimal   float64 `json:"base_krw_decimal,omitempty"`
	CloseKrwDecimal  float64 `json:"close_krw_decimal,omitempty"`
}
```

- [ ] **Step 2: Build**

`go build ./...` → success.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/models.go
git commit -m "feat(domain): add OptionExpiry, OptionChainRow, OptionPrice types"
```

---

## Task 2: Client — `ListOptionExpiries`

**Files:**
- Modify: `internal/client/options.go`
- Modify: `internal/client/options_test.go`
- Create fixture: `fixtures/responses/public/option-expiries-sndk.json`
- Modify: `fixtures/responses/public/manifest.json`

- [ ] **Step 1: Create fixture `fixtures/responses/public/option-expiries-sndk.json`**

```json
{
  "result": {
    "items": [
      {
        "maturityDate": "2026-05-15",
        "maturityDateTime": "2026-05-16T03:50:00.000+09:00",
        "liquidationDateTime": "2026-05-15T14:50:00-04:00",
        "displayLiquidationDateTime": "24분 후 거래 종료",
        "corporateActionDateTime": null,
        "corporateActionName": null,
        "displayCorporateActionName": null
      },
      {
        "maturityDate": "2026-05-22",
        "maturityDateTime": "2026-05-23T03:50:00.000+09:00",
        "liquidationDateTime": "2026-05-22T14:50:00-04:00",
        "displayLiquidationDateTime": "7일 후 거래 종료",
        "corporateActionDateTime": null,
        "corporateActionName": null,
        "displayCorporateActionName": null
      }
    ]
  }
}
```

- [ ] **Step 2: Append to `fixtures/responses/public/manifest.json`**

```json
{
  "file": "option-expiries-sndk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/option-maturity-date/get-all?underlyingGuid=NAS0250224006",
  "method": "GET"
}
```

- [ ] **Step 3: Failing test** (append to `internal/client/options_test.go`)

```go
func TestListOptionExpiriesFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "option-expiries-sndk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/option-maturity-date/get-all":
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
	exps, err := c.ListOptionExpiries(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("ListOptionExpiries error: %v", err)
	}
	if len(exps) != 2 {
		t.Fatalf("expected 2 expiries, got %d", len(exps))
	}
	if exps[0].MaturityDate != "2026-05-15" || exps[0].DisplayLiquidationDateTime != "24분 후 거래 종료" {
		t.Fatalf("unexpected first expiry: %+v", exps[0])
	}
}
```

- [ ] **Step 4: Run (must fail)**

`go test ./internal/client/ -run TestListOptionExpiriesFromFixture -v` → `undefined`.

- [ ] **Step 5: Implement in `internal/client/options.go`** (append)

```go
type optionExpiriesEnvelope struct {
	Result struct {
		Items []struct {
			MaturityDate               string `json:"maturityDate"`
			MaturityDateTime           string `json:"maturityDateTime"`
			LiquidationDateTime        string `json:"liquidationDateTime"`
			DisplayLiquidationDateTime string `json:"displayLiquidationDateTime"`
			CorporateActionDateTime    string `json:"corporateActionDateTime"`
			CorporateActionName        string `json:"corporateActionName"`
			DisplayCorporateActionName string `json:"displayCorporateActionName"`
		} `json:"items"`
	} `json:"result"`
}

// ListOptionExpiries returns the expiry ladder for an underlying.
// Accepts symbol or productCode.
func (c *Client) ListOptionExpiries(ctx context.Context, underlying string) ([]domain.OptionExpiry, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-maturity-date/get-all", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	endpoint.RawQuery = q.Encode()

	var env optionExpiriesEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionExpiry, 0, len(env.Result.Items))
	for _, it := range env.Result.Items {
		out = append(out, domain.OptionExpiry{
			MaturityDate:               it.MaturityDate,
			MaturityDateTime:           it.MaturityDateTime,
			LiquidationDateTime:        it.LiquidationDateTime,
			DisplayLiquidationDateTime: it.DisplayLiquidationDateTime,
			CorporateActionDateTime:    it.CorporateActionDateTime,
			CorporateActionName:        it.CorporateActionName,
			DisplayCorporateActionName: it.DisplayCorporateActionName,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Test passes**

`go test ./internal/client/ -run TestListOptionExpiriesFromFixture -v` → PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/client/options.go internal/client/options_test.go fixtures/responses/public/option-expiries-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add ListOptionExpiries"
```

---

## Task 3: Client — `GetOptionChain`

**Files:**
- Modify: `internal/client/options.go`, `internal/client/options_test.go`
- Create fixture: `fixtures/responses/public/option-chain-sndk-2026-05-15.json`
- Modify: manifest.json

- [ ] **Step 1: Create fixture** (3 rows, real shape):

```json
{
  "result": [
    {"strikePrice": 260.0, "callGuid": "OPT_SNDK260515C00260000_20260129", "putGuid": "OPT_SNDK260515P00260000_20260129", "callOpenInterest": 65, "putOpenInterest": 1042},
    {"strikePrice": 265.0, "callGuid": "OPT_SNDK260515C00265000_20260129", "putGuid": "OPT_SNDK260515P00265000_20260129", "callOpenInterest": 12, "putOpenInterest": 88},
    {"strikePrice": 1395.0, "callGuid": "OPT_SNDK260515C01395000_20260506", "putGuid": "OPT_SNDK260515P01395000_20260506", "callOpenInterest": 82, "putOpenInterest": 41}
  ]
}
```

- [ ] **Step 2: Add manifest entry**

```json
{
  "file": "option-chain-sndk-2026-05-15.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/option-both-chain/get-all?underlyingGuid=NAS0250224006&maturityDate=2026-05-15",
  "method": "GET"
}
```

- [ ] **Step 3: Failing test** (append):

```go
func TestGetOptionChainFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "option-chain-sndk-2026-05-15.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/option-both-chain/get-all" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("underlyingGuid") != "NAS0250224006" {
			t.Fatalf("bad underlyingGuid: %s", r.URL.Query().Get("underlyingGuid"))
		}
		if r.URL.Query().Get("maturityDate") != "2026-05-15" {
			t.Fatalf("bad maturityDate: %s", r.URL.Query().Get("maturityDate"))
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	rows, err := c.GetOptionChain(context.Background(), "NAS0250224006", "2026-05-15")
	if err != nil {
		t.Fatalf("GetOptionChain error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[2].StrikePrice != 1395 || rows[2].CallGuid != "OPT_SNDK260515C01395000_20260506" {
		t.Fatalf("unexpected last row: %+v", rows[2])
	}
}
```

- [ ] **Step 4: Run failing**

`go test ./internal/client/ -run TestGetOptionChainFromFixture -v` → undefined.

- [ ] **Step 5: Implement** (append to `options.go`):

```go
type optionChainEnvelope struct {
	Result []struct {
		StrikePrice      float64 `json:"strikePrice"`
		CallGuid         string  `json:"callGuid"`
		PutGuid          string  `json:"putGuid"`
		CallOpenInterest int     `json:"callOpenInterest"`
		PutOpenInterest  int     `json:"putOpenInterest"`
	} `json:"result"`
}

// GetOptionChain returns the strike chain for an underlying's specific expiry.
// `maturityDate` must be in YYYY-MM-DD form.
func (c *Client) GetOptionChain(ctx context.Context, underlying, maturityDate string) ([]domain.OptionChainRow, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-both-chain/get-all", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	q.Set("maturityDate", maturityDate)
	endpoint.RawQuery = q.Encode()

	var env optionChainEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionChainRow, 0, len(env.Result))
	for _, r := range env.Result {
		out = append(out, domain.OptionChainRow{
			StrikePrice:      r.StrikePrice,
			CallGuid:         r.CallGuid,
			PutGuid:          r.PutGuid,
			CallOpenInterest: r.CallOpenInterest,
			PutOpenInterest:  r.PutOpenInterest,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Test passes**

`go test ./internal/client/ -run TestGetOptionChainFromFixture -v` → PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/client/options.go internal/client/options_test.go fixtures/responses/public/option-chain-sndk-2026-05-15.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetOptionChain"
```

---

## Task 4: Client — `GetOptionPrices` (bulk)

**Files:**
- Modify: `internal/client/options.go`, `internal/client/options_test.go`
- Create fixture: `fixtures/responses/public/option-prices-bulk.json`
- Modify: manifest.json

- [ ] **Step 1: Fixture**

```json
{
  "result": {
    "prices": [
      {"code":"OPT_SNDK260515C01395000_20260506","base":31.8,"close":15.4,"changeType":"DOWN","currency":"USD","volume":462,"baseKrw":47439,"closeKrw":22974,"baseKrwDecimal":47439.24,"closeKrwDecimal":22974.92},
      {"code":"OPT_SNDK260515P01395000_20260506","base":1.8,"close":4.2,"changeType":"UP","currency":"USD","volume":210,"baseKrw":2685,"closeKrw":6266,"baseKrwDecimal":2685.24,"closeKrwDecimal":6266.56}
    ]
  }
}
```

- [ ] **Step 2: Add manifest entry**

```json
{
  "file": "option-prices-bulk.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/stock-prices?codes=OPT_SNDK260515C01395000_20260506%2COPT_SNDK260515P01395000_20260506",
  "method": "GET"
}
```

- [ ] **Step 3: Failing test**

```go
func TestGetOptionPricesFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "option-prices-bulk.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/stock-prices" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		got := r.URL.Query().Get("codes")
		want := "OPT_SNDK260515C01395000_20260506,OPT_SNDK260515P01395000_20260506"
		if got != want {
			t.Fatalf("codes mismatch:\n got: %s\nwant: %s", got, want)
		}
		w.Write(body)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	prices, err := c.GetOptionPrices(context.Background(), []string{
		"OPT_SNDK260515C01395000_20260506",
		"OPT_SNDK260515P01395000_20260506",
	})
	if err != nil {
		t.Fatalf("GetOptionPrices error: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].Code != "OPT_SNDK260515C01395000_20260506" || prices[0].Close != 15.4 {
		t.Fatalf("unexpected first price: %+v", prices[0])
	}
}
```

- [ ] **Step 4: Run failing**

`go test ./internal/client/ -run TestGetOptionPricesFromFixture -v` → undefined.

- [ ] **Step 5: Implement** (append):

```go
type optionPricesEnvelope struct {
	Result struct {
		Prices []struct {
			Code            string  `json:"code"`
			Base            float64 `json:"base"`
			Close           float64 `json:"close"`
			ChangeType      string  `json:"changeType"`
			Currency        string  `json:"currency"`
			Volume          float64 `json:"volume"`
			BaseKrw         float64 `json:"baseKrw"`
			CloseKrw        float64 `json:"closeKrw"`
			BaseKrwDecimal  float64 `json:"baseKrwDecimal"`
			CloseKrwDecimal float64 `json:"closeKrwDecimal"`
		} `json:"prices"`
	} `json:"result"`
}

// GetOptionPrices fetches the bulk price list for a slice of productCodes
// (typically OPT_ codes but also works for stocks). The URL encodes codes as
// a comma-separated `codes` parameter, matching what the chain UI sends.
func (c *Client) GetOptionPrices(ctx context.Context, codes []string) ([]domain.OptionPrice, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("GetOptionPrices: codes is empty")
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v2/stock-prices", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("codes", strings.Join(codes, ","))
	endpoint.RawQuery = q.Encode()

	var env optionPricesEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionPrice, 0, len(env.Result.Prices))
	for _, p := range env.Result.Prices {
		out = append(out, domain.OptionPrice{
			Code:            p.Code,
			Base:            p.Base,
			Close:           p.Close,
			ChangeType:      p.ChangeType,
			Currency:        p.Currency,
			Volume:          p.Volume,
			BaseKrw:         p.BaseKrw,
			CloseKrw:        p.CloseKrw,
			BaseKrwDecimal:  p.BaseKrwDecimal,
			CloseKrwDecimal: p.CloseKrwDecimal,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Pass**

`go test ./internal/client/ -run TestGetOptionPricesFromFixture -v` → PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/client/options.go internal/client/options_test.go fixtures/responses/public/option-prices-bulk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetOptionPrices (bulk /api/v2/stock-prices)"
```

---

## Task 5: Output writers — `WriteOptionExpiries`, `WriteOptionChain`, `WriteOptionPrices`

**Files:**
- Modify: `internal/output/options.go`
- Modify: `internal/output/options_test.go`

- [ ] **Step 1: Failing tests** (append):

```go
func TestWriteOptionExpiriesTable(t *testing.T) {
	exps := []domain.OptionExpiry{
		{MaturityDate: "2026-05-15", DisplayLiquidationDateTime: "24분 후 거래 종료"},
		{MaturityDate: "2026-05-22", DisplayLiquidationDateTime: "7일 후 거래 종료"},
	}
	var buf bytes.Buffer
	if err := WriteOptionExpiries(&buf, FormatTable, exps); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "2026-05-15") || !strings.Contains(s, "거래 종료") {
		t.Fatalf("expected expiry rows in table:\n%s", s)
	}
}

func TestWriteOptionChainTable(t *testing.T) {
	rows := []domain.OptionChainRow{
		{StrikePrice: 1395, CallGuid: "OPT_C", PutGuid: "OPT_P", CallOpenInterest: 82, PutOpenInterest: 41},
	}
	var buf bytes.Buffer
	if err := WriteOptionChain(&buf, FormatTable, rows); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "1395") || !strings.Contains(s, "82") {
		t.Fatalf("expected strike row in table:\n%s", s)
	}
}

func TestWriteOptionPricesJSON(t *testing.T) {
	prices := []domain.OptionPrice{{Code: "OPT_X", Close: 12.3, Volume: 100}}
	var buf bytes.Buffer
	if err := WriteOptionPrices(&buf, FormatJSON, prices); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.OptionPrice
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed[0].Close != 12.3 {
		t.Fatalf("unexpected roundtrip: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run failing**

- [ ] **Step 3: Implement** (append to `internal/output/options.go`):

```go
// WriteOptionExpiries renders the expiry ladder.
func WriteOptionExpiries(w io.Writer, format Format, exps []domain.OptionExpiry) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(exps)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "maturity_date,display_liquidation,maturity_date_time"); err != nil {
			return err
		}
		for _, e := range exps {
			if _, err := fmt.Fprintf(w, "%s,%s,%s\n", e.MaturityDate, e.DisplayLiquidationDateTime, e.MaturityDateTime); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"MATURITY", "COUNTDOWN", "LIQUIDATION"}
		rows := make([][]string, 0, len(exps))
		for _, e := range exps {
			rows = append(rows, []string{e.MaturityDate, e.DisplayLiquidationDateTime, e.LiquidationDateTime})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteOptionChain renders the strike chain. When rows include CallPrice/PutPrice
// (joined via --with-prices), price columns are populated.
func WriteOptionChain(w io.Writer, format Format, rows []domain.OptionChainRow) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "strike,call_oi,put_oi,call_close,put_close,call_guid,put_guid"); err != nil {
			return err
		}
		for _, r := range rows {
			callClose, putClose := "", ""
			if r.CallPrice != nil {
				callClose = formatFloat(r.CallPrice.Close)
			}
			if r.PutPrice != nil {
				putClose = formatFloat(r.PutPrice.Close)
			}
			if _, err := fmt.Fprintf(w, "%s,%d,%d,%s,%s,%s,%s\n",
				formatFloat(r.StrikePrice), r.CallOpenInterest, r.PutOpenInterest,
				callClose, putClose, r.CallGuid, r.PutGuid); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"STRIKE", "CALL OI", "CALL ₵", "PUT ₵", "PUT OI"}
		body := make([][]string, 0, len(rows))
		for _, r := range rows {
			callClose, putClose := "-", "-"
			if r.CallPrice != nil {
				callClose = formatFloat(r.CallPrice.Close)
			}
			if r.PutPrice != nil {
				putClose = formatFloat(r.PutPrice.Close)
			}
			body = append(body, []string{
				formatFloat(r.StrikePrice),
				fmt.Sprintf("%d", r.CallOpenInterest),
				callClose,
				putClose,
				fmt.Sprintf("%d", r.PutOpenInterest),
			})
		}
		return renderTable(w, headers, body)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteOptionPrices renders a flat list of option/stock price rows.
func WriteOptionPrices(w io.Writer, format Format, prices []domain.OptionPrice) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(prices)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "code,base,close,change_type,currency,volume"); err != nil {
			return err
		}
		for _, p := range prices {
			if _, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s\n",
				p.Code, formatFloat(p.Base), formatFloat(p.Close),
				p.ChangeType, p.Currency, formatFloat(p.Volume)); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		headers := []string{"CODE", "BASE", "CLOSE", "Δ", "VOLUME"}
		body := make([][]string, 0, len(prices))
		for _, p := range prices {
			body = append(body, []string{p.Code, formatFloat(p.Base), formatFloat(p.Close), p.ChangeType, formatFloat(p.Volume)})
		}
		return renderTable(w, headers, body)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

- [ ] **Step 4: Pass + regression**

- [ ] **Step 5: Commit**

```bash
git add internal/output/options.go internal/output/options_test.go
git commit -m "feat(output): WriteOptionExpiries + WriteOptionChain + WriteOptionPrices"
```

---

## Task 6: CLI — `tossctl options expiries / chain / prices`

**Files:**
- Modify: `cmd/tossctl/options.go`

- [ ] **Step 1: Add three subcommands to `newOptionsCmd`**

Edit `cmd/tossctl/options.go`. Keep existing `infoCmd` and `atmCmd`. Add:

```go
// Just before cmd.AddCommand(...) line, declare the three new sub-commands and
// register them in the AddCommand call.

expiriesCmd := &cobra.Command{
	Use:   "expiries <underlying>",
	Short: "List the option expiry ladder for an underlying",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := newAppContext(opts)
		if err != nil {
			return err
		}
		exps, err := app.client.ListOptionExpiries(cmd.Context(), args[0])
		if err != nil {
			return userFacingCommandError(err)
		}
		return output.WriteOptionExpiries(cmd.OutOrStdout(), app.format, exps)
	},
}

var (
	chainExpiry     string
	chainType       string
	chainWithPrices bool
)
chainCmd := &cobra.Command{
	Use:   "chain <underlying>",
	Short: "Show the strike chain (call + put per strike) for an expiry",
	Long: `Show the strike chain for an underlying's expiry.

If --expiry is omitted, the nearest expiry from the expiry ladder is used.

Use --with-prices to join in bulk option prices (one extra round-trip; ~58
codes per call). --type call|put filters the rows client-side.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := newAppContext(opts)
		if err != nil {
			return err
		}
		expiry := chainExpiry
		if expiry == "" {
			exps, err := app.client.ListOptionExpiries(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			if len(exps) == 0 {
				return fmt.Errorf("no expiries available for %s", args[0])
			}
			expiry = exps[0].MaturityDate
		}
		rows, err := app.client.GetOptionChain(cmd.Context(), args[0], expiry)
		if err != nil {
			return userFacingCommandError(err)
		}
		if chainWithPrices {
			codes := make([]string, 0, len(rows)*2)
			for _, r := range rows {
				if r.CallGuid != "" {
					codes = append(codes, r.CallGuid)
				}
				if r.PutGuid != "" {
					codes = append(codes, r.PutGuid)
				}
			}
			prices, err := app.client.GetOptionPrices(cmd.Context(), codes)
			if err != nil {
				return userFacingCommandError(err)
			}
			priceByCode := make(map[string]*domain.OptionPrice, len(prices))
			for i := range prices {
				priceByCode[prices[i].Code] = &prices[i]
			}
			for i := range rows {
				if p, ok := priceByCode[rows[i].CallGuid]; ok {
					rows[i].CallPrice = p
				}
				if p, ok := priceByCode[rows[i].PutGuid]; ok {
					rows[i].PutPrice = p
				}
			}
		}
		switch strings.ToLower(chainType) {
		case "", "both":
			// no filter
		case "call":
			for i := range rows {
				rows[i].PutGuid = ""
				rows[i].PutPrice = nil
				rows[i].PutOpenInterest = 0
			}
		case "put":
			for i := range rows {
				rows[i].CallGuid = ""
				rows[i].CallPrice = nil
				rows[i].CallOpenInterest = 0
			}
		default:
			return fmt.Errorf("--type must be call/put/both (got %q)", chainType)
		}
		return output.WriteOptionChain(cmd.OutOrStdout(), app.format, rows)
	},
}
chainCmd.Flags().StringVar(&chainExpiry, "expiry", "", "Expiry date YYYY-MM-DD (default: nearest)")
chainCmd.Flags().StringVar(&chainType, "type", "both", "Filter: call / put / both")
chainCmd.Flags().BoolVar(&chainWithPrices, "with-prices", false, "Join bulk prices into the chain rows (extra round-trip)")

pricesCmd := &cobra.Command{
	Use:   "prices <code> [<code>...]",
	Short: "Bulk option/stock prices (lighter than quote get)",
	Long: `Fetch a flat price list for one or many productCodes (typically OPT_ codes).

Example:
  tossctl options prices OPT_SNDK260515C01395000_20260506 OPT_SNDK260515P01395000_20260506`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := newAppContext(opts)
		if err != nil {
			return err
		}
		prices, err := app.client.GetOptionPrices(cmd.Context(), args)
		if err != nil {
			return userFacingCommandError(err)
		}
		return output.WriteOptionPrices(cmd.OutOrStdout(), app.format, prices)
	},
}

cmd.AddCommand(infoCmd, atmCmd, expiriesCmd, chainCmd, pricesCmd)
```

Make sure the file imports `fmt`, `strings`, and the `domain` package alongside the existing `cobra` / `output`.

- [ ] **Step 2: Build + help check**

```
go build ./...
go run ./cmd/tossctl options --help
go run ./cmd/tossctl options expiries --help
go run ./cmd/tossctl options chain --help
go run ./cmd/tossctl options prices --help
```

- [ ] **Step 3: Full regression**

```
go vet ./...
go test ./... -count=1
```

- [ ] **Step 4: Commit**

```bash
git add cmd/tossctl/options.go
git commit -m "feat(cli): tossctl options expiries / chain / prices"
```

---

## Task 7: Docs — CHANGELOG + brief reference update

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Append to Unreleased / Added**

```markdown
- `tossctl options expiries <underlying>` — list option expiry ladder via `/api/v1/option-maturity-date/get-all`.
- `tossctl options chain <underlying> [--expiry] [--type call|put] [--with-prices]` — strike chain (call + put per strike) via `/api/v1/option-both-chain/get-all`; optional bulk-price join.
- `tossctl options prices <codes>` — bulk option/stock prices via `/api/v2/stock-prices?codes=…` (lighter than `quote get`).
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): note option chain commands"
```

---

## Self-review notes

- All three endpoints are public GET. No auth changes required.
- `option-both-chain/get-all` `result` is a flat array — easy to confuse with `option-maturity-date/get-all` (which nests under `result.items`). Tests verify both shapes.
- `chain --with-prices` issues two round-trips (chain + bulk prices) — acceptable since the chain endpoint returns no prices and the bulk endpoint is the same one Toss UI uses.
- `--type call|put` filtering is client-side because the server returns both sides in one call; sending the filter back would waste the row we already have.
- No `--follow` for chain in PR7. Could be added later by polling `GetOptionPrices` and dedup'ing on `(code, close, volume)` — same pattern as the PR5 stream types — but adds scope. Defer until a user asks.
- Memory `options-scope-read-only` still applies — no option trading code added.

---

## Execution

Subagent-driven, fresh subagent per task. After Task 7, merge `--no-ff` back into `feat/order-page-integration`, push to fork only.
