package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockFinancials renders stability + revenue/net-profit series + operating
// income series in a sectioned table; JSON dumps full struct; CSV emits the
// time series.
func WriteStockFinancials(w io.Writer, format Format, fin domain.StockFinancials) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(fin)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "period", "metric", "value_usd", "value_krw", "ratio_pct"}); err != nil {
			return err
		}
		for _, p := range fin.Revenue.Graph {
			if err := cw.Write([]string{
				"revenue", p.Period, "revenue",
				fmt.Sprintf("%.0f", p.Revenue), fmt.Sprintf("%.0f", p.RevenueKrw), "",
			}); err != nil {
				return err
			}
			if err := cw.Write([]string{
				"revenue", p.Period, "net_profit",
				fmt.Sprintf("%.0f", p.NetProfit), fmt.Sprintf("%.0f", p.NetProfitKrw),
				fmt.Sprintf("%.2f", p.NetProfitRatio),
			}); err != nil {
				return err
			}
		}
		for _, p := range fin.OperatingIncome.Graph {
			if err := cw.Write([]string{
				"operating_income", p.Period, "operating_income",
				fmt.Sprintf("%.0f", p.OperatingIncome), fmt.Sprintf("%.0f", p.OperatingIncomeKrw),
				fmt.Sprintf("%.2f", p.OperatingIncomeRatio),
			}); err != nil {
				return err
			}
		}
		if err := cw.Write([]string{
			"stability", "", "liability_ratio",
			"", "", strconv.FormatFloat(fin.Stability.LiabilityRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		if err := cw.Write([]string{
			"stability", "", "current_ratio",
			"", "", strconv.FormatFloat(fin.Stability.CurrentRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		if err := cw.Write([]string{
			"stability", "", "interest_coverage_ratio",
			"", "", strconv.FormatFloat(fin.Stability.InterestCoverageRatio, 'f', 2, 64),
		}); err != nil {
			return err
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — financials\n", fin.ProductCode); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 안정성 (vs 업종 중앙값 %.2f%%) ===\n", fin.Stability.IndustryMedian)
		stHeaders := []string{"METRIC", "VALUE", "POSITION"}
		stRows := [][]string{
			{"부채비율", fmt.Sprintf("%.2f%%", fin.Stability.LiabilityRatio), fin.Stability.Position},
			{"유동비율", fmt.Sprintf("%.2f%%", fin.Stability.CurrentRatio), ""},
			{"이자보상비율", fmt.Sprintf("%.2f%%", fin.Stability.InterestCoverageRatio), ""},
		}
		if err := renderTable(w, stHeaders, stRows); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 매출 & 순이익 — 최근 %d Q%d ===\n", fin.Revenue.RecentFiscalYear, fin.Revenue.RecentFiscalQuarter)
		fmt.Fprintf(w, "Recent net profit: %s (%s KRW)  Fluctuation: %+.2f%%  Position: %s\n",
			formatUSDLarge(fin.Revenue.RecentNetProfit),
			formatWithCommas(int64(fin.Revenue.RecentNetProfitKrw)),
			fin.Revenue.FluctuationRate,
			fin.Revenue.Position,
		)
		revHeaders := []string{"PERIOD", "REVENUE (USD)", "NET PROFIT (USD)", "NET PROFIT MARGIN"}
		revRows := make([][]string, len(fin.Revenue.Graph))
		for i, p := range fin.Revenue.Graph {
			revRows[i] = []string{
				p.Period,
				formatUSDLarge(p.Revenue),
				formatUSDLarge(p.NetProfit),
				fmt.Sprintf("%.2f%%", p.NetProfitRatio),
			}
		}
		if err := renderTable(w, revHeaders, revRows); err != nil {
			return err
		}

		fmt.Fprintf(w, "\n=== 영업이익 — 최근 %d Q%d ===\n", fin.OperatingIncome.RecentFiscalYear, fin.OperatingIncome.RecentFiscalQuarter)
		fmt.Fprintf(w, "Recent operating income: %s (%s KRW)  Fluctuation: %+.2f%%  Position: %s\n",
			formatUSDLarge(fin.OperatingIncome.RecentOperatingIncome),
			formatWithCommas(int64(fin.OperatingIncome.RecentOperatingIncomeKrw)),
			fin.OperatingIncome.FluctuationRate,
			fin.OperatingIncome.Position,
		)
		opHeaders := []string{"PERIOD", "OPERATING INCOME (USD)", "OPERATING MARGIN"}
		opRows := make([][]string, len(fin.OperatingIncome.Graph))
		for i, p := range fin.OperatingIncome.Graph {
			opRows[i] = []string{
				p.Period,
				formatUSDLarge(p.OperatingIncome),
				fmt.Sprintf("%.2f%%", p.OperatingIncomeRatio),
			}
		}
		return renderTable(w, opHeaders, opRows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
