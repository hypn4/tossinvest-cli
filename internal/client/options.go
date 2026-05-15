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

type optionChainEnvelope struct {
	Result []struct {
		StrikePrice      float64 `json:"strikePrice"`
		CallGuid         string  `json:"callGuid"`
		PutGuid          string  `json:"putGuid"`
		CallOpenInterest int     `json:"callOpenInterest"`
		PutOpenInterest  int     `json:"putOpenInterest"`
	} `json:"result"`
}

// GetOptionChain returns the strike chain for an underlying's specific expiry.
// `maturityDate` must be in YYYY-MM-DD form.
func (c *Client) GetOptionChain(ctx context.Context, underlying, maturityDate string) ([]domain.OptionChainRow, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-both-chain/get-all", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	q.Set("maturityDate", maturityDate)
	endpoint.RawQuery = q.Encode()

	var env optionChainEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionChainRow, 0, len(env.Result))
	for _, r := range env.Result {
		out = append(out, domain.OptionChainRow{
			StrikePrice:      r.StrikePrice,
			CallGuid:         r.CallGuid,
			PutGuid:          r.PutGuid,
			CallOpenInterest: r.CallOpenInterest,
			PutOpenInterest:  r.PutOpenInterest,
		})
	}
	return out, nil
}

type optionExpiriesEnvelope struct {
	Result struct {
		Items []struct {
			MaturityDate               string `json:"maturityDate"`
			MaturityDateTime           string `json:"maturityDateTime"`
			LiquidationDateTime        string `json:"liquidationDateTime"`
			DisplayLiquidationDateTime string `json:"displayLiquidationDateTime"`
			CorporateActionDateTime    string `json:"corporateActionDateTime"`
			CorporateActionName        string `json:"corporateActionName"`
			DisplayCorporateActionName string `json:"displayCorporateActionName"`
		} `json:"items"`
	} `json:"result"`
}

// ListOptionExpiries returns the expiry ladder for an underlying.
// Accepts symbol or productCode.
func (c *Client) ListOptionExpiries(ctx context.Context, underlying string) ([]domain.OptionExpiry, error) {
	productCode, err := c.resolveProductCode(ctx, underlying)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/option-maturity-date/get-all", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("underlyingGuid", productCode)
	endpoint.RawQuery = q.Encode()

	var env optionExpiriesEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionExpiry, 0, len(env.Result.Items))
	for _, it := range env.Result.Items {
		out = append(out, domain.OptionExpiry{
			MaturityDate:               it.MaturityDate,
			MaturityDateTime:           it.MaturityDateTime,
			LiquidationDateTime:        it.LiquidationDateTime,
			DisplayLiquidationDateTime: it.DisplayLiquidationDateTime,
			CorporateActionDateTime:    it.CorporateActionDateTime,
			CorporateActionName:        it.CorporateActionName,
			DisplayCorporateActionName: it.DisplayCorporateActionName,
		})
	}
	return out, nil
}

type optionPricesEnvelope struct {
	Result struct {
		Prices []struct {
			Code            string  `json:"code"`
			Base            float64 `json:"base"`
			Close           float64 `json:"close"`
			ChangeType      string  `json:"changeType"`
			Currency        string  `json:"currency"`
			Volume          float64 `json:"volume"`
			BaseKrw         float64 `json:"baseKrw"`
			CloseKrw        float64 `json:"closeKrw"`
			BaseKrwDecimal  float64 `json:"baseKrwDecimal"`
			CloseKrwDecimal float64 `json:"closeKrwDecimal"`
		} `json:"prices"`
	} `json:"result"`
}

// optionPricesBatchSize caps codes per /api/v2/stock-prices call. Toss web
// batches ~58 codes; we use 50 to stay well under any URL-length / proxy
// limits (each OPT_ code is ~32 chars; 50 × 32 plus separators ≈ 1.6 KB).
const optionPricesBatchSize = 50

// GetOptionPrices fetches the bulk price list for a slice of productCodes
// (typically OPT_ codes but also works for stocks). Large code lists are
// chunked into batches of optionPricesBatchSize to stay within URL limits.
func (c *Client) GetOptionPrices(ctx context.Context, codes []string) ([]domain.OptionPrice, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("GetOptionPrices: codes is empty")
	}
	out := make([]domain.OptionPrice, 0, len(codes))
	for start := 0; start < len(codes); start += optionPricesBatchSize {
		end := start + optionPricesBatchSize
		if end > len(codes) {
			end = len(codes)
		}
		batch, err := c.getOptionPricesBatch(ctx, codes[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
	}
	return out, nil
}

func (c *Client) getOptionPricesBatch(ctx context.Context, codes []string) ([]domain.OptionPrice, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v2/stock-prices", c.infoBaseURL))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("codes", strings.Join(codes, ","))
	endpoint.RawQuery = q.Encode()

	var env optionPricesEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &env); err != nil {
		return nil, err
	}
	out := make([]domain.OptionPrice, 0, len(env.Result.Prices))
	for _, p := range env.Result.Prices {
		out = append(out, domain.OptionPrice{
			Code:            p.Code,
			Base:            p.Base,
			Close:           p.Close,
			ChangeType:      p.ChangeType,
			Currency:        p.Currency,
			Volume:          p.Volume,
			BaseKrw:         p.BaseKrw,
			CloseKrw:        p.CloseKrw,
			BaseKrwDecimal:  p.BaseKrwDecimal,
			CloseKrwDecimal: p.CloseKrwDecimal,
		})
	}
	return out, nil
}
