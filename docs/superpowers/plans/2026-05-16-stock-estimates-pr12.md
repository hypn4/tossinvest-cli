# PR12 — Stock estimates (revenue + EPS + operating-income forecasts)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add `tossctl stock estimates <sym>` covering four endpoints:
- `GET /api/v2/companies/{code}/financial/estimate/date` — next-earnings headline (announceAt, current revenueEst/epsEst/operatingIncomeEst)
- `POST /api/v2/companies/{code}/financial/estimate/revenue` (body `{}`) — revenue time series with actual + est + surprise
- `POST /api/v2/companies/{code}/financial/estimate/eps` (body `{}`) — EPS time series
- `POST /api/v2/companies/{code}/financial/estimate/operating-income` (body `{}`) — operating-income time series

All four use `{stockCode}` (productCode), not companyCode. The `estimate/operating-income` endpoint can return all-null estimates for stocks without analyst coverage (e.g. SNDK shows null position, null operatingIncomeEst); the others return populated data for analyst-covered stocks.

**Architecture:** One client method `GetStockEstimates` makes 1 GET + 3 sequential `postJSONEmpty` calls and stitches into `domain.StockEstimates`. Output writer renders headline + 3 time-series tables.

**Tech Stack:** Go, cobra, postJSONEmpty + getJSON, encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `6197838`.

---

## File Structure

**New:**
- `internal/client/estimates.go` + `_test.go` — `GetStockEstimates`
- `internal/output/estimates.go` + `_test.go` — `WriteStockEstimates`
- `fixtures/responses/public/stock-estimates-sndk.json` — bundles all 4 responses

**Modified:**
- `internal/domain/models.go` — append `StockEstimates`, `EstimateHeadline`, `EstimateRevenueSeries`, `EstimateRevenuePoint`, `EstimateEpsSeries`, `EstimateEpsPoint`, `EstimateOperatingIncomeSeries`, `EstimateOperatingIncomePoint`
- `cmd/tossctl/stock.go` — register `estimatesCmd`
- `fixtures/responses/public/manifest.json` — append fixture entry
- `CHANGELOG.md` — add line

---

## Task 1: Domain types + client + fixture + test

### Step 1: Save fixture

Create `fixtures/responses/public/stock-estimates-sndk.json`. Trim each `graphs` array to last 4 quarters:

