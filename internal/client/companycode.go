package client

import (
	"context"
	"fmt"
)

// resolveCompanyCode maps a symbol or productCode to the company-level code
// returned by /api/v2/stock-infos/{code}/overview (e.g. NAS116LTR-E0). This
// company code is required by /api/v1/companies/{code}/... endpoints.
func (c *Client) resolveCompanyCode(ctx context.Context, symbolOrCode string) (string, error) {
	ov, err := c.GetCompanyOverview(ctx, symbolOrCode)
	if err != nil {
		return "", fmt.Errorf("resolveCompanyCode(%q): %w", symbolOrCode, err)
	}
	if ov.Company.Code == "" {
		return "", fmt.Errorf("resolveCompanyCode(%q): overview returned empty company code", symbolOrCode)
	}
	return ov.Company.Code, nil
}
