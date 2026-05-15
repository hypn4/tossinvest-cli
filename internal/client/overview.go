package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type overviewEnvelope struct {
	Result struct {
		Type   string `json:"type"`
		Market struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"market"`
		Company struct {
			Code            string `json:"code"`
			Name            string `json:"name"`
			EnglishName     string `json:"englishName"`
			FullEnglishName string `json:"fullEnglishName"`
			Industry        *struct {
				Code        string `json:"code"`
				DisplayName string `json:"displayName"`
			} `json:"industry"`
			Description       string  `json:"description"`
			EstablishYear     int     `json:"establishYear"`
			ListDate          string  `json:"listDate"`
			CEO               string  `json:"ceo"`
			HomepageURL       string  `json:"homepageUrl"`
			LogoImageURL      string  `json:"logoImageUrl"`
			SharesOutstanding int64   `json:"sharesOutstanding"`
			MarketValue       float64 `json:"marketValue"`
			MarketValueKrw    float64 `json:"marketValueKrw"`
			Currency          string  `json:"currency"`
		} `json:"company"`
		MarketValue        float64 `json:"marketValue"`
		MarketValueKrw     float64 `json:"marketValueKrw"`
		EnterpriseValue    float64 `json:"enterpriseValue"`
		EnterpriseValueKrw float64 `json:"enterpriseValueKrw"`
		DataSource         string  `json:"dataSource"`
		ListDate           string  `json:"listDate"`
	} `json:"result"`
}

// GetCompanyOverview fetches the company overview card (CEO, EV, industry,
// description, listing) for a product. Accepts symbol or productCode.
func (c *Client) GetCompanyOverview(ctx context.Context, symbol string) (domain.CompanyOverview, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.CompanyOverview{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/stock-infos/%s/overview", c.infoBaseURL, productCode)
	var env overviewEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.CompanyOverview{}, err
	}
	r := env.Result
	out := domain.CompanyOverview{
		ProductCode:        productCode,
		Type:               r.Type,
		MarketCode:         r.Market.Code,
		Market:             r.Market.DisplayName,
		ListDate:           r.ListDate,
		MarketValue:        r.MarketValue,
		MarketValueKrw:     r.MarketValueKrw,
		EnterpriseValue:    r.EnterpriseValue,
		EnterpriseValueKrw: r.EnterpriseValueKrw,
		DataSource:         r.DataSource,
		Company: domain.CompanyProfile{
			Code:              r.Company.Code,
			Name:              r.Company.Name,
			EnglishName:       r.Company.EnglishName,
			FullEnglishName:   r.Company.FullEnglishName,
			Description:       r.Company.Description,
			EstablishYear:     r.Company.EstablishYear,
			ListDate:          r.Company.ListDate,
			CEO:               r.Company.CEO,
			HomepageURL:       r.Company.HomepageURL,
			LogoImageURL:      r.Company.LogoImageURL,
			SharesOutstanding: r.Company.SharesOutstanding,
			MarketValue:       r.Company.MarketValue,
			MarketValueKrw:    r.Company.MarketValueKrw,
			Currency:          r.Company.Currency,
		},
		FetchedAt: time.Now().UTC(),
	}
	if r.Company.Industry != nil {
		out.Company.IndustryCode = r.Company.Industry.Code
		out.Company.IndustryName = r.Company.Industry.DisplayName
	}
	return out, nil
}
