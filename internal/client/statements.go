package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type statementRecordsEnvelope struct {
	Result struct {
		SelectedFactor struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedFactor"`
		SelectedPeriod struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
		} `json:"selectedPeriod"`
		IsKr  bool `json:"isKr"`
		Table []struct {
			Period string `json:"period"`
			Value  []struct {
				Item        string   `json:"item"`
				ParentItem  string   `json:"parentItem"`
				ItemNameKor string   `json:"itemNameKor"`
				ItemNameEng string   `json:"itemNameEng"`
				UnitType    string   `json:"unitType"`
				Value       *float64 `json:"value"`
			} `json:"value"`
		} `json:"table"`
	} `json:"result"`
}

// GetStockStatements fetches the financial-statement-records payload for a
// chosen factor (BAL|INC|CAS) and period (Q|Y). Returns the pivoted view.
func (c *Client) GetStockStatements(ctx context.Context, symbol, factorCode, period string) (domain.StockStatements, error) {
	factorCode = strings.ToUpper(factorCode)
	if factorCode != "BAL" && factorCode != "INC" && factorCode != "CAS" {
		return domain.StockStatements{}, fmt.Errorf("GetStockStatements: factorCode must be one of BAL|INC|CAS (got %q)", factorCode)
	}
	period = strings.ToUpper(period)
	if period != "Q" && period != "Y" {
		return domain.StockStatements{}, fmt.Errorf("GetStockStatements: period must be Q or Y (got %q)", period)
	}

	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.StockStatements{}, err
	}

	endpoint := fmt.Sprintf("%s/api/v2/companies/%s/financial-statement-records", c.infoBaseURL, productCode)
	selectorBody := map[string]string{"factorCode": factorCode, "period": period}
	bodyBytes, err := json.Marshal(selectorBody)
	if err != nil {
		return domain.StockStatements{}, err
	}
	var env statementRecordsEnvelope
	if err := c.postJSON(ctx, endpoint, json.RawMessage(bodyBytes), &env); err != nil {
		return domain.StockStatements{}, err
	}

	periods := make([]domain.StatementPeriod, len(env.Result.Table))
	for i, p := range env.Result.Table {
		items := make([]domain.StatementLineItem, len(p.Value))
		for j, v := range p.Value {
			items[j] = domain.StatementLineItem{
				Item:       v.Item,
				ParentItem: v.ParentItem,
				NameKor:    v.ItemNameKor,
				NameEng:    v.ItemNameEng,
				Unit:       v.UnitType,
				Value:      v.Value,
			}
		}
		periods[i] = domain.StatementPeriod{Period: p.Period, Items: items}
	}

	return domain.StockStatements{
		ProductCode: productCode,
		Factor: domain.StatementFactor{
			Code:        env.Result.SelectedFactor.Code,
			DisplayName: env.Result.SelectedFactor.DisplayName,
		},
		Period:    env.Result.SelectedPeriod.Code,
		IsKr:      env.Result.IsKr,
		Periods:   periods,
		FetchedAt: time.Now().UTC(),
	}, nil
}
