package client

import (
	"context"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// GetOrderableSummary fans out to cached-orderable-amount and per-market
// transaction overview, returning a single struct for the CLI.
func (c *Client) GetOrderableSummary(ctx context.Context) (domain.OrderableSummary, error) {
	if err := c.requireSession(); err != nil {
		return domain.OrderableSummary{}, err
	}

	cached, err := c.GetCachedOrderableAmount(ctx)
	if err != nil {
		return domain.OrderableSummary{}, err
	}
	kr, err := c.GetTransactionsOverview(ctx, "kr")
	if err != nil {
		return domain.OrderableSummary{}, err
	}
	us, err := c.GetTransactionsOverview(ctx, "us")
	if err != nil {
		return domain.OrderableSummary{}, err
	}

	return domain.OrderableSummary{
		OrderableKR: cached.KR,
		OrderableUS: cached.US,
		KR:          kr,
		US:          us,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
