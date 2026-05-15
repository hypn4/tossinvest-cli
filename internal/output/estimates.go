package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockEstimates renders the headline consensus + 3 time-series tables
// (revenue, EPS, operating-income forecasts).
func WriteStockEstimates(w io.Writer, format Format, est domain.StockEstimates) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(est)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "period", "actual_usd", "estimate_usd", "surprise_pct"}); err != nil {
			return err
		}
		for _, p := range est.Revenue.Graph {
			if err := cw.Write([]string{"revenue", p.Period, fmtFloatPtr(p.Revenue), fmtFloatPtr(p.RevenueEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		for _, p := range est.EPS.Graph {
			if err := cw.Write([]string{"eps", p.Period, fmtFloatPtr(p.EPS), fmtFloatPtr(p.EPSEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		for _, p := range est.OperatingIncome.Graph {
			if err := cw.Write([]string{"operating_income", p.Period, fmtFloatPtr(p.OperatingIncome), fmtFloatPtr(p.OperatingIncomeEst), fmtFloatPtr(p.Surprise)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — analyst estimates\n", est.ProductCode); err != nil {
			return err
		}

		fmt.Fprintln(w, "\n=== Consensus headline ===")
		hd := est.Headline
		announce := "—"
		if hd.AnnounceAt != nil {
			announce = *hd.AnnounceAt
		}
		fmt.Fprintf(w, "Announce: %s\n", announce)
		hdHeaders := []string{"FIELD", "USD", "KRW"}
		hdRows := [][]string{
			{"Revenue est", fmtUSDLargePtr(hd.RevenueEst), fmtKrwPtr(hd.RevenueEstKrw)},
			{"EPS est", fmtUSDSmallPtr(hd.EPSEst), fmtKrwPtr(hd.EPSEstKrw)},
			{"Operating-income est", fmtUSDLargePtr(hd.OperatingIncomeEst), fmtKrwPtr(hd.OperatingIncomeEstKrw)},
		}
		if err := renderTable(w, hdHeaders, hdRows); err != nil {
			return err
		}

		renderRevenueSection(w, est.Revenue)
		renderEpsSection(w, est.EPS)
		renderOpIncomeSection(w, est.OperatingIncome)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func renderRevenueSection(w io.Writer, s domain.EstimateRevenueSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== Revenue forecast (next %s) ===\n", pos)
	fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDLargePtr(s.RevenueEst), s.FluctuationRate)
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDLargePtr(p.Revenue), fmtUSDLargePtr(p.RevenueEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func renderEpsSection(w io.Writer, s domain.EstimateEpsSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== EPS forecast (next %s) ===\n", pos)
	fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDSmallPtr(s.EPSEst), s.FluctuationRate)
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDSmallPtr(p.EPS), fmtUSDSmallPtr(p.EPSEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func renderOpIncomeSection(w io.Writer, s domain.EstimateOperatingIncomeSeries) {
	pos := positionLabel(s.Position)
	fmt.Fprintf(w, "\n=== Operating-income forecast (%s) ===\n", pos)
	if s.OperatingIncomeEst == nil {
		fmt.Fprintln(w, "Next-period est: (no analyst coverage)")
	} else {
		fmt.Fprintf(w, "Next-period est: %s  fluctuation %+.2f%%\n", fmtUSDLargePtr(s.OperatingIncomeEst), s.FluctuationRate)
	}
	headers := []string{"PERIOD", "ACTUAL", "ESTIMATE", "SURPRISE"}
	rows := make([][]string, len(s.Graph))
	for i, p := range s.Graph {
		rows[i] = []string{p.Period, fmtUSDLargePtr(p.OperatingIncome), fmtUSDLargePtr(p.OperatingIncomeEst), fmtSurprisePtr(p.Surprise)}
	}
	renderTable(w, headers, rows)
}

func positionLabel(p *string) string {
	if p == nil {
		return "no analyst coverage"
	}
	return *p
}

func fmtUSDLargePtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return formatUSDLarge(*p)
}

func fmtUSDSmallPtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("$%.2f", *p)
}

func fmtKrwPtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return "₩" + formatWithCommas(int64(*p))
}

func fmtSurprisePtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%+.2f%%", *p)
}

func fmtFloatPtr(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', 2, 64)
}
