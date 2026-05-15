package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type ticsEnvelope struct {
	Result struct {
		BaseDate  string      `json:"baseDate"`
		MajorList []ticsEntry `json:"majorList"`
		MinorList []ticsEntry `json:"minorList"`
	} `json:"result"`
}

type ticsEntry struct {
	ID             int           `json:"id"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	CompanyCount   int           `json:"companyCount"`
	Representative bool          `json:"representative"`
	Rankings       []ticsRanking `json:"rankings"`
}

type ticsRanking struct {
	BaseDate     string `json:"baseDate"`
	FiscalPeriod string `json:"fiscalPeriod"`
	Type         struct {
		DisplayName string `json:"displayName"`
	} `json:"type"`
	Ranking      int     `json:"ranking"`
	CompanyCount int     `json:"companyCount"`
	DisplayValue string  `json:"displayValue"`
	Value        float64 `json:"value"`
}

// GetTICSIndustry fetches the TICS industry taxonomy and peer rankings.
func (c *Client) GetTICSIndustry(ctx context.Context, symbol string) (domain.TICSIndustry, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.TICSIndustry{}, err
	}
	// Pass productCode (not symbol) to avoid a second search round-trip:
	// resolveCompanyCode → GetCompanyOverview → resolveProductCode short-circuits
	// when the input already looks like a productCode.
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return domain.TICSIndustry{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/tics", c.infoBaseURL, companyCode)
	var env ticsEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.TICSIndustry{}, err
	}
	return domain.TICSIndustry{
		ProductCode: productCode,
		CompanyCode: companyCode,
		BaseDate:    env.Result.BaseDate,
		Major:       convertTICSEntries(env.Result.MajorList),
		Minor:       convertTICSEntries(env.Result.MinorList),
		FetchedAt:   time.Now().UTC(),
	}, nil
}

func convertTICSEntries(src []ticsEntry) []domain.TICSEntry {
	out := make([]domain.TICSEntry, len(src))
	for i, e := range src {
		ranks := make([]domain.TICSRanking, len(e.Rankings))
		for j, r := range e.Rankings {
			ranks[j] = domain.TICSRanking{
				BaseDate:     r.BaseDate,
				FiscalPeriod: r.FiscalPeriod,
				TypeName:     r.Type.DisplayName,
				Ranking:      r.Ranking,
				CompanyCount: r.CompanyCount,
				DisplayValue: r.DisplayValue,
				Value:        r.Value,
			}
		}
		out[i] = domain.TICSEntry{
			ID:             e.ID,
			Title:          e.Title,
			Description:    e.Description,
			CompanyCount:   e.CompanyCount,
			Representative: e.Representative,
			Rankings:       ranks,
		}
	}
	return out
}
