package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteFills(w io.Writer, format Format, fills []domain.CompactExecution) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(fills)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"product_code", "trade_type", "bucket_start",
			"quantity", "execution_avg_local_price", "execution_avg_krw_price",
			"execution_total_local", "execution_total_krw",
		}); err != nil {
			return err
		}
		for _, f := range fills {
			if err := writer.Write([]string{
				f.ProductCode,
				f.TradeType,
				f.BucketStart.Format("2006-01-02T15:04:05Z07:00"),
				formatFloat(f.Quantity),
				formatFloat(f.ExecutionAvgLocalPrice),
				formatFloat(f.ExecutionAvgKRWPrice),
				formatFloat(f.ExecutionTotalLocal),
				formatFloat(f.ExecutionTotalKRWAmount),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"BUCKET", "SIDE", "QTY", "AVG (local)", "AVG (KRW)", "TOTAL (local)"}
		rows := make([][]string, 0, len(fills))
		for _, f := range fills {
			rows = append(rows, []string{
				f.BucketStart.Format("2006-01-02 15:04"),
				f.TradeType,
				formatQty(f.Quantity),
				formatFloat(f.ExecutionAvgLocalPrice),
				formatKRW(f.ExecutionAvgKRWPrice),
				formatFloat(f.ExecutionTotalLocal),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
