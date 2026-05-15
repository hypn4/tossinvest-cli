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
