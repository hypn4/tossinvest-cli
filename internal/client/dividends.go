package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type dividendYieldCardEnvelope struct {
	Result struct {
		DividendCount         int      `json:"dividendCount"`
		DividendMonths        []int    `json:"dividendMonths"`
		DividendCash          float64  `json:"dividendCash"`
		DividendCashKrw       *float64 `json:"dividendCashKrw"`
		DividendYieldRatio    float64  `json:"dividendYieldRatio"`
		TTMDividendYieldRatio float64  `json:"ttmDividendYieldRatio"`
		TTMDividendMonths     []string `json:"ttmDividendMonths"`
		TTMDps                float64  `json:"ttmDps"`
		TTMDpsKrw             *float64 `json:"ttmDpsKrw"`
		TTMDividendTotalCount int      `json:"ttmDividendTotalCount"`
		DividendGrowthRatio   *float64 `json:"dividendGrowthRatio"`
		Currency              string   `json:"currency"`
	} `json:"result"`
}

type dividendYearsEnvelope struct {
	Result struct {
		StartDate     string `json:"startDate"`
		SelectedRange struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedRange"`
		Histories    []dividendPayoutWire `json:"histories"`
		TotalCash    float64              `json:"totalCash"`
		TotalCashKrw float64              `json:"totalCashKrw"`
	} `json:"result"`
}

type dividendHistoryEnvelope struct {
	Result []dividendPayoutWire `json:"result"`
}

type dividendPayoutWire struct {
	ExDate        string  `json:"exDate"`
	PaymentDate   string  `json:"paymentDate"`
	Currency      string  `json:"currency"`
	Cash          float64 `json:"cash"`
	CashKrw       float64 `json:"cashKrw"`
	YieldRatio    float64 `json:"yieldRatio"`
	TTMYieldRatio float64 `json:"ttmYieldRatio"`
}

// GetStockDividends fetches three dividend-related endpoints (TTM summary card,
// recent-range payouts, full history) and stitches them into a single view.
// All three return empty/zero values gracefully for non-dividend-paying stocks.
func (c *Client) GetStockDividends(ctx context.Context, symbol string) (domain.StockDividends, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockDividends{}, err
	}

	var card dividendYieldCardEnvelope
	cardURL := fmt.Sprintf("%s/api/v1/stock-infos/%s/dividends/yield-ratio/histories", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, cardURL, &card); err != nil {
		return domain.StockDividends{}, err
	}

	var years dividendYearsEnvelope
	yearsURL := fmt.Sprintf("%s/api/v1/stock-infos/dividend/%s/years", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, yearsURL, &years); err != nil {
		return domain.StockDividends{}, err
	}

	var hist dividendHistoryEnvelope
	histURL := fmt.Sprintf("%s/api/v1/stock-infos/dividend/%s/summary", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, histURL, &hist); err != nil {
		return domain.StockDividends{}, err
	}

	return domain.StockDividends{
		ProductCode: productCode,
		Summary: domain.DividendYieldCard{
			DividendCount:         card.Result.DividendCount,
			DividendMonths:        card.Result.DividendMonths,
			DividendCash:          card.Result.DividendCash,
			DividendCashKrw:       card.Result.DividendCashKrw,
			DividendYieldRatio:    card.Result.DividendYieldRatio,
			TTMDividendYieldRatio: card.Result.TTMDividendYieldRatio,
			TTMDividendMonths:     card.Result.TTMDividendMonths,
			TTMDps:                card.Result.TTMDps,
			TTMDpsKrw:             card.Result.TTMDpsKrw,
			TTMDividendTotalCount: card.Result.TTMDividendTotalCount,
			DividendGrowthRatio:   card.Result.DividendGrowthRatio,
			Currency:              card.Result.Currency,
		},
		RecentYears: domain.DividendYearsPayouts{
			StartDate:    years.Result.StartDate,
			RangeLabel:   years.Result.SelectedRange.DisplayName,
			Payouts:      convertPayouts(years.Result.Histories),
			TotalCash:    years.Result.TotalCash,
			TotalCashKrw: years.Result.TotalCashKrw,
		},
		FullHistory: convertPayouts(hist.Result),
		FetchedAt:   time.Now().UTC(),
	}, nil
}

func convertPayouts(src []dividendPayoutWire) []domain.DividendPayout {
	out := make([]domain.DividendPayout, len(src))
	for i, p := range src {
		out[i] = domain.DividendPayout{
			ExDate:        p.ExDate,
			PaymentDate:   p.PaymentDate,
			Currency:      p.Currency,
			Cash:          p.Cash,
			CashKrw:       p.CashKrw,
			YieldRatio:    p.YieldRatio,
			TTMYieldRatio: p.TTMYieldRatio,
		}
	}
	return out
}
