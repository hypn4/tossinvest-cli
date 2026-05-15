package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockFinancialsTable(t *testing.T) {
	t.Parallel()
	fin := domain.StockFinancials{
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
