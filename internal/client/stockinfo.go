package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// GetStockInfoDetail fetches the 종목정보 deep-tab payload for a product. The
// endpoint returns {"result": null} for products without coverage; this is
// surfaced as an empty Sections slice (no error).
func (c *Client) GetStockInfoDetail(ctx context.Context, symbol string) (domain.StockInfoDetail, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockInfoDetail{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/stock-detail/ui/%s/info", c.infoBaseURL, productCode)
	var envelope struct {
		Result *struct {
			Sections []domain.StockInfoSection `json:"sections"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.StockInfoDetail{}, err
	}
	out := domain.StockInfoDetail{
		ProductCode: productCode,
		FetchedAt:   time.Now().UTC(),
	}
	if envelope.Result != nil {
		out.Sections = envelope.Result.Sections
	}
	return out, nil
}
