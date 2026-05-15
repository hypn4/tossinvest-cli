package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

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
