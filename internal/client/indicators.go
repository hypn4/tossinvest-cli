package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type indicatorsEnvelope struct {
	Result struct {
		IndicatorSections []struct {
			SectionName string                 `json:"sectionName"`
			Data        domain.IndicatorFields `json:"data"`
		} `json:"indicatorSections"`
	} `json:"result"`
}

// GetStockIndicators fetches the investment-indicators payload (가치평가/수익/
// 배당/안정성). Accepts symbol or productCode.
func (c *Client) GetStockIndicators(ctx context.Context, symbol string) (domain.StockIndicators, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockIndicators{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/investment-indicators", c.infoBaseURL, productCode)
	var env indicatorsEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.StockIndicators{}, err
	}
	sections := make(map[string]domain.IndicatorFields, len(env.Result.IndicatorSections))
	for _, s := range env.Result.IndicatorSections {
		sections[s.SectionName] = s.Data
	}
	return domain.StockIndicators{
		ProductCode: productCode,
		Sections:    sections,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
