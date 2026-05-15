package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteSalesCompositionCSVQuotesCommaBusiness(t *testing.T) {
	t.Parallel()
	sc := domain.SalesComposition{
		ProductCode: "NAS0250224006",
		Items: []domain.SalesCompositionItem{
			{Business: "클라우드, 엔터프라이즈", Product: "스토리지", Ratio: 42.50},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteSalesComposition(&buf, FormatCSV, sc); err != nil {
		t.Fatalf("WriteSalesComposition CSV error: %v", err)
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
	if rows[1][0] != "클라우드, 엔터프라이즈" {
		t.Fatalf("expected business preserved as %q, got %q", "클라우드, 엔터프라이즈", rows[1][0])
	}
}

func TestWriteSalesCompositionTable(t *testing.T) {
	t.Parallel()
	sc := domain.SalesComposition{
		ProductCode: "NAS0250224006",
		CompanyCode: "NAS116LTR-E0",
		FiscalYear:  2025,
		EndDate:     "2025-06-30",
		Items: []domain.SalesCompositionItem{
			{Business: "클라이언트", Ratio: 56.11},
			{Business: "소비자", Ratio: 30.84},
			{Business: "클라우드 서비스", Ratio: 13.05},
		},
		DataSource: "출처: 연합인포맥스 및 기업 IR자료",
		FetchedAt:  time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteSalesComposition(&buf, FormatTable, sc); err != nil {
		t.Fatalf("WriteSalesComposition error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "FY2025", "2025-06-30", "클라이언트", "56.11", "소비자", "30.84", "클라우드 서비스", "13.05", "연합인포맥스"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
