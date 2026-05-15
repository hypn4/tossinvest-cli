package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type analystOpinionEnvelope struct {
	Result struct {
		Type       string `json:"type"`
		StrongSell int    `json:"strongSell"`
		Sell       int    `json:"sell"`
		Hold       int    `json:"hold"`
		Buy        int    `json:"buy"`
		StrongBuy  int    `json:"strongBuy"`
		TargetPrice struct {
			USD float64 `json:"USD"`
			KRW float64 `json:"KRW"`
		} `json:"targetPrice"`
		Description string `json:"description"`
	} `json:"result"`
}

type consensusEnvelope struct {
	Result struct {
		TargetPrice struct {
			Mean     float64 `json:"mean"`
			High     float64 `json:"high"`
			Low      float64 `json:"low"`
			MeanKrw  float64 `json:"meanKrw"`
			HighKrw  float64 `json:"highKrw"`
			LowKrw   float64 `json:"lowKrw"`
			Currency string  `json:"currency"`
		} `json:"targetPrice"`
		PointDate       string `json:"pointDate"`
		PastClosePrices []struct {
			Price    float64 `json:"price"`
			PriceKrw float64 `json:"priceKrw"`
			Date     string  `json:"date"`
		} `json:"pastClosePrices"`
	} `json:"result"`
}

type analystReportsEnvelope struct {
	Result struct {
		AnalystReportGroups []struct {
			Date    string `json:"date"`
			Reports []struct {
				Title  string `json:"title"`
				Source string `json:"source"`
				URL    string `json:"url"`
			} `json:"reports"`
		} `json:"analystReportGroups"`
	} `json:"result"`
}

// GetAnalystSnapshot stitches opinion + consensus + reports into one bundle.
func (c *Client) GetAnalystSnapshot(ctx context.Context, symbol string) (domain.AnalystSnapshot, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var opEnv analystOpinionEnvelope
	opURL := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/analyst-opinion", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, opURL, &opEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var conEnv consensusEnvelope
	conURL := fmt.Sprintf("%s/api/v2/stock-infos/consensus/%s", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, conURL, &conEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	var repEnv analystReportsEnvelope
	repURL := fmt.Sprintf("%s/api/v1/stock-detail/ui/wts/%s/analyst-reports", c.infoBaseURL, productCode)
	if err := c.getJSON(ctx, repURL, &repEnv); err != nil {
		return domain.AnalystSnapshot{}, err
	}

	pastCloses := make([]domain.ConsensusPastClose, len(conEnv.Result.PastClosePrices))
	for i, p := range conEnv.Result.PastClosePrices {
		pastCloses[i] = domain.ConsensusPastClose{Date: p.Date, Price: p.Price, PriceKRW: p.PriceKrw}
	}

	var reports []domain.AnalystReport
	for _, group := range repEnv.Result.AnalystReportGroups {
		for _, r := range group.Reports {
			reports = append(reports, domain.AnalystReport{
				Title: r.Title, Source: r.Source, Date: group.Date, URL: r.URL,
			})
		}
	}

	return domain.AnalystSnapshot{
		ProductCode: productCode,
		Opinion: domain.AnalystOpinion{
			Type:        opEnv.Result.Type,
			StrongBuy:   opEnv.Result.StrongBuy,
			Buy:         opEnv.Result.Buy,
			Hold:        opEnv.Result.Hold,
			Sell:        opEnv.Result.Sell,
			StrongSell:  opEnv.Result.StrongSell,
			TargetUSD:   opEnv.Result.TargetPrice.USD,
			TargetKRW:   opEnv.Result.TargetPrice.KRW,
			Description: opEnv.Result.Description,
		},
		Consensus: domain.ConsensusTarget{
			Mean:       conEnv.Result.TargetPrice.Mean,
			High:       conEnv.Result.TargetPrice.High,
			Low:        conEnv.Result.TargetPrice.Low,
			MeanKRW:    conEnv.Result.TargetPrice.MeanKrw,
			HighKRW:    conEnv.Result.TargetPrice.HighKrw,
			LowKRW:     conEnv.Result.TargetPrice.LowKrw,
			Currency:   conEnv.Result.TargetPrice.Currency,
			PointDate:  conEnv.Result.PointDate,
			PastCloses: pastCloses,
		},
		Reports:   reports,
		FetchedAt: time.Now().UTC(),
	}, nil
}
