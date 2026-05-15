package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteOrderBook renders an orderbook snapshot.
func WriteOrderBook(w io.Writer, format Format, book domain.OrderBook) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(book)
	case FormatCSV:
		return writeOrderBookCSV(w, book)
	case FormatTable:
		return writeOrderBookTable(w, book)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeOrderBookCSV(w io.Writer, book domain.OrderBook) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"side", "price", "volume"}); err != nil {
		return err
	}
	for _, level := range book.Offers {
		if err := writer.Write([]string{"offer", formatFloat(level.Price), formatFloat(level.Volume)}); err != nil {
			return err
		}
	}
	for _, level := range book.Bids {
		if err := writer.Write([]string{"bid", formatFloat(level.Price), formatFloat(level.Volume)}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeOrderBookTable(w io.Writer, book domain.OrderBook) error {
	if _, err := fmt.Fprintf(
		w,
		"%s (%s)  last=%s  offers=%d  bids=%d\n",
		labelFor(book.Symbol, book.Name, book.ProductCode),
		book.Market,
		formatFloat(book.Last),
		len(book.Offers),
		len(book.Bids),
	); err != nil {
		return err
	}
	if book.SinglePrice {
		if _, err := fmt.Fprintf(
			w, "(single price) est=%s  est_volume=%s\n",
			formatFloat(book.EstimatedPrice), formatFloat(book.EstimatedVolume),
		); err != nil {
			return err
		}
	}
	headers := []string{"SIDE", "PRICE", "VOLUME"}
	rows := make([][]string, 0, len(book.Offers)+len(book.Bids))
	for i := len(book.Offers) - 1; i >= 0; i-- {
		level := book.Offers[i]
		rows = append(rows, []string{"OFFER", formatFloat(level.Price), formatFloat(level.Volume)})
	}
	for _, level := range book.Bids {
		rows = append(rows, []string{"BID", formatFloat(level.Price), formatFloat(level.Volume)})
	}
	return renderTable(w, headers, rows)
}

// WriteTicks renders a snapshot of ticks newest-first.
func WriteTicks(w io.Writer, format Format, ticks []domain.Tick) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(ticks)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"time", "code", "price", "volume", "trade_type", "cumulative_volume"}); err != nil {
			return err
		}
		for _, t := range ticks {
			if err := writer.Write([]string{
				t.Time, t.ProductCode,
				formatFloat(t.Price), formatFloat(t.Volume),
				t.TradeType, formatFloat(t.CumulativeVolume),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"TIME", "TYPE", "PRICE", "VOLUME", "CUMVOL"}
		rows := make([][]string, 0, len(ticks))
		for _, t := range ticks {
			rows = append(rows, []string{
				t.Time, t.TradeType, formatFloat(t.Price), formatFloat(t.Volume), formatFloat(t.CumulativeVolume),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteTickNDJSON writes a single tick as a single JSON object terminated by '\n'.
// Newline-delimited JSON is what consumers (jq, LLM pipelines) expect from
// `tossctl quotes ticks --follow`.
func WriteTickNDJSON(w io.Writer, tick domain.Tick) error {
	data, err := json.Marshal(tick)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func labelFor(symbol, name, productCode string) string {
	if symbol != "" && name != "" && symbol != name {
		return fmt.Sprintf("%s — %s", symbol, name)
	}
	if symbol != "" {
		return symbol
	}
	return productCode
}
