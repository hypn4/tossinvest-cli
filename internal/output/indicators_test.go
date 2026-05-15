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
	for _, want := range []string{"NAS0250224006", "가치평가", "PER", "45.4배", "수익", "EPS", "$28.76", "ROE", "39.3%", "배당", "0.00%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
