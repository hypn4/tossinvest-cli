package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

const dateColumnWidth = 10 // YYYY-MM-DD

// WriteStockNews renders a news feed. Table mode shows date / source / title
// (truncated to 80 runes per row). CSV emits the full structure (one row per
// item). JSON dumps the full slice.
func WriteStockNews(w io.Writer, format Format, items []domain.NewsItem) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"id", "created_at", "title", "summary", "source_code", "source_name"}); err != nil {
			return err
		}
		for _, n := range items {
			if err := cw.Write([]string{n.ID, n.CreatedAt, n.Title, n.Summary, n.Source.Code, n.Source.Name}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if len(items) == 0 {
			_, err := fmt.Fprintln(w, "No news found.")
			return err
		}
		headers := []string{"DATE", "SOURCE", "TITLE"}
		rows := make([][]string, len(items))
		for i, n := range items {
			date := n.CreatedAt
			if len(date) > dateColumnWidth {
				date = date[:dateColumnWidth]
			}
			rows[i] = []string{date, n.Source.Name, truncateName(n.Title, 80)}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteStockFilings renders KR filings (DART + KIND). EARNINGS form items get
// a follow-on row showing the earning-call title and status.
func WriteStockFilings(w io.Writer, format Format, items []domain.FilingItem) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"id", "created_at", "form", "title", "summary", "earning_call_status", "earning_call_title"}); err != nil {
			return err
		}
		for _, f := range items {
			callStatus, callTitle := "", ""
			if f.EarningCall != nil {
				callStatus = f.EarningCall.Status
				callTitle = f.EarningCall.Title
			}
			if err := cw.Write([]string{f.ID, f.CreatedAt, f.Form, f.Title, f.Summary, callStatus, callTitle}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if len(items) == 0 {
			_, err := fmt.Fprintln(w, "No filings found. (US stocks typically have no Korean filings; check news.)")
			return err
		}
		headers := []string{"DATE", "FORM", "TITLE"}
		rows := make([][]string, 0, len(items)*2)
		for _, f := range items {
			date := f.CreatedAt
			if len(date) > dateColumnWidth {
				date = date[:dateColumnWidth]
			}
			rows = append(rows, []string{date, f.Form, truncateName(f.Title, 80)})
			if f.EarningCall != nil {
				rows = append(rows, []string{"", "↳ 어닝콜", fmt.Sprintf("%s (%s)", f.EarningCall.Title, f.EarningCall.Status)})
			}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