```json
{
  "date": {
    "result": {
      "announceAt": null,
      "revenueEst": 7736730000.0,
      "revenueEstKrw": 11567185023000.0,
      "epsEst": 31.979,
      "epsEstKrw": 47811.8029,
      "operatingIncomeEst": null,
      "operatingIncomeEstKrw": null
    }
  },
  "revenue": {
    "result": {
      "revenueEst": 7736730000.0,
      "revenueEstKrw": 11567185023000.0,
      "fluctuationRate": 26.52,
      "fluctuation": 1247330000.0,
      "fluctuationKrw": 1894444804000.0,
      "position": "HIGH",
      "graphs": [
        {"period": "2025-06", "revenue": 1923000000.0, "revenueKrw": 2607132000000.0, "revenueEst": 1872000000.0, "revenueEstKrw": 2538013600000.0, "surprise": 2.72},
        {"period": "2025-09", "revenue": 2308000000.0, "revenueKrw": 3243160800000.0, "revenueEst": 2210000000.0, "revenueEstKrw": 3105410800000.0, "surprise": 4.43},
        {"period": "2025-12", "revenue": 2306000000.0, "revenueKrw": 3380840800000.0, "revenueEst": 2189000000.0, "revenueEstKrw": 3209357300000.0, "surprise": 5.34},
        {"period": "2026-03", "revenue": 3092000000.0, "revenueKrw": 4699192000000.0, "revenueEst": 2440000000.0, "revenueEstKrw": 3708680000000.0, "surprise": 26.72}
      ]
    }
  },
  "eps": {
    "result": {
      "epsEst": 31.979,
      "epsEstKrw": 47811.8029,
      "fluctuationRate": 36.6,
      "fluctuation": 8.569,
      "fluctuationKrw": 12256.6949,
      "position": "HIGH",
      "graphs": [
        {"period": "2025-06", "eps": 0.29, "epsKrw": 393.70, "epsEst": 0.041, "epsEstKrw": 55.66, "surprise": 607.32},
        {"period": "2025-09", "eps": 1.22, "epsKrw": 1715.32, "epsEst": 0.927, "epsEstKrw": 1303.36, "surprise": 31.61},
        {"period": "2025-12", "eps": 6.2, "epsKrw": 8896.38, "epsEst": 3.206, "epsEstKrw": 4600.29, "surprise": 93.39},
        {"period": "2026-03", "eps": 23.41, "epsKrw": 35530.43, "epsEst": 11.5, "epsEstKrw": 17480.0, "surprise": 103.57}
      ]
    }
  },
  "operatingIncome": {
    "result": {
      "operatingIncomeEst": null,
      "operatingIncomeEstKrw": null,
      "fluctuationRate": 0,
      "fluctuation": 0,
      "fluctuationKrw": 0,
      "position": null,
      "graphs": [
        {"period": "2025-06", "operatingIncome": 18000000.0, "operatingIncomeKrw": 24436800000.0, "operatingIncomeEst": null, "operatingIncomeEstKrw": null, "surprise": null},
        {"period": "2025-09", "operatingIncome": 879000000.0, "operatingIncomeKrw": 1234950600000.0, "operatingIncomeEst": null, "operatingIncomeEstKrw": null, "surprise": null},
        {"period": "2025-12", "operatingIncome": 1067000000.0, "operatingIncomeKrw": 1564265600000.0, "operatingIncomeEst": null, "operatingIncomeEstKrw": null, "surprise": null},
        {"period": "2026-03", "operatingIncome": 4111000000.0, "operatingIncomeKrw": 6243786800000.0, "operatingIncomeEst": null, "operatingIncomeEstKrw": null, "surprise": null}
      ]
    }
  }
}
```

### Step 2: Update manifest

Append:

```json
{
  "file": "stock-estimates-sndk.json",
  "url": "(combined estimate-date+revenue+eps+operating-income fixture for testing only)",
  "method": "GET+POST"
}
```

### Step 3: Add domain types

Append to `internal/domain/models.go`:

