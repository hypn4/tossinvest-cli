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

func financialsTestFixture() domain.StockFinancials {
	return domain.StockFinancials{
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
}

func TestWriteStockFinancialsTable(t *testing.T) {
	t.Parallel()
	fin := financialsTestFixture()
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

func TestWriteStockFinancialsCSV(t *testing.T) {
	t.Parallel()
	fin := financialsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockFinancials(&buf, FormatCSV, fin); err != nil {
		t.Fatalf("WriteStockFinancials CSV error: %v", err)
	}
	// Parse the CSV back
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) < 4 {
		t.Fatalf("expected at least header + 3 rows, got %d", len(rows))
	}
	if rows[0][0] != "section" {
		t.Fatalf("expected first header col 'section', got %q", rows[0][0])
	}
	// Find one row per section to verify all three discriminators appear
	sections := map[string]bool{}
	for _, r := range rows[1:] {
		sections[r[0]] = true
	}
	for _, want := range []string{"revenue", "operating_income", "stability"} {
		if !sections[want] {
			t.Fatalf("CSV missing section %q; sections=%v", want, sections)
		}
	}
}

func TestWriteStockFinancialsJSON(t *testing.T) {
	t.Parallel()
	fin := financialsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockFinancials(&buf, FormatJSON, fin); err != nil {
		t.Fatalf("WriteStockFinancials JSON error: %v", err)
	}
	// Roundtrip
	var got domain.StockFinancials
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, buf.String())
	}
	if got.ProductCode != fin.ProductCode {
		t.Fatalf("product_code roundtrip mismatch: %q vs %q", got.ProductCode, fin.ProductCode)
	}
	if got.Stability.Position != fin.Stability.Position {
		t.Fatalf("stability.position roundtrip mismatch")
	}
	if len(got.Revenue.Graph) != len(fin.Revenue.Graph) {
		t.Fatalf("revenue.graph length mismatch: %d vs %d", len(got.Revenue.Graph), len(fin.Revenue.Graph))
	}
	// Also verify snake_case in raw JSON
	if !strings.Contains(buf.String(), `"product_code"`) {
		t.Fatalf("expected snake_case product_code in JSON, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), `"liability_ratio"`) {
		t.Fatalf("expected snake_case liability_ratio in JSON")
	}
}
