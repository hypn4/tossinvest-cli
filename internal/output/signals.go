package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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

// WriteSignalDetail renders a rich SignalDetail.
func WriteSignalDetail(w io.Writer, format Format, detail domain.SignalDetail) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(detail)
	case FormatCSV:
		return fmt.Errorf("csv output is not supported for signal detail")
	case FormatTable:
		direction := "▶"
		switch {
		case detail.SignalDirection > 0:
			direction = "▲ 매수"
		case detail.SignalDirection < 0:
			direction = "▼ 매도"
		}
		if _, err := fmt.Fprintf(w, "%s  %s  %s\n", direction, detail.AssetName, detail.Description); err != nil {
			return err
		}
		if len(detail.Keywords) > 0 {
			if _, err := fmt.Fprintf(w, "키워드: %s\n", strings.Join(detail.Keywords, ", ")); err != nil {
				return err
			}
		}
		if len(detail.DescriptionItems) > 0 {
			if _, err := fmt.Fprintln(w, "\n근거:"); err != nil {
				return err
			}
			for _, item := range detail.DescriptionItems {
				if _, err := fmt.Fprintf(w, "  • %s\n", item); err != nil {
					return err
				}
			}
		}
		if len(detail.News) > 0 {
			if _, err := fmt.Fprintln(w, "\n뉴스:"); err != nil {
				return err
			}
			for _, n := range detail.News {
				if _, err := fmt.Fprintf(w, "  [%s] %s\n", n.AgencyName, n.Title); err != nil {
					return err
				}
			}
		}
		if len(detail.Related) > 0 {
			if _, err := fmt.Fprintln(w, "\n관련 종목:"); err != nil {
				return err
			}
			for _, r := range detail.Related {
				if _, err := fmt.Fprintf(w, "  • %s (%s)\n", r.AssetName, r.Relation); err != nil {
					return err
				}
				for _, line := range r.Description {
					if _, err := fmt.Fprintf(w, "      %s\n", line); err != nil {
						return err
					}
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
