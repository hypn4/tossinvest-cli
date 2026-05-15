package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteChart writes a Chart in the requested format.
func WriteChart(w io.Writer, format Format, chart domain.Chart) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(chart)
	case FormatCSV:
		return writeChartCSV(w, chart)
	case FormatTable:
		return writeChartTable(w, chart)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeChartCSV(w io.Writer, chart domain.Chart) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"datetime", "session", "base", "open", "high", "low", "close", "volume", "amount",
	}); err != nil {
		return err
	}
	for _, c := range chart.Candles {
		if err := writer.Write([]string{
			c.DateTime,
			c.SessionType,
			formatFloat(c.Base),
			formatFloat(c.Open),
			formatFloat(c.High),
			formatFloat(c.Low),
			formatFloat(c.Close),
			formatFloat(c.Volume),
			formatFloat(c.Amount),
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeChartTable(w io.Writer, chart domain.Chart) error {
	if _, err := fmt.Fprintf(
		w,
		"%s (%s)  %s:%d  session=%s  candles=%d\n",
		chartLabel(chart),
		chart.Market,
		chart.Unit,
		chart.Step,
		nonEmptyOr(chart.Session, "all"),
		len(chart.Candles),
	); err != nil {
		return err
	}
	if len(chart.Candles) == 0 {
		_, err := fmt.Fprintln(w, "(no candles)")
		return err
	}
	headers := []string{"DT", "SESSION", "OPEN", "HIGH", "LOW", "CLOSE", "VOLUME"}
	rows := make([][]string, 0, len(chart.Candles))
	for _, c := range chart.Candles {
		rows = append(rows, []string{
			c.DateTime,
			c.SessionType,
			formatFloat(c.Open),
			formatFloat(c.High),
			formatFloat(c.Low),
			formatFloat(c.Close),
			formatVolume(c.Volume),
		})
	}
	if err := renderTable(w, headers, rows); err != nil {
		return err
	}
	if chart.NextDateTime != "" {
		if _, err := fmt.Fprintf(w, "next: %s\n", chart.NextDateTime); err != nil {
			return err
		}
	}
	if chart.ExchangeRate > 0 {
		if _, err := fmt.Fprintf(w, "fx (USD→KRW): %s\n", formatFloat(chart.ExchangeRate)); err != nil {
			return err
		}
	}
	return nil
}

func chartLabel(chart domain.Chart) string {
	if chart.Symbol != "" && chart.Name != "" && chart.Symbol != chart.Name {
		return fmt.Sprintf("%s — %s", chart.Symbol, chart.Name)
	}
	if chart.Symbol != "" {
		return chart.Symbol
	}
	return chart.ProductCode
}

func nonEmptyOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func formatVolume(v float64) string {
	if v == 0 {
		return "-"
	}
	if v >= 1e9 {
		return strconv.FormatFloat(v/1e9, 'f', 2, 64) + "B"
	}
	if v >= 1e6 {
		return strconv.FormatFloat(v/1e6, 'f', 2, 64) + "M"
	}
	if v >= 1e3 {
		return strconv.FormatFloat(v/1e3, 'f', 2, 64) + "K"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
