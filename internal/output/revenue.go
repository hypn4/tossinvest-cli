package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteSalesComposition renders the revenue composition payload.
func WriteSalesComposition(w io.Writer, format Format, sc domain.SalesComposition) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(sc)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"business", "product", "ratio"}); err != nil {
			return err
		}
		for _, it := range sc.Items {
			if err := cw.Write([]string{it.Business, it.Product, fmt.Sprintf("%.2f", it.Ratio)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — 매출 구성 (FY%d, ending %s)\n", sc.ProductCode, sc.FiscalYear, sc.EndDate); err != nil {
			return err
		}
		headers := []string{"BUSINESS", "PRODUCT", "RATIO"}
		rows := make([][]string, len(sc.Items))
		for i, it := range sc.Items {
			rows[i] = []string{it.Business, it.Product, fmt.Sprintf("%.2f%%", it.Ratio)}
		}
		if err := renderTable(w, headers, rows); err != nil {
			return err
		}
		if sc.DataSource != "" {
			_, err := fmt.Fprintf(w, "%s\n", sc.DataSource)
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