```go
type StockEstimates struct {
	ProductCode     string                       `json:"product_code"`
	Headline        EstimateHeadline             `json:"headline"`
	Revenue         EstimateRevenueSeries        `json:"revenue"`
	EPS             EstimateEpsSeries            `json:"eps"`
	OperatingIncome EstimateOperatingIncomeSeries `json:"operating_income"`
	FetchedAt       time.Time                    `json:"fetched_at"`
}

// EstimateHeadline is the consensus-summary card from GET .../financial/estimate/date.
type EstimateHeadline struct {
	AnnounceAt            *string  `json:"announce_at"`
	RevenueEst            *float64 `json:"revenue_est"`
	RevenueEstKrw         *float64 `json:"revenue_est_krw"`
	EPSEst                *float64 `json:"eps_est"`
	EPSEstKrw             *float64 `json:"eps_est_krw"`
	OperatingIncomeEst    *float64 `json:"operating_income_est"`
	OperatingIncomeEstKrw *float64 `json:"operating_income_est_krw"`
}

type EstimateRevenueSeries struct {
	RevenueEst      *float64               `json:"revenue_est"`
	RevenueEstKrw   *float64               `json:"revenue_est_krw"`
	FluctuationRate float64                `json:"fluctuation_rate"`
	Fluctuation     float64                `json:"fluctuation"`
	FluctuationKrw  float64                `json:"fluctuation_krw"`
	Position        *string                `json:"position"`
	Graph           []EstimateRevenuePoint `json:"graph"`
}

type EstimateRevenuePoint struct {
	Period        string   `json:"period"`
	Revenue       *float64 `json:"revenue"`
	RevenueEst    *float64 `json:"revenue_est"`
	RevenueKrw    *float64 `json:"revenue_krw"`
	RevenueEstKrw *float64 `json:"revenue_est_krw"`
	Surprise      *float64 `json:"surprise"`
}

type EstimateEpsSeries struct {
	EPSEst          *float64           `json:"eps_est"`
	EPSEstKrw       *float64           `json:"eps_est_krw"`
	FluctuationRate float64            `json:"fluctuation_rate"`
	Fluctuation     float64            `json:"fluctuation"`
	FluctuationKrw  float64            `json:"fluctuation_krw"`
	Position        *string            `json:"position"`
	Graph           []EstimateEpsPoint `json:"graph"`
}

type EstimateEpsPoint struct {
	Period    string   `json:"period"`
	EPS       *float64 `json:"eps"`
	EPSEst    *float64 `json:"eps_est"`
	EPSKrw    *float64 `json:"eps_krw"`
	EPSEstKrw *float64 `json:"eps_est_krw"`
	Surprise  *float64 `json:"surprise"`
}

type EstimateOperatingIncomeSeries struct {
	OperatingIncomeEst    *float64                       `json:"operating_income_est"`
	OperatingIncomeEstKrw *float64                       `json:"operating_income_est_krw"`
	FluctuationRate       float64                        `json:"fluctuation_rate"`
	Fluctuation           float64                        `json:"fluctuation"`
	FluctuationKrw        float64                        `json:"fluctuation_krw"`
	Position              *string                        `json:"position"`
	Graph                 []EstimateOperatingIncomePoint `json:"graph"`
}

type EstimateOperatingIncomePoint struct {
	Period                string   `json:"period"`
	OperatingIncome       *float64 `json:"operating_income"`
	OperatingIncomeEst    *float64 `json:"operating_income_est"`
	OperatingIncomeKrw    *float64 `json:"operating_income_krw"`
	OperatingIncomeEstKrw *float64 `json:"operating_income_est_krw"`
	Surprise              *float64 `json:"surprise"`
}
```

### Step 4: Failing client test

Create `internal/client/estimates_test.go`:

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

func TestGetStockEstimatesFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Date            json.RawMessage `json:"date"`
		Revenue         json.RawMessage `json:"revenue"`
		EPS             json.RawMessage `json:"eps"`
		OperatingIncome json.RawMessage `json:"operatingIncome"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-estimates-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/companies/NAS0250224006/financial/estimate/date":
			w.Write(bundle.Date)
		case "/api/v2/companies/NAS0250224006/financial/estimate/revenue":
			w.Write(bundle.Revenue)
		case "/api/v2/companies/NAS0250224006/financial/estimate/eps":
			w.Write(bundle.EPS)
		case "/api/v2/companies/NAS0250224006/financial/estimate/operating-income":
			w.Write(bundle.OperatingIncome)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	est, err := c.GetStockEstimates(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockEstimates error: %v", err)
	}
	if est.Headline.RevenueEst == nil || *est.Headline.RevenueEst != 7736730000.0 {
		t.Fatalf("unexpected headline revenueEst: %v", est.Headline.RevenueEst)
	}
	if est.Headline.OperatingIncomeEst != nil {
		t.Fatalf("expected nil OperatingIncomeEst in headline; got %v", *est.Headline.OperatingIncomeEst)
	}
	if len(est.Revenue.Graph) != 4 {
		t.Fatalf("expected 4 revenue points, got %d", len(est.Revenue.Graph))
	}
	if est.Revenue.Position == nil || *est.Revenue.Position != "HIGH" {
		t.Fatalf("expected revenue position HIGH, got %v", est.Revenue.Position)
	}
	if len(est.EPS.Graph) != 4 {
		t.Fatalf("expected 4 EPS points, got %d", len(est.EPS.Graph))
	}
	if est.EPS.Graph[3].Surprise == nil || *est.EPS.Graph[3].Surprise < 100 {
		t.Fatalf("expected large positive surprise on last EPS point; got %v", est.EPS.Graph[3].Surprise)
	}
	// operating-income has null position (no analyst coverage for that metric)
	if est.OperatingIncome.Position != nil {
		t.Fatalf("expected nil OperatingIncome.Position; got %q", *est.OperatingIncome.Position)
	}
	if len(est.OperatingIncome.Graph) != 4 {
		t.Fatalf("expected 4 OI points, got %d", len(est.OperatingIncome.Graph))
	}
	// All OI points have null est
	for i, p := range est.OperatingIncome.Graph {
		if p.OperatingIncomeEst != nil {
			t.Fatalf("OI[%d] expected nil est, got %v", i, *p.OperatingIncomeEst)
		}
	}
}
```

