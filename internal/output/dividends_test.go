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

func dividendsTestFixture() domain.StockDividends {
	growthRatio := 0.0179428877941688
	dpsKrw := 1551.0
	cashKrw := 1536.0
	return domain.StockDividends{
		ProductCode: "US19801212001",
		Summary: domain.DividendYieldCard{
			DividendCount:         4,
			DividendMonths:        []int{2, 5, 8, 11},
			DividendCash:          1.03,
			DividendCashKrw:       &cashKrw,
			DividendYieldRatio:    0.0035,
			TTMDividendYieldRatio: 0.0035,
			TTMDividendMonths:     []string{"2025-05", "2025-08", "2025-11", "2026-02"},
			TTMDps:                1.04,
			TTMDpsKrw:             &dpsKrw,
			TTMDividendTotalCount: 4,
			DividendGrowthRatio:   &growthRatio,
			Currency:              "USD",
		},
		RecentYears: domain.DividendYearsPayouts{
			StartDate: "2023-01-01", RangeLabel: "3년",
			Payouts: []domain.DividendPayout{
				{ExDate: "2026-02-09", PaymentDate: "2026-02-13", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0039},
				{ExDate: "2026-05-09", PaymentDate: "2026-05-15", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0035},
			},
			TotalCash: 1.04, TotalCashKrw: 1551,
		},
		FullHistory: []domain.DividendPayout{
			{ExDate: "2026-02-09", PaymentDate: "2026-02-13", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0039},
			{ExDate: "2026-05-09", PaymentDate: "2026-05-15", Currency: "USD", Cash: 0.26, CashKrw: 388, YieldRatio: 0.0009, TTMYieldRatio: 0.0035},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockDividendsTable(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, false); err != nil {
		t.Fatalf("WriteStockDividends error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"US19801212001", "USD",
		"TTM yield", "0.35%", "TTM dps", "$1.04",
		"+1.79%", // growth ratio
		"3년", "2026-05-09", "$0.26",
		"Total: $1.04",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
	// full-history flag off → should NOT include separate "전체 히스토리" header
	if strings.Contains(out, "전체 히스토리") {
		t.Fatalf("expected no full-history section when flag is false; got:\n%s", out)
	}
}

func TestWriteStockDividendsTableWithAllHistory(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, true); err != nil {
		t.Fatalf("WriteStockDividends error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "전체 히스토리") {
		t.Fatalf("expected full-history section when flag is true; got:\n%s", out)
	}
}

func TestWriteStockDividendsTableEmpty(t *testing.T) {
	t.Parallel()
	div := domain.StockDividends{
		ProductCode: "NAS0250224006",
		Summary:     domain.DividendYieldCard{DividendCount: 0, Currency: ""},
		RecentYears: domain.DividendYearsPayouts{RangeLabel: "3년"},
		FullHistory: nil,
		FetchedAt:   time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatTable, div, false); err != nil {
		t.Fatalf("WriteStockDividends empty error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No dividends") {
		t.Fatalf("expected 'No dividends' message for empty case; got:\n%s", out)
	}
}

func TestWriteStockDividendsCSV(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatCSV, div, true); err != nil {
		t.Fatalf("WriteStockDividends CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) < 3 {
		t.Fatalf("expected header + multiple rows, got %d", len(rows))
	}
	if rows[0][0] != "section" {
		t.Fatalf("expected first header col 'section', got %q", rows[0][0])
	}
}

func TestWriteStockDividendsJSON(t *testing.T) {
	t.Parallel()
	div := dividendsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockDividends(&buf, FormatJSON, div, true); err != nil {
		t.Fatalf("WriteStockDividends JSON error: %v", err)
	}
	var got domain.StockDividends
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, buf.String())
	}
	if got.ProductCode != div.ProductCode {
		t.Fatalf("product_code roundtrip mismatch: %q vs %q", got.ProductCode, div.ProductCode)
	}
	// Verify snake_case in raw JSON
	if !strings.Contains(buf.String(), `"ttm_dividend_total_count"`) {
		t.Fatalf("expected snake_case ttm_dividend_total_count in JSON")
	}
}
