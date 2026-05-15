package client

import (
	"context"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type salesCompositionEnvelope struct {
	Result struct {
		Code       string `json:"code"`
		FiscalYear int    `json:"fiscalYear"`
		EndDate    string `json:"endDate"`
		Compositions []struct {
			Business string  `json:"business"`
			Product  string  `json:"product"`
			Ratio    float64 `json:"ratio"`
		} `json:"compositions"`
		DataSource string `json:"dataSource"`
	} `json:"result"`
}

// GetSalesComposition fetches the revenue-composition breakdown. The endpoint
// uses companyCode (NAS116LTR-E0 form) which is resolved via the overview call.
func (c *Client) GetSalesComposition(ctx context.Context, symbol string) (domain.SalesComposition, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.SalesComposition{}, err
	}
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return domain.SalesComposition{}, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/companies/%s/sales-compositions", c.infoBaseURL, companyCode)
	var env salesCompositionEnvelope
	if err := c.getJSON(ctx, endpoint, &env); err != nil {
		return domain.SalesComposition{}, err
	}
	items := make([]domain.SalesCompositionItem, len(env.Result.Compositions))
	for i, comp := range env.Result.Compositions {
		items[i] = domain.SalesCompositionItem{
			Business: comp.Business,
			Product:  comp.Product,
			Ratio:    comp.Ratio,
		}
	}
	return domain.SalesComposition{
		ProductCode: productCode,
		CompanyCode: env.Result.Code,
		FiscalYear:  env.Result.FiscalYear,
		EndDate:     env.Result.EndDate,
		Items:       items,
		DataSource:  env.Result.DataSource,
		FetchedAt:   time.Now().UTC(),
	}, nil
}
