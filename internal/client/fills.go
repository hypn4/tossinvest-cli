package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type compactExecutedEnvelope struct {
	Result struct {
		Body []struct {
			ProductCode               string    `json:"productCode"`
			TradeType                 string    `json:"tradeType"`
			ExecutionAvgKrwPrice      float64   `json:"executionAvgKrwPrice"`
			ExecutionTotalKrwAmount   float64   `json:"executionTotalKrwAmount"`
			ExecutionAvgLocalPrice    float64   `json:"executionAvgLocalPrice"`
			ExecutionTotalLocalAmount float64   `json:"executionTotalLocalAmount"`
			Quantity                  float64   `json:"quantity"`
			ExecutedTimeBucket        time.Time `json:"executedTimeBucket"`
		} `json:"body"`
	} `json:"result"`
}

// ListCompactExecutions wraps /api/v3/trading/orders/histories/compact/executed.
// timeUnit accepted by Toss includes "thirty_minute" (verified in capture);
// other granularities are observed but not validated here — we forward as-is.
func (c *Client) ListCompactExecutions(ctx context.Context, productCode, timeUnit string) ([]domain.CompactExecution, error) {
	if err := c.requireSession(); err != nil {
		return nil, err
	}
	productCode = strings.TrimSpace(productCode)
	if productCode == "" {
		return nil, fmt.Errorf("productCode is required")
	}
	timeUnit = strings.TrimSpace(timeUnit)
	if timeUnit == "" {
		timeUnit = "thirty_minute"
	}

	endpoint, err := url.Parse(c.certBaseURL + "/api/v3/trading/orders/histories/compact/executed")
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("productCode", productCode)
	q.Set("timeUnit", timeUnit)
	q.Set("excludeSavings", "false")
	endpoint.RawQuery = q.Encode()

	var envelope compactExecutedEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.CompactExecution, 0, len(envelope.Result.Body))
	for _, raw := range envelope.Result.Body {
		out = append(out, domain.CompactExecution{
			ProductCode:             raw.ProductCode,
			TradeType:               raw.TradeType,
			ExecutionAvgKRWPrice:    raw.ExecutionAvgKrwPrice,
			ExecutionAvgLocalPrice:  raw.ExecutionAvgLocalPrice,
			ExecutionTotalKRWAmount: raw.ExecutionTotalKrwAmount,
			ExecutionTotalLocal:     raw.ExecutionTotalLocalAmount,
			Quantity:                raw.Quantity,
			BucketStart:             raw.ExecutedTimeBucket,
		})
	}
	return out, nil
}
