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
		"13,100,000,000", // amount comma-formatted
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
