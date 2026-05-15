package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type ratiosEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedFactor"`
		SelectedRange struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedRange"`
		SelectedPeriod struct {
			Code string `json:"code"`
		} `json:"selectedPeriod"`
		Graph []struct {
			Code   string `json:"code"`
			Unit   string `json:"unit"`
			Name   string `json:"name"`
			Values []struct {
				Period   string  `json:"period"`
				Value    float64 `json:"value"`
				ValueKrw float64 `json:"valueKrw"`
			} `json:"values"`
		} `json:"graph"`
	} `json:"result"`
}

// GetStockRatios fetches the financial-statements/comprehensive payload —
// the time series for the default factor (DEBT_RATIO) and its component
// line items, across the default range (3년) and period (Q). Toss's server
// supports other factor/range/period selections via body, but PR14 ships
// only the default case; future PRs may add --factor / --period flags
// after fresh captures verify those variants.
func (c *Client) GetStockRatios(ctx context.Context, symbol string) (domain.StockRatios, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockRatios{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statements/comprehensive", c.infoBaseURL, productCode)
	var env ratiosEnvelope
	if err := c.postJSONEmpty(ctx, endpoint, &env); err != nil {
		return domain.StockRatios{}, err
	}

	items := make([]domain.RatioLineItem, len(env.Result.Graph))
	for i, g := range env.Result.Graph {
		vals := make([]domain.RatioValue, len(g.Values))
		for j, v := range g.Values {
			vals[j] = domain.RatioValue{Period: v.Period, Value: v.Value, ValueKrw: v.ValueKrw}
		}
		items[i] = domain.RatioLineItem{
			Code:   g.Code,
			Unit:   g.Unit,
			Name:   g.Name,
			Values: vals,
		}
	}

	return domain.StockRatios{
		ProductCode: productCode,
		Factor: domain.RatioFactor{
			Code:        env.Result.SelectedFactor.Code,
			DisplayName: env.Result.SelectedFactor.DisplayName,
		},
		Period:     env.Result.SelectedPeriod.Code,
		RangeLabel: env.Result.SelectedRange.DisplayName,
		Items:      items,
		FetchedAt:  time.Now().UTC(),
	}, nil
}
