package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteSignals renders a batch of one-line signals.
func WriteSignals(w io.Writer, format Format, signals []domain.Signal) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(signals)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"product_code", "reasoning_description"}); err != nil {
			return err
		}
		for _, s := range signals {
			if err := writer.Write([]string{s.ProductCode, s.ReasoningDescription}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"CODE", "REASONING"}
		rows := make([][]string, 0, len(signals))
		for _, s := range signals {
			rows = append(rows, []string{s.ProductCode, s.ReasoningDescription})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
