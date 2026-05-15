package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteTICSIndustry renders the TICS industry taxonomy + peer rankings.
func WriteTICSIndustry(w io.Writer, format Format, ind domain.TICSIndustry) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(ind)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "industry_id,industry,base_date,metric,ranking,company_count,display_value"); err != nil {
			return err
		}
		for _, e := range ind.Major {
			for _, r := range e.Rankings {
				if _, err := fmt.Fprintf(w, "%d,%s,%s,%s,%d,%d,%s\n", e.ID, e.Title, r.BaseDate, r.TypeName, r.Ranking, r.CompanyCount, r.DisplayValue); err != nil {
					return err
				}
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — TICS industry & peer ranking\n", ind.ProductCode); err != nil {
			return err
		}
		for _, e := range ind.Major {
			if _, err := fmt.Fprintf(w, "\n%s (id %d, %d개사)\n", e.Title, e.ID, e.CompanyCount); err != nil {
				return err
			}
			if e.Description != "" {
				if _, err := fmt.Fprintf(w, "  \"%s\"\n", e.Description); err != nil {
					return err
				}
			}
			headers := []string{"METRIC", "RANK", "OUT OF", "VALUE", "AS OF"}
			rows := make([][]string, len(e.Rankings))
			for i, r := range e.Rankings {
				rows[i] = []string{r.TypeName, fmt.Sprintf("%d", r.Ranking), fmt.Sprintf("%d", r.CompanyCount), r.DisplayValue, r.BaseDate}
			}
			if err := renderTable(w, headers, rows); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
