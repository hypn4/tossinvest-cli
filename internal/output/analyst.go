package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteAnalystSnapshot renders the analyst opinion + consensus + reports
// bundle as a sectioned table.
func WriteAnalystSnapshot(w io.Writer, format Format, snap domain.AnalystSnapshot) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(snap)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"section", "key", "value"}); err != nil {
			return err
		}
		rows := [][]string{
			{"opinion", "type", snap.Opinion.Type},
			{"opinion", "strongBuy", strconv.Itoa(snap.Opinion.StrongBuy)},
			{"opinion", "buy", strconv.Itoa(snap.Opinion.Buy)},
			{"opinion", "hold", strconv.Itoa(snap.Opinion.Hold)},
			{"opinion", "sell", strconv.Itoa(snap.Opinion.Sell)},
			{"opinion", "strongSell", strconv.Itoa(snap.Opinion.StrongSell)},
			{"consensus", "mean", fmt.Sprintf("%.2f", snap.Consensus.Mean)},
			{"consensus", "high", fmt.Sprintf("%.2f", snap.Consensus.High)},
			{"consensus", "low", fmt.Sprintf("%.2f", snap.Consensus.Low)},
		}
		for _, r := range rows {
			if err := cw.Write(r); err != nil {
				return err
			}
		}
		for _, p := range snap.Consensus.PastCloses {
			if err := cw.Write([]string{"pastClose", p.Date, fmt.Sprintf("%.2f", p.Price)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "%s — analyst snapshot\n", snap.ProductCode); err != nil {
			return err
		}
		op := snap.Opinion
		fmt.Fprintf(w, "\n=== 의견 ===\n")
		fmt.Fprintf(w, "Type: %s\n", op.Type)
		fmt.Fprintf(w, "strongBuy %d  buy %d  hold %d  sell %d  strongSell %d\n", op.StrongBuy, op.Buy, op.Hold, op.Sell, op.StrongSell)
		if op.Description != "" {
			fmt.Fprintf(w, "%s\n", op.Description)
		}

		fmt.Fprintf(w, "\n=== 컨센서스 목표가 (%s 기준) ===\n", snap.Consensus.PointDate)
		headers := []string{"FIELD", "USD", "KRW"}
		rows := [][]string{
			{"mean", formatUSD(snap.Consensus.Mean), formatWithCommas(int64(snap.Consensus.MeanKRW))},
			{"high", formatUSD(snap.Consensus.High), formatWithCommas(int64(snap.Consensus.HighKRW))},
			{"low", formatUSD(snap.Consensus.Low), formatWithCommas(int64(snap.Consensus.LowKRW))},
		}
		if err := renderTable(w, headers, rows); err != nil {
			return err
		}

		if len(snap.Consensus.PastCloses) > 0 {
			fmt.Fprintf(w, "\n=== 과거 종가 ===\n")
			headers = []string{"DATE", "PRICE (USD)", "PRICE (KRW)"}
			rows = make([][]string, len(snap.Consensus.PastCloses))
			for i, p := range snap.Consensus.PastCloses {
				rows[i] = []string{p.Date, formatUSD(p.Price), formatWithCommas(int64(p.PriceKRW))}
			}
			if err := renderTable(w, headers, rows); err != nil {
				return err
			}
		}

		fmt.Fprintf(w, "\n애널리스트 보고서: %d건\n", len(snap.Reports))
		if len(snap.Reports) > 0 {
			headers = []string{"DATE", "SOURCE", "TITLE"}
			rows = make([][]string, len(snap.Reports))
			for i, r := range snap.Reports {
				rows[i] = []string{r.Date, r.Source, r.Title}
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
