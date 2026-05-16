package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// GetStockRatios fetches the financial-statements/comprehensive endpoint for
// a selected factor + period. The Toss API accepts a JSON body
// {factorCode, period} and returns 3 line items (two components + the
// derived ratio) across N periods (3-year default; the 1년/3년/5년/전체
// range is server-controlled and not exposed via this call).
//
// Validated factor codes: DEBT_RATIO, CURRENT_RATIO, INTEREST_COVERAGE_RATIO.
// Validated periods: Q (quarterly), Y (annual). Inputs are case-insensitive;
// they are normalized to uppercase before the wire call.
func (c *Client) GetStockRatios(ctx context.Context, symbol, factorCode, period string) (domain.StockRatios, error) {
	factorCode = strings.ToUpper(factorCode)
	switch factorCode {
	case "DEBT_RATIO", "CURRENT_RATIO", "INTEREST_COVERAGE_RATIO":
	default:
		return domain.StockRatios{}, fmt.Errorf("GetStockRatios: factorCode must be one of DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO (got %q)", factorCode)
	}
	period = strings.ToUpper(period)
	if period != "Q" && period != "Y" {
		return domain.StockRatios{}, fmt.Errorf("GetStockRatios: period must be Q or Y (got %q)", period)
	}

	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockRatios{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statements/comprehensive", c.infoBaseURL, productCode)

	bodyBytes, err := json.Marshal(map[string]string{"factorCode": factorCode, "period": period})
	if err != nil {
		return domain.StockRatios{}, err
	}

	var env ratiosEnvelope
	if err := c.postJSON(ctx, endpoint, json.RawMessage(bodyBytes), &env); err != nil {
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
