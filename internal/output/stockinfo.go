package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteStockInfoDetail renders the deep-tab payload. Table mode emits one row
// per section (type + byte count); JSON mode emits the full structure
// including raw section bodies; CSV mode emits (section_type, data_bytes) pairs.
func WriteStockInfoDetail(w io.Writer, format Format, detail domain.StockInfoDetail) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(detail)
	case FormatCSV:
		if _, err := fmt.Fprintln(w, "section_type,data_bytes"); err != nil {
			return err
		}
		for _, s := range detail.Sections {
			if _, err := fmt.Fprintf(w, "%s,%d\n", s.Type, len(s.Data)); err != nil {
				return err
			}
		}
		return nil
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s  sections=%d  fetched=%s\n",
			detail.ProductCode, len(detail.Sections), detail.FetchedAt.Format("2006-01-02 15:04:05Z07:00")); err != nil {
			return err
		}
		headers := []string{"TYPE", "BYTES"}
		rows := make([][]string, 0, len(detail.Sections))
		for _, s := range detail.Sections {
			rows = append(rows, []string{s.Type, fmt.Sprintf("%d", len(s.Data))})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
