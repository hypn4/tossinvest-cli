package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type estimateDateEnvelope struct {
	Result struct {
		AnnounceAt            *string  `json:"announceAt"`
		RevenueEst            *float64 `json:"revenueEst"`
		RevenueEstKrw         *float64 `json:"revenueEstKrw"`
		EpsEst                *float64 `json:"epsEst"`
		EpsEstKrw             *float64 `json:"epsEstKrw"`
		OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
		OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
	} `json:"result"`
}

type estimateRevenueEnvelope struct {
	Result struct {
		RevenueEst      *float64 `json:"revenueEst"`
		RevenueEstKrw   *float64 `json:"revenueEstKrw"`
		FluctuationRate float64  `json:"fluctuationRate"`
		Fluctuation     float64  `json:"fluctuation"`
		FluctuationKrw  float64  `json:"fluctuationKrw"`
		Position        *string  `json:"position"`
		Graphs          []struct {
			Period        string   `json:"period"`
			Revenue       *float64 `json:"revenue"`
			RevenueEst    *float64 `json:"revenueEst"`
			RevenueKrw    *float64 `json:"revenueKrw"`
			RevenueEstKrw *float64 `json:"revenueEstKrw"`
			Surprise      *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

type estimateEpsEnvelope struct {
	Result struct {
		EpsEst          *float64 `json:"epsEst"`
		EpsEstKrw       *float64 `json:"epsEstKrw"`
		FluctuationRate float64  `json:"fluctuationRate"`
		Fluctuation     float64  `json:"fluctuation"`
		FluctuationKrw  float64  `json:"fluctuationKrw"`
		Position        *string  `json:"position"`
		Graphs          []struct {
			Period    string   `json:"period"`
			Eps       *float64 `json:"eps"`
			EpsEst    *float64 `json:"epsEst"`
			EpsKrw    *float64 `json:"epsKrw"`
			EpsEstKrw *float64 `json:"epsEstKrw"`
			Surprise  *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

type estimateOperatingIncomeEnvelope struct {
	Result struct {
		OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
		OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
		FluctuationRate       float64  `json:"fluctuationRate"`
		Fluctuation           float64  `json:"fluctuation"`
		FluctuationKrw        float64  `json:"fluctuationKrw"`
		Position              *string  `json:"position"`
		Graphs                []struct {
			Period                string   `json:"period"`
			OperatingIncome       *float64 `json:"operatingIncome"`
			OperatingIncomeEst    *float64 `json:"operatingIncomeEst"`
			OperatingIncomeKrw    *float64 `json:"operatingIncomeKrw"`
			OperatingIncomeEstKrw *float64 `json:"operatingIncomeEstKrw"`
			Surprise              *float64 `json:"surprise"`
		} `json:"graphs"`
	} `json:"result"`
}

// GetStockEstimates fetches the next-earnings consensus headline plus three
// time-series endpoints (revenue / EPS / operating-income estimates). The
// headline is a GET; the three series are POST with empty body. All four use
// productCode (not companyCode).
func (c *Client) GetStockEstimates(ctx context.Context, symbol string) (domain.StockEstimates, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockEstimates{}, err
	}

	base := fmt.Sprintf("%s/api/v2/companies/%s/financial/estimate", c.infoBaseURL, productCode)

	var hd estimateDateEnvelope
	if err := c.getJSON(ctx, base+"/date", &hd); err != nil {
		return domain.StockEstimates{}, err
	}

	var rev estimateRevenueEnvelope
	if err := c.postJSONEmpty(ctx, base+"/revenue", &rev); err != nil {
		return domain.StockEstimates{}, err
	}

	var eps estimateEpsEnvelope
	if err := c.postJSONEmpty(ctx, base+"/eps", &eps); err != nil {
		return domain.StockEstimates{}, err
	}

	var op estimateOperatingIncomeEnvelope
	if err := c.postJSONEmpty(ctx, base+"/operating-income", &op); err != nil {
		return domain.StockEstimates{}, err
	}

	revPoints := make([]domain.EstimateRevenuePoint, len(rev.Result.Graphs))
	for i, g := range rev.Result.Graphs {
		revPoints[i] = domain.EstimateRevenuePoint{
			Period: g.Period, Revenue: g.Revenue, RevenueEst: g.RevenueEst,
			RevenueKrw: g.RevenueKrw, RevenueEstKrw: g.RevenueEstKrw, Surprise: g.Surprise,
		}
	}
	epsPoints := make([]domain.EstimateEpsPoint, len(eps.Result.Graphs))
	for i, g := range eps.Result.Graphs {
		epsPoints[i] = domain.EstimateEpsPoint{
			Period: g.Period, EPS: g.Eps, EPSEst: g.EpsEst,
			EPSKrw: g.EpsKrw, EPSEstKrw: g.EpsEstKrw, Surprise: g.Surprise,
		}
	}
	opPoints := make([]domain.EstimateOperatingIncomePoint, len(op.Result.Graphs))
	for i, g := range op.Result.Graphs {
		opPoints[i] = domain.EstimateOperatingIncomePoint{
			Period: g.Period, OperatingIncome: g.OperatingIncome, OperatingIncomeEst: g.OperatingIncomeEst,
			OperatingIncomeKrw: g.OperatingIncomeKrw, OperatingIncomeEstKrw: g.OperatingIncomeEstKrw, Surprise: g.Surprise,
		}
	}

	return domain.StockEstimates{
		ProductCode: productCode,
		Headline: domain.EstimateHeadline{
			AnnounceAt:            hd.Result.AnnounceAt,
			RevenueEst:            hd.Result.RevenueEst,
			RevenueEstKrw:         hd.Result.RevenueEstKrw,
			EPSEst:                hd.Result.EpsEst,
			EPSEstKrw:             hd.Result.EpsEstKrw,
			OperatingIncomeEst:    hd.Result.OperatingIncomeEst,
			OperatingIncomeEstKrw: hd.Result.OperatingIncomeEstKrw,
		},
		Revenue: domain.EstimateRevenueSeries{
			RevenueEst: rev.Result.RevenueEst, RevenueEstKrw: rev.Result.RevenueEstKrw,
			FluctuationRate: rev.Result.FluctuationRate, Fluctuation: rev.Result.Fluctuation, FluctuationKrw: rev.Result.FluctuationKrw,
			Position: rev.Result.Position, Graph: revPoints,
		},
		EPS: domain.EstimateEpsSeries{
			EPSEst: eps.Result.EpsEst, EPSEstKrw: eps.Result.EpsEstKrw,
			FluctuationRate: eps.Result.FluctuationRate, Fluctuation: eps.Result.Fluctuation, FluctuationKrw: eps.Result.FluctuationKrw,
			Position: eps.Result.Position, Graph: epsPoints,
		},
		OperatingIncome: domain.EstimateOperatingIncomeSeries{
			OperatingIncomeEst: op.Result.OperatingIncomeEst, OperatingIncomeEstKrw: op.Result.OperatingIncomeEstKrw,
			FluctuationRate: op.Result.FluctuationRate, Fluctuation: op.Result.Fluctuation, FluctuationKrw: op.Result.FluctuationKrw,
			Position: op.Result.Position, Graph: opPoints,
		},
		FetchedAt: time.Now().UTC(),
	}, nil
}
