package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type evaluationEnvelope struct {
	Result struct {
		PER      float64 `json:"per"`
		PBR      float64 `json:"pbr"`
		PSR      float64 `json:"psr"`
		Median   float64 `json:"median"`
		Position string  `json:"position"`
	} `json:"result"`
}

type evaluationComparisonEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code string `json:"code"`
		} `json:"selectedFactor"`
		SelectedTics struct {
			DisplayName string `json:"displayName"`
		} `json:"selectedTics"`
		StockGraphs []struct {
			Code  string `json:"code"`
			Name  string `json:"name"`
			Graph []struct {
				Period string  `json:"period"`
				Value  float64 `json:"value"`
			} `json:"graph"`
		} `json:"stockGraphs"`
	} `json:"result"`
}

// GetStockValuation fetches the per-stock valuation snapshot and peer
// comparison matrix. Two POST calls (both with empty body) are stitched into
// a single StockValuation result.
func (c *Client) GetStockValuation(ctx context.Context, symbol string) (domain.StockValuation, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockValuation{}, err
	}

	evalEndpoint := fmt.Sprintf("%s/api/v2/stock-infos/evaluation/%s", c.infoBaseURL, productCode)
	var evalEnv evaluationEnvelope
	if err := c.postJSONEmpty(ctx, evalEndpoint, &evalEnv); err != nil {
		return domain.StockValuation{}, err
	}

	cmpEndpoint := fmt.Sprintf("%s/api/v2/stock-infos/evaluation-comparison/%s", c.infoBaseURL, productCode)
	var cmpEnv evaluationComparisonEnvelope
	if err := c.postJSONEmpty(ctx, cmpEndpoint, &cmpEnv); err != nil {
		return domain.StockValuation{}, err
	}

	peers := make([]domain.PeerValuation, 0, len(cmpEnv.Result.StockGraphs))
	for _, g := range cmpEnv.Result.StockGraphs {
		if len(g.Graph) == 0 {
			continue
		}
		last := g.Graph[len(g.Graph)-1]
		peers = append(peers, domain.PeerValuation{
			ProductCode: g.Code,
			Name:        g.Name,
			Value:       last.Value,
			Period:      last.Period,
			IsSelf:      g.Code == productCode,
		})
	}

	return domain.StockValuation{
		ProductCode: productCode,
		PER:         evalEnv.Result.PER,
		PBR:         evalEnv.Result.PBR,
		PSR:         evalEnv.Result.PSR,
		Median:      evalEnv.Result.Median,
		Position:    evalEnv.Result.Position,
		Factor:      cmpEnv.Result.SelectedFactor.Code,
		Industry:    cmpEnv.Result.SelectedTics.DisplayName,
		Peers:       peers,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
