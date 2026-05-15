package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockIndicatorsCSVQuotesCommaValues(t *testing.T) {
	t.Parallel()
	ind := domain.StockIndicators{
		ProductCode: "NAS0250224006",
		Sections: map[string]domain.IndicatorFields{
			"가치평가": {"displayPer": "123,456배"},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockIndicators(&buf, FormatCSV, ind); err != nil {
		t.Fatalf("WriteStockIndicators CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) != 2 { // header + 1 data row
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if len(rows[1]) != 3 {
		t.Fatalf("expected 3 fields in data row, got %d: %v", len(rows[1]), rows[1])
	}
	if rows[1][2] != "123,456배" {
		t.Fatalf("expected value preserved as %q, got %q", "123,456배", rows[1][2])
	}
}

func TestWriteStockIndicatorsTable(t *testing.T) {
	t.Parallel()
	ind := domain.StockIndicators{
		ProductCode: "NAS0250224006",
		Sections: map[string]domain.IndicatorFields{
			"가치평가": {"displayPer": "45.4배", "displayPbr": "14.9배", "displayPsr": "15.5배"},
			"수익":   {"eps": 28.76, "epsKrw": float64(42904), "bps": 93.08, "bpsKrw": float64(138856), "roe": "39.3%"},
			"배당":   {"dividendYieldRatio": float64(0), "annualCash": nil},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockIndicators(&buf, FormatTable, ind); err != nil {
		t.Fatalf("WriteStockIndicators error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "가치평가", "PER", "45.4배", "수익", "EPS", "$28.76", "ROE", "39.3%", "배당", "0.00%", "연간 배당금", "—"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
