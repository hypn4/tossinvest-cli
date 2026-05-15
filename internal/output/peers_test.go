package output

import (
	"bytes"
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
