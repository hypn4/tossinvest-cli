package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockStatements renders the pivoted financial-statement-records table:
// rows = line items (in document order, children indented under parents),
// columns = periods (oldest-first). Values display in millions when the unit
// is USD, raw otherwise.
func WriteStockStatements(w io.Writer, format Format, st domain.StockStatements) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(st)

	case FormatCSV:
		// Pivot: one row per (item, period); columns = item, parent_item, name_kor, name_eng, unit, period, value
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"item", "parent_item", "name_kor", "name_eng", "unit", "period", "value"}); err != nil {
			return err
		}
		for _, p := range st.Periods {
			for _, it := range p.Items {
				val := ""
				if it.Value != nil {
					val = strconv.FormatFloat(*it.Value, 'f', 2, 64)
				}
				if err := cw.Write([]string{
					it.Item, it.ParentItem, it.NameKor, it.NameEng, it.Unit, p.Period, val,
				}); err != nil {
					return err
				}
			}
		}
		cw.Flush()
		return cw.Error()

	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — %s (%s)\n", st.ProductCode, st.Factor.DisplayName, st.Period); err != nil {
			return err
		}

		// Collect line items in document order from the first non-empty period.
		var refItems []domain.StatementLineItem
		for _, p := range st.Periods {
			if len(p.Items) > 0 {
				refItems = p.Items
				break
			}
		}
		if len(refItems) == 0 {
			fmt.Fprintln(w, "  (no rows returned)")
			return nil
		}

		// Build a value lookup: map[period][item] -> value
		valuesByPeriod := make(map[string]map[string]*float64, len(st.Periods))
		for _, p := range st.Periods {
			m := make(map[string]*float64, len(p.Items))
			for _, it := range p.Items {
				m[it.Item] = it.Value
			}
			valuesByPeriod[p.Period] = m
		}

		// Header row
		periodCols := make([]string, len(st.Periods))
		for i, p := range st.Periods {
			periodCols[i] = p.Period
		}
		headers := append([]string{"ITEM", "NAME"}, periodCols...)

		// Data rows
		rows := make([][]string, len(refItems))
		for i, ref := range refItems {
			label := ref.Item
			if ref.ParentItem != "" {
				label = "  " + ref.Item
			}
			name := ref.NameKor
			if name == "" {
				name = ref.NameEng
			}
			row := make([]string, 0, 2+len(periodCols))
			row = append(row, label, name)
			for _, p := range st.Periods {
				v := valuesByPeriod[p.Period][ref.Item]
				row = append(row, formatStatementValue(v, ref.Unit))
			}
			rows[i] = row
		}

		if err := renderTable(w, headers, rows); err != nil {
			return err
		}

		// Footer unit hint
		unit := ""
		if len(refItems) > 0 && refItems[0].Unit != "" {
			unit = refItems[0].Unit
		}
		if unit != "" {
			fmt.Fprintf(w, "(values in millions of %s; — = not reported)\n", unit)
		}
		return nil

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatStatementValue(v *float64, unit string) string {
	if v == nil {
		return "—"
	}
	// Values come in as raw units (e.g. 1665.0 for $1.665B). Toss's web UI
	// displays these as-is with comma separators — they are already in
	// millions per the unitType (USD=USD millions).
	return formatWithCommas(int64(*v))
}
