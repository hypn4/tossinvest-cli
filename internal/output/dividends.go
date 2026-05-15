package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockDividends renders the dividends snapshot. `allHistory` controls
// whether the full historical payout list is rendered in table mode (it's
// always emitted in JSON; CSV honors the flag).
func WriteStockDividends(w io.Writer, format Format, div domain.StockDividends, allHistory bool) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(div)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "ex_date", "payment_date", "currency", "cash", "cash_krw", "yield_ratio_pct", "ttm_yield_ratio_pct"}); err != nil {
			return err
		}
		// Summary card as a single row
		curr := div.Summary.Currency
		if curr == "" {
			curr = "-"
		}
		if err := cw.Write([]string{
			"summary", "", "", curr,
			strconv.FormatFloat(div.Summary.TTMDps, 'f', 4, 64),
			ptrFloatString(div.Summary.TTMDpsKrw),
			fmt.Sprintf("%.4f", div.Summary.TTMDividendYieldRatio*100),
			"",
		}); err != nil {
			return err
		}
		for _, p := range div.RecentYears.Payouts {
			if err := cw.Write([]string{
				"recent", p.ExDate, p.PaymentDate, p.Currency,
				strconv.FormatFloat(p.Cash, 'f', 4, 64),
				strconv.FormatFloat(p.CashKrw, 'f', 2, 64),
				fmt.Sprintf("%.4f", p.YieldRatio*100),
				fmt.Sprintf("%.4f", p.TTMYieldRatio*100),
			}); err != nil {
				return err
			}
		}
		if allHistory {
			for _, p := range div.FullHistory {
				if err := cw.Write([]string{
					"history", p.ExDate, p.PaymentDate, p.Currency,
					strconv.FormatFloat(p.Cash, 'f', 4, 64),
					strconv.FormatFloat(p.CashKrw, 'f', 2, 64),
					fmt.Sprintf("%.4f", p.YieldRatio*100),
					fmt.Sprintf("%.4f", p.TTMYieldRatio*100),
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — dividends\n", div.ProductCode); err != nil {
			return err
		}

		// Empty case: short-circuit
		if div.Summary.DividendCount == 0 && len(div.RecentYears.Payouts) == 0 {
			_, err := fmt.Fprintln(w, "  No dividends recorded for this stock.")
			return err
		}

		// Summary card
		fmt.Fprintf(w, "\n=== Summary (TTM, %s) ===\n", div.Summary.Currency)
		sumHeaders := []string{"FIELD", "VALUE"}
		sumRows := [][]string{
			{"TTM yield", fmt.Sprintf("%.2f%%", div.Summary.TTMDividendYieldRatio*100)},
			{"TTM dps", fmt.Sprintf("$%.2f", div.Summary.TTMDps)},
			{"TTM dividend count", strconv.Itoa(div.Summary.TTMDividendTotalCount)},
			{"Dividend cadence", fmt.Sprintf("%v", div.Summary.DividendMonths)},
			{"Growth", growthPct(div.Summary.DividendGrowthRatio)},
		}
		if err := renderTable(w, sumHeaders, sumRows); err != nil {
			return err
		}

		// Recent years
		fmt.Fprintf(w, "\n=== Recent payouts (%s, since %s) ===\n", div.RecentYears.RangeLabel, div.RecentYears.StartDate)
		if len(div.RecentYears.Payouts) == 0 {
			fmt.Fprintln(w, "  (no payouts in selected range)")
		} else {
			recHeaders := []string{"EX DATE", "PAYMENT", "CASH (USD)", "CASH (KRW)", "YIELD", "TTM YIELD"}
			recRows := make([][]string, len(div.RecentYears.Payouts))
			for i, p := range div.RecentYears.Payouts {
				recRows[i] = []string{
					p.ExDate, p.PaymentDate,
					fmt.Sprintf("$%.2f", p.Cash),
					formatWithCommas(int64(p.CashKrw)),
					fmt.Sprintf("%.2f%%", p.YieldRatio*100),
					fmt.Sprintf("%.2f%%", p.TTMYieldRatio*100),
				}
			}
			if err := renderTable(w, recHeaders, recRows); err != nil {
				return err
			}
			fmt.Fprintf(w, "Total: $%.2f  (₩%s)\n", div.RecentYears.TotalCash, formatWithCommas(int64(div.RecentYears.TotalCashKrw)))
		}

		// Full history (opt-in)
		if allHistory && len(div.FullHistory) > 0 {
			fmt.Fprintf(w, "\n=== 전체 히스토리 (%d 건) ===\n", len(div.FullHistory))
			hHeaders := []string{"EX DATE", "PAYMENT", "CASH (USD)", "YIELD"}
			hRows := make([][]string, len(div.FullHistory))
			for i, p := range div.FullHistory {
				hRows[i] = []string{
					p.ExDate, p.PaymentDate,
					fmt.Sprintf("$%.2f", p.Cash),
					fmt.Sprintf("%.2f%%", p.YieldRatio*100),
				}
			}
			if err := renderTable(w, hHeaders, hRows); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func growthPct(p *float64) string {
	if p == nil {
		return "—"
	}
	v := *p * 100
	sign := "+"
	if v < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.2f%%", sign, v)
}

func ptrFloatString(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', 2, 64)
}
