package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type signalsBatchEnvelope struct {
	Result struct {
		Signals []struct {
			ProductCode          string `json:"productCode"`
			ReasoningDescription string `json:"reasoningDescription"`
		} `json:"signals"`
	} `json:"result"`
}

// ListSignals returns one-line Korean reasoning per code in the input list.
// Toss documents no explicit cap; observed batches of 100 codes return 200 ok.
func (c *Client) ListSignals(ctx context.Context, productCodes []string) ([]domain.Signal, error) {
	cleaned := make([]string, 0, len(productCodes))
	for _, code := range productCodes {
		code = strings.TrimSpace(code)
		if code != "" {
			cleaned = append(cleaned, code)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("ListSignals: at least one product code is required")
	}

	body, err := json.Marshal(map[string]any{"productCodes": cleaned})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/dashboard/wts/overview/ai-signals", c.infoBaseURL)
	var envelope signalsBatchEnvelope
	if err := c.postJSON(ctx, endpoint, body, &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.Signal, 0, len(envelope.Result.Signals))
	for _, s := range envelope.Result.Signals {
		out = append(out, domain.Signal{
			ProductCode:          s.ProductCode,
			ReasoningDescription: s.ReasoningDescription,
		})
	}
	return out, nil
}

type signalDetailEnvelope struct {
	Result struct {
		CreatedAt       time.Time `json:"createdAt"`
		SignalID        string    `json:"signalId"`
		SignalDirection int       `json:"signalDirection"`
		Reasoning       struct {
			Description string `json:"description"`
			Issue       struct {
				AssetName      string  `json:"assetName"`
				AssetCode      string  `json:"assetCode"`
				ProfitLossRate float64 `json:"profitLossRate"`
				Description    struct {
					Data []string `json:"data"`
				} `json:"description"`
			} `json:"issue"`
			News struct {
				Data []struct {
					ID         string    `json:"id"`
					Source     string    `json:"source"`
					AgencyName string    `json:"agencyName"`
					Title      string    `json:"title"`
					FaviconURL string    `json:"faviconUrl"`
					CreatedAt  time.Time `json:"createdAt"`
				} `json:"data"`
			} `json:"news"`
			Keywords []string `json:"keywords"`
		} `json:"reasoning"`
		RelatedReasoning struct {
			Details []struct {
				AssetCode     string `json:"assetCode"`
				AssetName     string `json:"assetName"`
				RelatedStocks []struct {
					StockCode string `json:"stockCode"`
					StockName string `json:"stockName"`
					Symbol    string `json:"symbol"`
				} `json:"relatedStocks"`
				Description struct {
					Data []string `json:"data"`
				} `json:"description"`
				Relationship struct {
					SubjectName string `json:"subjectName"`
					Relation    string `json:"relation"`
					ObjectName  string `json:"objectName"`
				} `json:"relationship"`
			} `json:"details"`
		} `json:"relatedReasoning"`
		HasRelatedReasoning bool `json:"hasRelatedReasoning"`
	} `json:"result"`
}

// GetSignalDetail returns the rich Korean reasoning + news + related stocks
// payload for a single product. The productType is fixed to "STOCKS" since
// non-stock Toss assets are out of scope.
func (c *Client) GetSignalDetail(ctx context.Context, productCode string) (domain.SignalDetail, error) {
	productCode = strings.TrimSpace(productCode)
	if productCode == "" {
		return domain.SignalDetail{}, fmt.Errorf("GetSignalDetail: productCode is required")
	}

	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/dashboard/wts/overview/ai-signals/detail", c.infoBaseURL))
	if err != nil {
		return domain.SignalDetail{}, err
	}
	q := endpoint.Query()
	q.Set("productCode", productCode)
	q.Set("productType", "STOCKS")
	endpoint.RawQuery = q.Encode()

	var envelope signalDetailEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return domain.SignalDetail{}, err
	}

	r := envelope.Result
	detail := domain.SignalDetail{
		ProductCode:      productCode,
		AssetName:        r.Reasoning.Issue.AssetName,
		SignalID:         r.SignalID,
		SignalDirection:  r.SignalDirection,
		CreatedAt:        r.CreatedAt,
		Description:      r.Reasoning.Description,
		DescriptionItems: r.Reasoning.Issue.Description.Data,
		ProfitLossRate:   r.Reasoning.Issue.ProfitLossRate,
		Keywords:         r.Reasoning.Keywords,
		FetchedAt:        time.Now().UTC(),
	}
	for _, n := range r.Reasoning.News.Data {
		detail.News = append(detail.News, domain.SignalNews{
			ID:         n.ID,
			Source:     n.Source,
			AgencyName: n.AgencyName,
			Title:      n.Title,
			FaviconURL: n.FaviconURL,
			CreatedAt:  n.CreatedAt,
		})
	}
	for _, rel := range r.RelatedReasoning.Details {
		entry := domain.RelatedSignal{
			AssetCode:   rel.AssetCode,
			AssetName:   rel.AssetName,
			Relation:    rel.Relationship.Relation,
			Description: rel.Description.Data,
		}
		if len(rel.RelatedStocks) > 0 {
			entry.StockCode = rel.RelatedStocks[0].StockCode
			entry.StockSymbol = rel.RelatedStocks[0].Symbol
		}
		detail.Related = append(detail.Related, entry)
	}
	return detail, nil
}

type eventSignalsEnvelope struct {
	Result struct {
		ExposeSignals bool `json:"exposeSignals"`
		SignalsList   []struct {
			ProductCode   string `json:"productCode"`
			PrimarySignal struct {
				SignalLabel string    `json:"signalLabel"`
				SignalInfo  string    `json:"signalInfo"`
				SignalID    int64     `json:"signalId"`
				DateTime    time.Time `json:"datetime"`
			} `json:"primarySignal"`
			Signals []struct {
				SignalLabel string    `json:"signalLabel"`
				SignalInfo  string    `json:"signalInfo"`
				SignalID    int64     `json:"signalId"`
				DateTime    time.Time `json:"datetime"`
			} `json:"signals"`
		} `json:"signalsList"`
	} `json:"result"`
}

// ListEventSignals returns scheduled event signals (earnings, disclosures, ...).
// Currently emits one row per product, using the "primarySignal" field.
func (c *Client) ListEventSignals(ctx context.Context, productCodes []string) ([]domain.EventSignal, error) {
	cleaned := make([]string, 0, len(productCodes))
	for _, code := range productCodes {
		code = strings.TrimSpace(code)
		if code != "" {
			cleaned = append(cleaned, code)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("ListEventSignals: at least one product code is required")
	}

	body, err := json.Marshal(map[string]any{"productCodes": cleaned, "filters": []string{}})
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/dashboard/wts/overview/signals", c.infoBaseURL)

	var envelope eventSignalsEnvelope
	if err := c.postJSON(ctx, endpoint, body, &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.EventSignal, 0, len(envelope.Result.SignalsList))
	for _, row := range envelope.Result.SignalsList {
		out = append(out, domain.EventSignal{
			ProductCode: row.ProductCode,
			SignalLabel: row.PrimarySignal.SignalLabel,
			SignalInfo:  row.PrimarySignal.SignalInfo,
			SignalID:    row.PrimarySignal.SignalID,
			DateTime:    row.PrimarySignal.DateTime,
		})
	}
	return out, nil
}
