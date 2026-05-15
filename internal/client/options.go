package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type optionStockInfoEnvelope struct {
	Result struct {
		Code                         string `json:"code"`
		OptionPennyPilotPriceSupport bool   `json:"optionPennyPilotPriceSupported"`
		OptionInstrument             struct {
			MarketCode        string  `json:"marketCode"`
			RootSymbol        string  `json:"rootSymbol"`
			Name              string  `json:"name"`
			FullName          string  `json:"fullName"`
			CompleteName      string  `json:"completeName"`
			UnderlyingSymbol  string  `json:"underlyingSymbol"`
			UnderlyingGuid    string  `json:"underlyingGuid"`
			UnderlyingName    string  `json:"underlyingName"`
			MaturityDate      string  `json:"maturityDate"`
			MaturityDateTime  string  `json:"maturityDateTime"`
			PutCall           string  `json:"putCall"`
			StrikePrice       float64 `json:"strikePrice"`
			BasePrice         float64 `json:"basePrice"`
			Last              float64 `json:"last"`
			Bid               float64 `json:"bid"`
			Ask               float64 `json:"ask"`
			Mid               float64 `json:"mid"`
			ContractUnit      float64 `json:"contractUnit"`
			OpenInterest      int     `json:"openInterest"`
			Halted            bool    `json:"halted"`
			TradingSuspended  bool    `json:"tradingSuspended"`
			BuySuspended      bool    `json:"buySuspended"`
			SellSuspended     bool    `json:"sellSuspended"`
			Status            string  `json:"status"`
			Overtime          bool    `json:"overtime"`
			OptionLiquidation struct {
				LiquidationDateTime        string `json:"liquidationDateTime"`
				DisplayLiquidationDateTime string `json:"displayLiquidationDateTime"`
			} `json:"optionLiquidation"`
		} `json:"optionInstrument"`
	} `json:"result"`
}

// GetOptionInstrument decodes the option-specific metadata block from
// /api/v2/stock-infos/{OPT_…}. Caller must provide a full OPT_ productCode.
func (c *Client) GetOptionInstrument(ctx context.Context, productCode string) (domain.OptionInstrument, error) {
	if !strings.HasPrefix(productCode, "OPT_") {
		return domain.OptionInstrument{}, fmt.Errorf("GetOptionInstrument: productCode must start with OPT_ (got %q)", productCode)
	}
	endpoint := fmt.Sprintf("%s/api/v2/stock-infos/%s", c.infoBaseURL, productCode)
	var envelope optionStockInfoEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.OptionInstrument{}, err
	}
	oi := envelope.Result.OptionInstrument
	return domain.OptionInstrument{
		ProductCode:         productCode,
		MarketCode:          oi.MarketCode,
		RootSymbol:          oi.RootSymbol,
		Name:                oi.Name,
		FullName:            oi.FullName,
		CompleteName:        oi.CompleteName,
		UnderlyingSymbol:    oi.UnderlyingSymbol,
		UnderlyingGuid:      oi.UnderlyingGuid,
		UnderlyingName:      oi.UnderlyingName,
		MaturityDate:        oi.MaturityDate,
		MaturityDateTime:    oi.MaturityDateTime,
		PutCall:             oi.PutCall,
		StrikePrice:         oi.StrikePrice,
		BasePrice:           oi.BasePrice,
		Last:                oi.Last,
		Bid:                 oi.Bid,
		Ask:                 oi.Ask,
		Mid:                 oi.Mid,
		ContractUnit:        oi.ContractUnit,
		OpenInterest:        oi.OpenInterest,
		Halted:              oi.Halted,
		TradingSuspended:    oi.TradingSuspended,
		BuySuspended:        oi.BuySuspended,
		SellSuspended:       oi.SellSuspended,
		Status:              oi.Status,
		Overtime:            oi.Overtime,
		LiquidationDisplay:  oi.OptionLiquidation.DisplayLiquidationDateTime,
		LiquidationDateTime: oi.OptionLiquidation.LiquidationDateTime,
		PennyPilot:          envelope.Result.OptionPennyPilotPriceSupport,
		FetchedAt:           time.Now().UTC(),
	}, nil
}

// GetNearestATMOption returns the OPT_ productCode for the underlying's
// nearest-expiry at-the-money option. Accepts symbol or productCode.
func (c *Client) GetNearestATMOption(ctx context.Context, underlying string) (string, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-infos/default-chart-option", c.infoBaseURL))
	if err != nil {
		return "", err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	endpoint.RawQuery = q.Encode()

	var envelope struct {
		Result string `json:"result"`
	}
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return "", err
	}
	if envelope.Result == "" {
		return "", fmt.Errorf("GetNearestATMOption: empty result for %s", productCode)
	}
	return envelope.Result, nil
}
