package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockRatios renders the factor time series — typically 3 line items
// (two components + the derived ratio) across N periods. Rows are line items,
// columns are periods. AMOUNT-unit values use comma-separated millions;
// PERCENT-unit values use %.2f%%.
func WriteStockRatios(w io.Writer, format Format, ra domain.StockRatios) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ra)

	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"code", "name", "unit", "period", "value", "value_krw"}); err != nil {
			return err
		}
		for _, it := range ra.Items {
			for _, v := range it.Values {
				if err := cw.Write([]string{
					it.Code, it.Name, it.Unit, v.Period,
					strconv.FormatFloat(v.Value, 'f', 4, 64),
					strconv.FormatFloat(v.ValueKrw, 'f', 2, 64),
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()

	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — %s (%s, %s)\n", ra.ProductCode, ra.Factor.DisplayName, ra.Period, ra.RangeLabel); err != nil {
			return err
		}
		if len(ra.Items) == 0 {
			fmt.Fprintln(w, "  (no rows returned)")
			return nil
		}

		// Periods come from the first item; the API always returns the same
		// period grid across items.
		periodCols := make([]string, len(ra.Items[0].Values))
		for i, v := range ra.Items[0].Values {
			periodCols[i] = v.Period
		}
		headers := append([]string{"ITEM"}, periodCols...)

		rows := make([][]string, len(ra.Items))
		for i, it := range ra.Items {
			row := make([]string, 0, 1+len(periodCols))
			row = append(row, it.Name)
			for _, v := range it.Values {
				row = append(row, formatRatioValue(v.Value, it.Unit))
			}
			rows[i] = row
		}

		if err := renderTable(w, headers, rows); err != nil {
			return err
		}
		fmt.Fprintln(w, "(AMOUNT values are raw units; PERCENT values are formatted as percentages)")
		return nil

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatRatioValue(v float64, unit string) string {
	switch unit {
	case "PERCENT":
		return fmt.Sprintf("%.2f%%", v)
	case "AMOUNT":
		return formatWithCommas(int64(math.Round(v)))
	default:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
}
