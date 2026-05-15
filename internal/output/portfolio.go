package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WritePositions(w io.Writer, format Format, positions []domain.Position) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(positions)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"product_code", "symbol", "name", "market_type", "market_code", "quantity", "average_price", "current_price", "market_value", "unrealized_pnl", "profit_rate", "daily_profit_loss", "daily_profit_rate", "average_price_usd", "current_price_usd", "market_value_usd", "unrealized_pnl_usd", "profit_rate_usd", "daily_profit_loss_usd", "daily_profit_rate_usd"}); err != nil {
			return err
		}
		for _, position := range positions {
			if err := writer.Write([]string{
				position.ProductCode,
				position.Symbol,
				position.Name,
				position.MarketType,
				position.MarketCode,
				formatFloat(position.Quantity),
				formatFloat(position.AveragePrice),
				formatFloat(position.CurrentPrice),
				formatFloat(position.MarketValue),
				formatFloat(position.UnrealizedPnL),
				formatFloat(position.ProfitRate),
				formatFloat(position.DailyProfitLoss),
				formatFloat(position.DailyProfitRate),
				formatFloat(position.AveragePriceUSD),
				formatFloat(position.CurrentPriceUSD),
				formatFloat(position.MarketValueUSD),
				formatFloat(position.UnrealizedPnLUSD),
				formatFloat(position.ProfitRateUSD),
				formatFloat(position.DailyProfitLossUSD),
				formatFloat(position.DailyProfitRateUSD),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		return writePositionsTable(w, positions)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func WriteAllocation(w io.Writer, format Format, markets map[string]domain.AccountMarketSummary) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(markets)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"market", "total_asset_amount", "principal_amount", "evaluated_profit_amount", "profit_rate"}); err != nil {
			return err
		}
		keys := sortedMarketKeys(markets)
		for _, key := range keys {
			market := markets[key]
			if err := writer.Write([]string{
				key,
				formatFloat(market.TotalAssetAmount),
				formatFloat(market.PrincipalAmount),
				formatFloat(market.EvaluatedProfitAmount),
				formatFloat(market.ProfitRate),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		keys := sortedMarketKeys(markets)
		for _, key := range keys {
			market := markets[key]
			if _, err := fmt.Fprintf(
				w,
				"- %s: total=%s principal=%s profit=%s rate=%.2f%%\n",
				key,
				formatFloat(market.TotalAssetAmount),
				formatFloat(market.PrincipalAmount),
				formatFloat(market.EvaluatedProfitAmount),
				market.ProfitRate*100,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writePositionsTable(w io.Writer, positions []domain.Position) error {
	hasUS := false
	for _, p := range positions {
		if p.MarketType == "US_STOCK" {
			hasUS = true
			break
		}
	}

	headers := []string{"종목", "수량", "매입가", "현재가", "평가금", "손익", "수익률", "일간손익", "일간률"}
	if hasUS {
		headers = append(headers, "USD 손익", "USD 률")
	}

	var rows [][]string
	for _, p := range positions {
		label := fmt.Sprintf("%s (%s)", truncateName(p.Name, 16), p.Symbol)
		row := []string{
			label,
			formatQty(p.Quantity),
			formatKRW(p.AveragePrice),
			formatKRW(p.CurrentPrice),
			formatKRW(p.MarketValue),
			formatKRW(p.UnrealizedPnL),
			formatPct(p.ProfitRate),
			formatKRW(p.DailyProfitLoss),
			formatPct(p.DailyProfitRate),
		}
		if hasUS {
			if p.MarketType == "US_STOCK" {
				row = append(row, formatUSD(p.UnrealizedPnLUSD), formatPct(p.ProfitRateUSD))
			} else {
				row = append(row, "", "")
			}
		}
		rows = append(rows, row)
	}

	if err := renderTable(w, headers, rows); err != nil {
		return err
	}

	// Footer notes for fees and notices
	for _, p := range positions {
		if p.EstimatedCommission == 0 && p.EstimatedTax == 0 && !p.Delisting && !p.NoticeSplitMerge && !p.NoticeEarningsAnnouncement {
			continue
		}
		notes := []string{}
		if p.EstimatedCommission != 0 {
			notes = append(notes, fmt.Sprintf("수수료 추정 %s (%.3f%%)", formatKRW(p.EstimatedCommission), p.CommissionRate*100))
		}
		if p.EstimatedTax != 0 {
			notes = append(notes, fmt.Sprintf("세금 추정 %s (%.3f%%)", formatKRW(p.EstimatedTax), p.TaxRate*100))
		}
		if p.Delisting {
			notes = append(notes, "상장폐지 예정")
		}
		if p.NoticeSplitMerge {
			notes = append(notes, "액면분할/합병 공지")
		}
		if p.NoticeEarningsAnnouncement {
			notes = append(notes, "실적 발표 예정")
		}
		fmt.Fprintf(w, "  · %s: %s\n", p.Symbol, strings.Join(notes, " · "))
	}
	return nil
}

func sortedMarketKeys(markets map[string]domain.AccountMarketSummary) []string {
	keys := make([]string, 0, len(markets))
	for key := range markets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