### Step 5: Run test (FAIL)

Run: `go test ./internal/client/ -run TestGetStockEstimates -v`

### Step 6: Implement client

Create `internal/client/estimates.go`:

```go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type estimateDateEnvelope struct {
	Result struct {
		AnnounceAt            *string  `json:"announceAt"`
		RevenueEst            *float64 `json:"revenueEst"`
		RevenueEstKrw         *float64 `json:"revenueEstKrw"`
		EpsEst                *float64 `json:"epsEst"`
		EpsEstKrw             *float64 `json:"epsEstKrw"`
		OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
		OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
	} `json:"result"`
}

type estimateRevenueEnvelope struct {
	Result struct {
		RevenueEst      *float64 `json:"revenueEst"`
		RevenueEstKrw   *float64 `json:"revenueEstKrw"`
		FluctuationRate float64  `json:"fluctuationRate"`
		Fluctuation     float64  `json:"fluctuation"`
		FluctuationKrw  float64  `json:"fluctuationKrw"`
		Position        *string  `json:"position"`
		Graphs          []struct {
			Period        string   `json:"period"`
			Revenue       *float64 `json:"revenue"`
			RevenueEst    *float64 `json:"revenueEst"`
			RevenueKrw    *float64 `json:"revenueKrw"`
			RevenueEstKrw *float64 `json:"revenueEstKrw"`
			Surprise      *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

type estimateEpsEnvelope struct {
	Result struct {
		EpsEst          *float64 `json:"epsEst"`
		EpsEstKrw       *float64 `json:"epsEstKrw"`
		FluctuationRate float64  `json:"fluctuationRate"`
		Fluctuation     float64  `json:"fluctuation"`
		FluctuationKrw  float64  `json:"fluctuationKrw"`
		Position        *string  `json:"position"`
		Graphs          []struct {
			Period    string   `json:"period"`
			Eps       *float64 `json:"eps"`
			EpsEst    *float64 `json:"epsEst"`
			EpsKrw    *float64 `json:"epsKrw"`
			EpsEstKrw *float64 `json:"epsEstKrw"`
			Surprise  *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

type estimateOperatingIncomeEnvelope struct {
	Result struct {
		OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
		OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
		FluctuationRate       float64  `json:"fluctuationRate"`
		Fluctuation           float64  `json:"fluctuation"`
		FluctuationKrw        float64  `json:"fluctuationKrw"`
		Position              *string  `json:"position"`
		Graphs                []struct {
			Period                string   `json:"period"`
			OperatingIncome       *float64 `json:"operatingIncome"`
			OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
			OperatingIncomeKrw    *float64 `json:"operatingIncomeKrw"`
			OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
			Surprise              *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

// GetStockEstimates fetches the next-earnings consensus headline plus three
// time-series endpoints (revenue / EPS / operating-income estimates). The
// headline is a GET; the three series are POST with empty body. All four use
// productCode (not companyCode).
func (c *Client) GetStockEstimates(ctx context.Context, symbol string) (domain.StockEstimates, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockEstimates{}, err
	}

	base := fmt.Sprintf("%s/api/v2/companies/%s/financial/estimate", c.infoBaseURL, productCode)

	var hd estimateDateEnvelope
	if err := c.getJSON(ctx, base+"/date", &hd); err != nil {
		return domain.StockEstimates{}, err
	}

	var rev estimateRevenueEnvelope
	if err := c.postJSONEmpty(ctx, base+"/revenue", &rev); err != nil {
		return domain.StockEstimates{}, err
	}

	var eps estimateEpsEnvelope
	if err := c.postJSONEmpty(ctx, base+"/eps", &eps); err != nil {
		return domain.StockEstimates{}, err
	}

	var op estimateOperatingIncomeEnvelope
	if err := c.postJSONEmpty(ctx, base+"/operating-income", &op); err != nil {
		return domain.StockEstimates{}, err
	}

	revPoints := make([]domain.EstimateRevenuePoint, len(rev.Result.Graphs))
	for i, g := range rev.Result.Graphs {
		revPoints[i] = domain.EstimateRevenuePoint{
			Period: g.Period, Revenue: g.Revenue, RevenueEst: g.RevenueEst,
			RevenueKrw: g.RevenueKrw, RevenueEstKrw: g.RevenueEstKrw, Surprise: g.Surprise,
		}
	}
	epsPoints := make([]domain.EstimateEpsPoint, len(eps.Result.Graphs))
	for i, g := range eps.Result.Graphs {
		epsPoints[i] = domain.EstimateEpsPoint{
			Period: g.Period, EPS: g.Eps, EPSEst: g.EpsEst,
			EPSKrw: g.EpsKrw, EPSEstKrw: g.EpsEstKrw, Surprise: g.Surprise,
		}
	}
	opPoints := make([]domain.EstimateOperatingIncomePoint, len(op.Result.Graphs))
	for i, g := range op.Result.Graphs {
		opPoints[i] = domain.EstimateOperatingIncomePoint{
			Period: g.Period, OperatingIncome: g.OperatingIncome, OperatingIncomeEst: g.OperatingIncomeEst,
			OperatingIncomeKrw: g.OperatingIncomeKrw, OperatingIncomeEstKrw: g.OperatingIncomeEstKrw, Surprise: g.Surprise,
		}
	}

	return domain.StockEstimates{
		ProductCode: productCode,
		Headline: domain.EstimateHeadline{
			AnnounceAt:            hd.Result.AnnounceAt,
			RevenueEst:            hd.Result.RevenueEst,
			RevenueEstKrw:         hd.Result.RevenueEstKrw,
			EPSEst:                hd.Result.EpsEst,
			EPSEstKrw:             hd.Result.EpsEstKrw,
			OperatingIncomeEst:    hd.Result.OperatingIncomeEst,
			OperatingIncomeEstKrw: hd.Result.OperatingIncomeEstKrw,
		},
		Revenue: domain.EstimateRevenueSeries{
			RevenueEst: rev.Result.RevenueEst, RevenueEstKrw: rev.Result.RevenueEstKrw,
			FluctuationRate: rev.Result.FluctuationRate, Fluctuation: rev.Result.Fluctuation, FluctuationKrw: rev.Result.FluctuationKrw,
			Position: rev.Result.Position, Graph: revPoints,
		},
		EPS: domain.EstimateEpsSeries{
			EPSEst: eps.Result.EpsEst, EPSEstKrw: eps.Result.EpsEstKrw,
			FluctuationRate: eps.Result.FluctuationRate, Fluctuation: eps.Result.Fluctuation, FluctuationKrw: eps.Result.FluctuationKrw,
			Position: eps.Result.Position, Graph: epsPoints,
		},
		OperatingIncome: domain.EstimateOperatingIncomeSeries{
			OperatingIncomeEst: op.Result.OperatingIncomeEst, OperatingIncomeEstKrw: op.Result.OperatingIncomeEstKrw,
			FluctuationRate: op.Result.FluctuationRate, Fluctuation: op.Result.Fluctuation, FluctuationKrw: op.Result.FluctuationKrw,
			Position: op.Result.Position, Graph: opPoints,
		},
		FetchedAt: time.Now().UTC(),
	}, nil
}
```

