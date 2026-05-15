package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteTICSIndustryTable(t *testing.T) {
	t.Parallel()
	ind := domain.TICSIndustry{
		ProductCode: "NAS0250224006",
		CompanyCode: "NAS116LTR-E0",
		Major: []domain.TICSEntry{
			{
				ID: 209, Title: "컴퓨터와 주변기기", Description: "sd카드, usb 메모리 등 판매",
				CompanyCount: 85, Representative: true,
				Rankings: []domain.TICSRanking{
					{BaseDate: "2026-05-16", TypeName: "시가총액", Ranking: 5, CompanyCount: 45, DisplayValue: "310조 6,978억"},
					{BaseDate: "2026-04-03", TypeName: "매출", Ranking: 8, CompanyCount: 45, DisplayValue: "20조 238억"},
					{BaseDate: "2026-04-03", TypeName: "영업이익률", Ranking: 3, CompanyCount: 45, DisplayValue: "40.7%"},
				},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteTICSIndustry(&buf, FormatTable, ind); err != nil {
		t.Fatalf("WriteTICSIndustry error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "컴퓨터와 주변기기", "85개사", "시가총액", "5", "310조 6,978억", "매출", "8", "영업이익률", "3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteTICSIndustryCSVQuotesCommaValues(t *testing.T) {
	t.Parallel()
	ind := domain.TICSIndustry{
		ProductCode: "NAS0250224006",
		Major: []domain.TICSEntry{{
			ID: 209, Title: "컴퓨터와 주변기기", CompanyCount: 85,
			Rankings: []domain.TICSRanking{
				{BaseDate: "2026-05-16", TypeName: "시가총액", Ranking: 5, CompanyCount: 45, DisplayValue: "310조 6,978억"},
			},
		}},
	}
	var buf bytes.Buffer
	if err := WriteTICSIndustry(&buf, FormatCSV, ind); err != nil {
		t.Fatalf("WriteTICSIndustry CSV error: %v", err)
	}
	// Parse the CSV back to verify field count
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) != 2 { // header + 1 data row
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if len(rows[1]) != 7 {
		t.Fatalf("expected 7 fields in data row, got %d: %v", len(rows[1]), rows[1])
	}
	if rows[1][6] != "310조 6,978억" {
		t.Fatalf("expected display_value preserved as %q, got %q", "310조 6,978억", rows[1][6])
	}
}
