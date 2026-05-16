package client

import (
	"context"
	"fmt"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type newsEnvelope struct {
	Result struct {
		Body []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Summary   string   `json:"summary"`
			ImageURLs []string `json:"imageUrls"`
			Source    struct {
				Code         string `json:"code"`
				Name         string `json:"name"`
				LogoImageURL string `json:"logoImageUrl"`
			} `json:"source"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
		} `json:"body"`
		LastPage bool `json:"lastPage"`
	} `json:"result"`
}

type filingsEnvelope struct {
	Result struct {
		Body []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			CompanyCode string `json:"companyCode"`
			StockCode   string `json:"stockCode"`
			Form        string `json:"form"`
			ReportID    string `json:"reportId"`
			EarningCall *struct {
				Status      string `json:"status"`
				LandingURL  string `json:"landingUrl"`
				Title       string `json:"title"`
				ReportTitle string `json:"reportTitle"`
				LiveAt      string `json:"liveAt"`
				ZonedLiveAt string `json:"zonedLiveAt"`
			} `json:"earningCall"`
			CreatedAt string `json:"createdAt"`
		} `json:"body"`
		LastPage bool `json:"lastPage"`
	} `json:"result"`
}

// ListStockNews returns the latest news items for a stock. Underlying endpoint
// pages 20 items at a time; this method auto-pages until `count` items are
// collected or the server reports lastPage. Accepts symbol or productCode.
func (c *Client) ListStockNews(ctx context.Context, symbol string, count int) ([]domain.NewsItem, error) {
	if count <= 0 {
		return nil, fmt.Errorf("ListStockNews: count must be > 0 (got %d)", count)
	}
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	// Pass productCode (not symbol) to skip the redundant search round-trip.
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return nil, err
	}

	const pageSize = 20
	out := make([]domain.NewsItem, 0, count)
	for page := 1; len(out) < count; page++ {
		var endpoint string
		// First page omits number= to match Toss web; do that here for parity.
		if page == 1 {
			endpoint = fmt.Sprintf("%s/api/v2/news/companies/%s?size=%d&orderBy=latest", c.infoBaseURL, companyCode, pageSize)
		} else {
			endpoint = fmt.Sprintf("%s/api/v2/news/companies/%s?size=%d&orderBy=latest&number=%d", c.infoBaseURL, companyCode, pageSize, page)
		}
		var env newsEnvelope
		if err := c.getJSON(ctx, endpoint, &env); err != nil {
			return nil, err
		}
		for _, n := range env.Result.Body {
			if len(out) >= count {
				break
			}
			out = append(out, domain.NewsItem{
				ID: n.ID, Title: n.Title, Summary: n.Summary, ImageURLs: n.ImageURLs,
				Source: domain.NewsSource{
					Code: n.Source.Code, Name: n.Source.Name, LogoImageURL: n.Source.LogoImageURL,
				},
				CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
			})
		}
		if env.Result.LastPage || len(env.Result.Body) == 0 {
			break
		}
	}
	return out, nil
}

// ListStockFilings returns the latest KR filings (DART + KIND) for a stock.
// Returns an empty slice for US stocks. Pagination behavior matches news.
func (c *Client) ListStockFilings(ctx context.Context, symbol string, count int) ([]domain.FilingItem, error) {
	if count <= 0 {
		return nil, fmt.Errorf("ListStockFilings: count must be > 0 (got %d)", count)
	}
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return nil, err
	}

	const pageSize = 20
	out := make([]domain.FilingItem, 0, count)
	for page := 1; len(out) < count; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/stock-detail/companies/%s/filings?number=%d&size=%d", c.infoBaseURL, companyCode, page, pageSize)
		var env filingsEnvelope
		if err := c.getJSON(ctx, endpoint, &env); err != nil {
			return nil, err
		}
		for _, f := range env.Result.Body {
			if len(out) >= count {
				break
			}
			it := domain.FilingItem{
				ID: f.ID, Title: f.Title, Summary: f.Summary,
				CompanyCode: f.CompanyCode, StockCode: f.StockCode,
				Form: f.Form, ReportID: f.ReportID, CreatedAt: f.CreatedAt,
			}
			if f.EarningCall != nil {
				it.EarningCall = &domain.EarningCall{
					Status:      f.EarningCall.Status,
					LandingURL:  f.EarningCall.LandingURL,
					Title:       f.EarningCall.Title,
					ReportTitle: f.EarningCall.ReportTitle,
					LiveAt:      f.EarningCall.LiveAt,
					ZonedLiveAt: f.EarningCall.ZonedLiveAt,
				}
			}
			out = append(out, it)
		}
		if env.Result.LastPage || len(env.Result.Body) == 0 {
			break
		}
	}
	return out, nil
}