### Step 7: Run test (PASS)

### Step 8: Commit

```bash
git add internal/domain/models.go internal/client/estimates.go internal/client/estimates_test.go fixtures/responses/public/stock-estimates-sndk.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add GetStockEstimates (headline + revenue/eps/op-income forecasts)"
```

---

## Task 2: Output writer + cobra + CHANGELOG

### Step 1: Failing output test

Create `internal/output/estimates_test.go`:

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

func estimatesTestFixture() domain.StockEstimates {
	announce := "2026-07-30"
	revEst := 7736730000.0
	revEstKrw := 11567185023000.0
	epsEst := 31.979
	epsEstKrw := 47811.8029
	posHigh := "HIGH"
	supr1 := 26.72
	supr2 := 103.57
	oiVal1 := 1067000000.0
	oiVal2 := 4111000000.0
	r1 := 2306000000.0
	r2 := 3092000000.0
	re1 := 2189000000.0
	re2 := 2440000000.0
	e1 := 6.2
	e2 := 23.41
	ee1 := 3.206
	ee2 := 11.5
	return domain.StockEstimates{
		ProductCode: "NAS0250224006",
		Headline: domain.EstimateHeadline{
			AnnounceAt:    &announce,
			RevenueEst:    &revEst,
			RevenueEstKrw: &revEstKrw,
			EPSEst:        &epsEst,
			EPSEstKrw:     &epsEstKrw,
		},
		Revenue: domain.EstimateRevenueSeries{
			RevenueEst: &revEst, RevenueEstKrw: &revEstKrw,
			FluctuationRate: 26.52, Fluctuation: 1247330000.0, FluctuationKrw: 1894444804000.0,
			Position: &posHigh,
			Graph: []domain.EstimateRevenuePoint{
				{Period: "2025-12", Revenue: &r1, RevenueEst: &re1, Surprise: &supr1},
				{Period: "2026-03", Revenue: &r2, RevenueEst: &re2, Surprise: &supr1},
			},
		},
		EPS: domain.EstimateEpsSeries{
			EPSEst: &epsEst, EPSEstKrw: &epsEstKrw,
			FluctuationRate: 36.6, Fluctuation: 8.569, FluctuationKrw: 12256.6949,
			Position: &posHigh,
			Graph: []domain.EstimateEpsPoint{
				{Period: "2025-12", EPS: &e1, EPSEst: &ee1, Surprise: &supr2},
				{Period: "2026-03", EPS: &e2, EPSEst: &ee2, Surprise: &supr2},
			},
		},
		OperatingIncome: domain.EstimateOperatingIncomeSeries{
			OperatingIncomeEst:    nil,
			OperatingIncomeEstKrw: nil,
			Position:              nil,
			Graph: []domain.EstimateOperatingIncomePoint{
				{Period: "2025-12", OperatingIncome: &oiVal1},
				{Period: "2026-03", OperatingIncome: &oiVal2},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockEstimatesTable(t *testing.T) {
	t.Parallel()
	est := estimatesTestFixture()
	var buf bytes.Buffer
	if err := WriteStockEstimates(&buf, FormatTable, est); err != nil {
		t.Fatalf("WriteStockEstimates error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"NAS0250224006",
		"Announce", "2026-07-30",
		"Revenue est",
		"=== Revenue forecast",
		"HIGH",
		"=== EPS forecast",
		"$31.98",
		"=== Operating-income forecast",
		"(no analyst coverage)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockEstimatesCSV(t *testing.T) {
	t.Parallel()
	est := estimatesTestFixture()
	var buf bytes.Buffer
	if err := WriteStockEstimates(&buf, FormatCSV, est); err != nil {
		t.Fatalf("WriteStockEstimates CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) < 4 {
		t.Fatalf("expected at least header + multiple rows, got %d", len(rows))
	}
	if rows[0][0] != "section" {
		t.Fatalf("expected first header col 'section', got %q", rows[0][0])
	}
	sections := map[string]bool{}
	for _, r := range rows[1:] {
		sections[r[0]] = true
	}
	for _, want := range []string{"revenue", "eps", "operating_income"} {
		if !sections[want] {
			t.Fatalf("CSV missing section %q; sections=%v", want, sections)
		}
	}
}

func TestWriteStockEstimatesJSON(t *testing.T) {
	t.Parallel()
	est := estimatesTestFixture()
	var buf bytes.Buffer
	if err := WriteStockEstimates(&buf, FormatJSON, est); err != nil {
		t.Fatalf("WriteStockEstimates JSON error: %v", err)
	}
	var got domain.StockEstimates
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, buf.String())
	}
	if got.ProductCode != est.ProductCode {
		t.Fatalf("product_code roundtrip mismatch")
	}
	if !strings.Contains(buf.String(), `"announce_at"`) {
		t.Fatalf("expected snake_case announce_at in JSON")
	}
	if !strings.Contains(buf.String(), `"operating_income_est"`) {
		t.Fatalf("expected snake_case operating_income_est in JSON")
	}
}
```

### Step 2: Run test (FAIL)

### Step 3: Implement output writer

Create `internal/output/estimates.go`:

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

// WriteStockEstimates renders the headline consensus + 3 time-series tables
// (revenue, EPS, operating-income forecasts).
func WriteStockEstimates(w io.Writer, format Format, est domain.StockEstimates) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(est)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "period", "actual_usd", "estimate_usd", "surprise_pct"}); err != nil {
			return err
		}
		for _, p := range est.Revenue.Graph {
			if err := cw.Write([]string{"revenue", p.Period, fmtFloatPtr(p.Revenue), fmtFloatPtr(p.RevenueEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		for _, p := range est.EPS.Graph {
			if err := cw.Write([]string{"eps", p.Period, fmtFloatPtr(p.EPS), fmtFloatPtr(p.EPSEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		for _, p := range est.OperatingIncome.Graph {
			if err := cw.Write([]string{"operating_income", p.Period, fmtFloatPtr(p.OperatingIncome), fmtFloatPtr(p.OperatingIncomeEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — analyst estimates\n", est.ProductCode); err != nil {
			return err
		}

		fmt.Fprintln(w, "\n=== Consensus headline ===")
		hd := est.Headline
		announce := "—"
		if hd.AnnounceAt != nil {
			announce = *hd.AnnounceAt
		}
		fmt.Fprintf(w, "Announce: %s\n", announce)
		hdHeaders := []string{"FIELD", "USD", "KRW"}
		hdRows := [][]string{
			{"Revenue est", fmtUSDLargePtr(hd.RevenueEst), fmtKrwPtr(hd.RevenueEstKrw)},
			{"EPS est", fmtUSDSmallPtr(hd.EPSEst), fmtKrwPtr(hd.EPSEstKrw)},
			{"Operating-income est", fmtUSDLargePtr(hd.OperatingIncomeEst), fmtKrwPtr(hd.OperatingIncomeEstKrw)},
		}
		if err := renderTable(w, hdHeaders, hdRows); err != nil {
			return err
		}

		renderRevenueSection(w, est.Revenue)
		renderEpsSection(w, est.EPS)
		renderOpIncomeSection(w, est.OperatingIncome)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func renderRevenueSection(w io.Writer, s domain.EstimateRevenueSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== Revenue forecast (next %s) ===\n", pos)
	fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDLargePtr(s.RevenueEst), s.FluctuationRate)
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDLargePtr(p.Revenue), fmtUSDLargePtr(p.RevenueEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func renderEpsSection(w io.Writer, s domain.EstimateEpsSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== EPS forecast (next %s) ===\n", pos)
	fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDSmallPtr(s.EPSEst), s.FluctuationRate)
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDSmallPtr(p.EPS), fmtUSDSmallPtr(p.EPSEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func renderOpIncomeSection(w io.Writer, s domain.EstimateOperatingIncomeSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== Operating-income forecast (%s) ===\n", pos)
	if s.OperatingIncomeEst == nil {
		fmt.Fprintln(w, "Next-period est: (no analyst coverage)")
	} else {
		fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDLargePtr(s.OperatingIncomeEst), s.FluctuationRate)
	}
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDLargePtr(p.OperatingIncome), fmtUSDLargePtr(p.OperatingIncomeEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func positionLabel(p *string) string {
	if p == nil {
		return "no analyst coverage"
	}
	return *p
}

func fmtUSDLargePtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return formatUSDLarge(*p)
}

func fmtUSDSmallPtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("$%.2f", *p)
}

func fmtKrwPtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return "₩" + formatWithCommas(int64(*p))
}

func fmtSurprisePtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%+.2f%%", *p)
}

func fmtFloatPtr(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', 2, 64)
}
```

### Step 4: Run test (PASS)

### Step 5: Register cobra subcommand

In `cmd/tossctl/stock.go`, after `dividendsCmd`:

```go
estimatesCmd := &cobra.Command{
    Use:   "estimates <symbol>",
    Short: "Analyst forecast snapshot (revenue + EPS + operating-income estimates vs actuals)",
    Long: `Fetch analyst estimates from four endpoints:
  - GET  /api/v2/companies/{code}/financial/estimate/date — next-earnings headline
  - POST /api/v2/companies/{code}/financial/estimate/revenue (body {}) — revenue time series
  - POST /api/v2/companies/{code}/financial/estimate/eps (body {}) — EPS time series
  - POST /api/v2/companies/{code}/financial/estimate/operating-income (body {}) — OI time series

Each time-series point pairs the actual value with the consensus estimate
and the surprise % (actual / est − 1). When a stock has no analyst coverage
for a metric (e.g. small-cap operating-income), the est fields are nil and
the section prints "(no analyst coverage)".

Examples:
  tossctl stock estimates SNDK
  tossctl stock estimates NAS0250224006 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        est, err := app.client.GetStockEstimates(cmd.Context(), args[0])
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockEstimates(cmd.OutOrStdout(), app.format, est)
    },
}
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd, dividendsCmd, estimatesCmd)
```

### Step 6: CHANGELOG entry

In `CHANGELOG.md`, after the `stock dividends` line, add:

```markdown
- `tossctl stock estimates <symbol>` — analyst forecast snapshot (next-earnings headline + revenue/EPS/operating-income time series with surprise %) via `/api/v2/companies/{code}/financial/estimate/{date,revenue,eps,operating-income}`. Stocks without analyst coverage for a metric render `(no analyst coverage)` for that section.
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`

### Step 8: Commit

```bash
git add internal/output/estimates.go internal/output/estimates_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock estimates (headline + revenue/EPS/OI forecasts)"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock estimates SNDK` prints 4 sections; operating-income shows `(no analyst coverage)`.
- [ ] `/tmp/tossctl stock estimates AAPL` prints 4 fully-populated sections.
- [ ] `/tmp/tossctl stock estimates SNDK --output json | head -30` shows snake_case keys including `announce_at`, `operating_income_est`.
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 4 endpoints (1 GET + 3 POST) ✅
- Headline + 3 series ✅
- Nullable handling throughout (pointer types in domain + envelope) ✅
- snake_case JSON tags ✅
- `encoding/csv` for CSV ✅
- CHANGELOG ✅

**Placeholder scan:** none.

**Type consistency:** All nullable fields are `*float64` / `*string`. Output helpers (`fmtUSDLargePtr`, `fmtKrwPtr`, `fmtSurprisePtr`, `positionLabel`) consistently return `"—"` or `"(no analyst coverage)"` for nil.
