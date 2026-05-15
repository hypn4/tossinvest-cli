package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockValuation renders the per-stock valuation snapshot + peer table.
func WriteStockValuation(w io.Writer, format Format, val domain.StockValuation) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(val)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"product_code", "name", "factor", "value", "period", "is_self"}); err != nil {
			return err
		}
		for _, p := range val.Peers {
			if err := cw.Write([]string{p.ProductCode, p.Name, val.Factor, fmt.Sprintf("%.2f", p.Value), p.Period, strconv.FormatBool(p.IsSelf)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — valuation snapshot\n", val.ProductCode); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "PER %.2f vs median %.2f  →  %s\n", val.PER, val.Median, val.Position); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "PBR %.2f   PSR %.2f\n", val.PBR, val.PSR); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "\n동종업계 비교 (%s 기준, %s)\n", val.Factor, val.Industry); err != nil {
			return err
		}
		headers := []string{"SYMBOL", "NAME", val.Factor, "PERIOD", ""}
		rows := make([][]string, len(val.Peers))
		for i, p := range val.Peers {
			marker := ""
			if p.IsSelf {
				marker = "← self"
			}
			rows[i] = []string{p.ProductCode, p.Name, fmt.Sprintf("%.2f", p.Value), p.Period, marker}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
