package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOrderable renders the orderable / withdrawable summary.
func WriteOrderable(w io.Writer, format Format, s domain.OrderableSummary) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(s)
	case FormatCSV:
		return fmt.Errorf("csv output is not supported for orderable summary")
	case FormatTable:
		if _, err := fmt.Fprintf(w, "Orderable KR: %s\n", formatKRW(s.OrderableKR.KRW)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Orderable US: %s (~ %s)\n", formatUSD(s.OrderableUS.USD), formatKRW(s.OrderableUS.KRW)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "\nKR overview:"); err != nil {
			return err
		}
		if err := WriteTransactionsOverview(w, FormatTable, s.KR); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "\nUS overview:"); err != nil {
			return err
		}
		return WriteTransactionsOverview(w, FormatTable, s.US)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
