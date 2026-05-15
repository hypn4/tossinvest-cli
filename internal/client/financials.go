package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type stabilityEnvelope struct {
	Result struct {
		LiabilityRatio        float64 `json:"liabilityRatio"`
		CurrentRatio          float64 `json:"currentRatio"`
		InterestCoverageRatio float64 `json:"interestCoverageRatio"`
		Median                float64 `json:"median"`
		Position              string  `json:"position"`
	} `json:"result"`
}

type revenueEnvelope struct {
	Result struct {
		CompanyName         string  `json:"companyName"`
		RecentFiscalYear    int     `json:"recentFiscalYear"`
		RecentFiscalQuarter int     `json:"recentFiscalQuarter"`
		RecentNetProfit     float64 `json:"recentNetProfit"`
		RecentNetProfitKrw  float64 `json:"recentNetProfitKrw"`
		FluctuationRate     float64 `json:"fluctuationRate"`
		Position            string  `json:"position"`
		Graph               []struct {
			Period         string  `json:"period"`
			Revenue        float64 `json:"revenue"`
			RevenueKrw     float64 `json:"revenueKrw"`
			NetProfit      float64 `json:"netProfit"`
			NetProfitKrw   float64 `json:"netProfitKrw"`
			NetProfitRatio float64 `json:"netProfitRatio"`
		} `json:"graph"`
	} `json:"result"`
}

type operatingIncomeEnvelope struct {
	Result struct {
		CompanyName              string  `json:"companyName"`
		RecentFiscalYear         int     `json:"recentFiscalYear"`
		RecentFiscalQuarter      int     `json:"recentFiscalQuarter"`
		RecentOperatingIncome    float64 `json:"recentOperatingIncome"`
		RecentOperatingIncomeKrw float64 `json:"recentOperatingIncomeKrw"`
		FluctuationRate          float64 `json:"fluctuationRate"`
		Position                 string  `json:"position"`
		Graph                    []struct {
			Period               string  `json:"period"`
			OperatingIncome      float64 `json:"operatingIncome"`
			OperatingIncomeKrw   float64 `json:"operatingIncomeKrw"`
			OperatingIncomeRatio float64 `json:"operatingIncomeRatio"`
		} `json:"graph"`
	} `json:"result"`
}

// GetStockFinancials fetches three financial-snapshot endpoints (stability,
// revenue-and-net-profit, operating-income) for a stock and stitches them
// into a single StockFinancials view. All three are POST with empty body.
func (c *Client) GetStockFinancials(ctx context.Context, symbol string) (domain.StockFinancials, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockFinancials{}, err
	}

	var stab stabilityEnvelope
	stabURL := fmt.Sprintf("%s/api/v2/stock-infos/stability/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, stabURL, &stab); err != nil {
		return domain.StockFinancials{}, err
	}

	var rev revenueEnvelope
	revURL := fmt.Sprintf("%s/api/v2/stock-infos/revenue-and-net-profit/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, revURL, &rev); err != nil {
		return domain.StockFinancials{}, err
	}

	var op operatingIncomeEnvelope
	opURL := fmt.Sprintf("%s/api/v2/stock-infos/operating-income/%s", c.infoBaseURL, productCode)
	if err := c.postJSONEmpty(ctx, opURL, &op); err != nil {
		return domain.StockFinancials{}, err
	}

	revGraph := make([]domain.RevenuePoint, len(rev.Result.Graph))
	for i, g := range rev.Result.Graph {
		revGraph[i] = domain.RevenuePoint{
			Period:         g.Period,
			Revenue:        g.Revenue,
			RevenueKrw:     g.RevenueKrw,
			NetProfit:      g.NetProfit,
			NetProfitKrw:   g.NetProfitKrw,
			NetProfitRatio: g.NetProfitRatio,
		}
	}

	opGraph := make([]domain.OperatingIncomePoint, len(op.Result.Graph))
	for i, g := range op.Result.Graph {
		opGraph[i] = domain.OperatingIncomePoint{
			Period:               g.Period,
			OperatingIncome:      g.OperatingIncome,
			OperatingIncomeKrw:   g.OperatingIncomeKrw,
			OperatingIncomeRatio: g.OperatingIncomeRatio,
		}
	}

	return domain.StockFinancials{
		ProductCode: productCode,
		Stability: domain.StabilityRatios{
			LiabilityRatio:        stab.Result.LiabilityRatio,
			CurrentRatio:          stab.Result.CurrentRatio,
			InterestCoverageRatio: stab.Result.InterestCoverageRatio,
			IndustryMedian:        stab.Result.Median,
			Position:              stab.Result.Position,
		},
		Revenue: domain.RevenueSeries{
			CompanyName:         rev.Result.CompanyName,
			RecentFiscalYear:    rev.Result.RecentFiscalYear,
			RecentFiscalQuarter: rev.Result.RecentFiscalQuarter,
			RecentNetProfit:     rev.Result.RecentNetProfit,
			RecentNetProfitKrw:  rev.Result.RecentNetProfitKrw,
			FluctuationRate:     rev.Result.FluctuationRate,
			Position:            rev.Result.Position,
			Graph:               revGraph,
		},
		OperatingIncome: domain.OperatingIncomeSeries{
			CompanyName:              op.Result.CompanyName,
			RecentFiscalYear:         op.Result.RecentFiscalYear,
			RecentFiscalQuarter:      op.Result.RecentFiscalQuarter,
			RecentOperatingIncome:    op.Result.RecentOperatingIncome,
			RecentOperatingIncomeKrw: op.Result.RecentOperatingIncomeKrw,
			FluctuationRate:          op.Result.FluctuationRate,
			Position:                 op.Result.Position,
			Graph:                    opGraph,
		},
		FetchedAt: time.Now().UTC(),
	}, nil
}
