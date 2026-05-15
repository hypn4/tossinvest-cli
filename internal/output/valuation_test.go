package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockValuationCSVQuotesCommaName(t *testing.T) {
	t.Parallel()
	val := domain.StockValuation{
		ProductCode: "NAS0250224006",
		Factor:      "PER",
		Peers: []domain.PeerValuation{
			{ProductCode: "US19990122001", Name: "Acme, Inc.", Value: 47.55, Period: "26년 1분기", IsSelf: false},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteStockValuation(&buf, FormatCSV, val); err != nil {
		t.Fatalf("WriteStockValuation CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) != 2 { // header + 1 data row
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if len(rows[1]) != 6 {
		t.Fatalf("expected 6 fields in data row, got %d: %v", len(rows[1]), rows[1])
	}
	if rows[1][1] != "Acme, Inc." {
		t.Fatalf("expected name preserved as %q, got %q", "Acme, Inc.", rows[1][1])
	}
}

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
