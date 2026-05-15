package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteQuote(w io.Writer, format Format, quote domain.Quote) error {
	switch format {
	case FormatTable:
		return writeQuoteTable(w, quote)
	case FormatJSON:
		return writeQuoteJSON(w, quote)
	case FormatCSV:
		return writeQuoteCSV(w, quote)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeQuoteJSON(w io.Writer, quote domain.Quote) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(quote)
}

func writeQuoteCSV(w io.Writer, quote domain.Quote) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"product_code",
		"symbol",
		"name",
		"market_code",
		"market",
		"currency",
		"reference_price",
		"last",
		"change",
		"change_rate",
		"volume",
		"status",
		"badge_count",
		"notice_count",
		"fetched_at",
	}); err != nil {
		return err
	}

	if err := writer.Write([]string{
		quote.ProductCode,
		quote.Symbol,
		quote.Name,
		quote.MarketCode,
		quote.Market,
		quote.Currency,
		formatFloat(quote.ReferencePrice),
		formatFloat(quote.Last),
		formatFloat(quote.Change),
		formatFloat(quote.ChangeRate),
		formatFloat(quote.Volume),
		quote.Status,
		strconv.Itoa(quote.BadgeCount),
		strconv.Itoa(quote.NoticeCount),
		quote.FetchedAt.Format("2006-01-02T15:04:05Z07:00"),
	}); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func writeQuoteTable(w io.Writer, quote domain.Quote) error {
	if _, err := fmt.Fprintf(
		w,
		"Product Code: %s\nSymbol: %s\nName: %s\nMarket: %s (%s)\nCurrency: %s\nReference Price: %s\nLast: %s\nChange: %s\nChange Rate: %.2f%%\nVolume: %s\nStatus: %s\nBadges: %d\nNotices: %d\nFetched At: %s\n",
		quote.ProductCode,
		quote.Symbol,
		quote.Name,
		quote.Market,
		quote.MarketCode,
		quote.Currency,
		formatFloat(quote.ReferencePrice),
		formatFloat(quote.Last),
		formatFloat(quote.Change),
		quote.ChangeRate*100,
		formatFloat(quote.Volume),
		quote.Status,
		quote.BadgeCount,
		quote.NoticeCount,
		quote.FetchedAt.Format("2006-01-02 15:04:05Z07:00"),
	); err != nil {
		return err
	}
	if quote.Open != 0 || quote.High != 0 || quote.Low != 0 {
		if _, err := fmt.Fprintf(w, "Open/High/Low: %s / %s / %s\n",
			formatFloat(quote.Open), formatFloat(quote.High), formatFloat(quote.Low)); err != nil {
			return err
		}
	}
	if quote.High52W != 0 || quote.Low52W != 0 {
		if _, err := fmt.Fprintf(w, "52W High/Low: %s / %s\n",
			formatFloat(quote.High52W), formatFloat(quote.Low52W)); err != nil {
			return err
		}
	}
	if quote.UpperLimit != 0 || quote.LowerLimit != 0 {
		if _, err := fmt.Fprintf(w, "Upper/Lower Limit: %s / %s\n",
			formatFloat(quote.UpperLimit), formatFloat(quote.LowerLimit)); err != nil {
			return err
		}
	}
	if quote.MarketCap != 0 {
		if _, err := fmt.Fprintf(w, "Market Cap: %s\n", formatFloat(quote.MarketCap)); err != nil {
			return err
		}
	}
	if quote.TradingStrength != 0 {
		if _, err := fmt.Fprintf(w, "Trading Strength: %.2f\n", quote.TradingStrength); err != nil {
			return err
		}
	}
	if quote.PreDayVolume != 0 {
		if _, err := fmt.Fprintf(w, "Prev Day Volume: %s\n", formatFloat(quote.PreDayVolume)); err != nil {
			return err
		}
	}
	if quote.AfterMarketClose != 0 {
		if _, err := fmt.Fprintf(w, "After-market Close: %s\n", formatFloat(quote.AfterMarketClose)); err != nil {
			return err
		}
	}
	if quote.GrossExpenseRatio != 0 || quote.DividendYieldRate != 0 {
		if _, err := fmt.Fprintf(w, "ETF Expense / Yield: %.4f%% / %.4f%%\n",
			quote.GrossExpenseRatio*100, quote.DividendYieldRate*100); err != nil {
			return err
		}
	}
	if quote.TradingAmountRank != 0 {
		if _, err := fmt.Fprintf(w, "Trading Amount Rank: %d\n", quote.TradingAmountRank); err != nil {
			return err
		}
	}
	return nil
}

func WriteQuotes(w io.Writer, format Format, quotes []domain.Quote) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(quotes)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"symbol", "name", "market", "currency", "last", "change", "change_rate", "volume",
		}); err != nil {
			return err
		}
		for _, q := range quotes {
			if err := writer.Write([]string{
				q.Symbol, q.Name, q.Market, q.Currency,
				formatFloat(q.Last), formatFloat(q.Change), formatFloat(q.ChangeRate), formatFloat(q.Volume),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		headers := []string{"종목", "이름", "현재가", "변동", "변동률"}
		var rows [][]string
		for _, q := range quotes {
			changeStr := formatKRW(q.Change)
			if q.Change > 0 {
				changeStr = "+" + changeStr
			}
			rows = append(rows, []string{
				q.Symbol,
				q.Name,
				formatKRW(q.Last),
				changeStr,
				formatPct(q.ChangeRate),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// WriteQuoteNDJSON writes a single quote snapshot as one JSON object terminated
// by '\n'. Used by `tossctl quote get --follow`.
func WriteQuoteNDJSON(w io.Writer, quote domain.Quote) error {
	data, err := json.Marshal(quote)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
