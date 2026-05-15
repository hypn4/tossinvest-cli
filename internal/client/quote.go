package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type quoteEnvelope[T any] struct {
	Result T `json:"result"`
}

type stockInfoResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	Market   struct {
		Code        string `json:"code"`
		DisplayName string `json:"displayName"`
	} `json:"market"`
}

type stockDetailCommonResult struct {
	Badges  []json.RawMessage `json:"badges"`
	Notices []json.RawMessage `json:"notices"`
}

type stockPriceResult struct {
	ProductCode string  `json:"productCode"`
	Exchange    string  `json:"exchange"`
	Currency    string  `json:"currency"`
	Base        float64 `json:"base"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
}

type stockPriceDetailsResult struct {
	Code             string  `json:"code"`
	Exchange         string  `json:"exchange"`
	Currency         string  `json:"currency"`
	TradeDateTime    string  `json:"tradeDateTime"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	Volume           float64 `json:"volume"`
	Value            float64 `json:"value"`
	Base             float64 `json:"base"`
	High52W          float64 `json:"high52w"`
	Low52W           float64 `json:"low52w"`
	High1Y           float64 `json:"high1y"`
	Low1Y            float64 `json:"low1y"`
	MarketCap        float64 `json:"marketCap"`
	TradingStrength  float64 `json:"tradingStrength"`
	PreDayVolume     float64 `json:"preDayVolume"`
	UpperLimit       float64 `json:"upperLimit"`
	LowerLimit       float64 `json:"lowerLimit"`
	AfterMarketOpen  float64 `json:"afterMarketOpen"`
	AfterMarketHigh  float64 `json:"afterMarketHigh"`
	AfterMarketLow   float64 `json:"afterMarketLow"`
	AfterMarketClose float64 `json:"afterMarketClose"`
	CloseKRW         float64 `json:"closeKrw"`
	OpenKRW          float64 `json:"openKrw"`
	HighKRW          float64 `json:"highKrw"`
	LowKRW           float64 `json:"lowKrw"`
	BaseKRW          float64 `json:"baseKrw"`
	ValueKRW         float64 `json:"valueKrw"`
}

type stockHeaderSection struct {
	Type               string  `json:"type"`
	GrossExpenseRatio  float64 `json:"grossExpenseRatio,omitempty"`
	DividendYieldRatio float64 `json:"dividendYieldRatio,omitempty"`
	Ranking            int     `json:"ranking,omitempty"`
	TradingStrength    float64 `json:"tradingStrength,omitempty"`
}

type stockHeaderResult struct {
	Sections []stockHeaderSection `json:"sections"`
}

type stockSearchEnvelope struct {
	Result struct {
		Stocks []struct {
			StockCode string `json:"stockCode"`
			StockName string `json:"stockName"`
			MatchType string `json:"matchType"`
		} `json:"stocks"`
	} `json:"result"`
}

func (c *Client) GetQuote(ctx context.Context, symbol string) (domain.Quote, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.Quote{}, err
	}

	info, err := c.getStockInfo(ctx, productCode)
	if err != nil {
		return domain.Quote{}, err
	}

	quote := domain.Quote{
		ProductCode: productCode,
		Symbol:      info.Symbol,
		Name:        info.Name,
		MarketCode:  info.Market.Code,
		Market:      info.Market.DisplayName,
		Currency:    info.Currency,
		Status:      info.Status,
		FetchedAt:   time.Now().UTC(),
	}

	if details, err := c.getStockPriceDetails(ctx, productCode); err == nil {
		applyPriceDetails(&quote, details)
	} else {
		price, err := c.getStockPrice(ctx, productCode)
		if err != nil {
			return domain.Quote{}, err
		}
		applyPriceFallback(&quote, price)
	}

	if header, err := c.getStockHeader(ctx, productCode); err == nil {
		applyStockHeader(&quote, header)
	}

	if detail, err := c.getStockDetailCommon(ctx, productCode); err == nil && detail != nil {
		quote.BadgeCount = len(detail.Badges)
		quote.NoticeCount = len(detail.Notices)
	}

	return quote, nil
}

func applyPriceDetails(q *domain.Quote, d stockPriceDetailsResult) {
	q.Currency = firstNonEmpty(d.Currency, q.Currency)
	q.ReferencePrice = d.Base
	q.Last = d.Close
	q.Change = d.Close - d.Base
	q.Volume = d.Volume
	q.Open = d.Open
	q.High = d.High
	q.Low = d.Low
	q.Value = d.Value
	q.High52W = d.High52W
	q.Low52W = d.Low52W
	q.High1Y = d.High1Y
	q.Low1Y = d.Low1Y
	q.MarketCap = d.MarketCap
	q.TradingStrength = d.TradingStrength
	q.PreDayVolume = d.PreDayVolume
	q.UpperLimit = d.UpperLimit
	q.LowerLimit = d.LowerLimit
	q.AfterMarketOpen = d.AfterMarketOpen
	q.AfterMarketHigh = d.AfterMarketHigh
	q.AfterMarketLow = d.AfterMarketLow
	q.AfterMarketClose = d.AfterMarketClose
	q.LastKRW = d.CloseKRW
	q.OpenKRW = d.OpenKRW
	q.HighKRW = d.HighKRW
	q.LowKRW = d.LowKRW
	q.ReferencePriceKRW = d.BaseKRW
	q.ValueKRW = d.ValueKRW

	if d.Base != 0 {
		q.ChangeRate = q.Change / d.Base
	}
}

func applyPriceFallback(q *domain.Quote, p stockPriceResult) {
	q.Currency = firstNonEmpty(p.Currency, q.Currency)
	q.ReferencePrice = p.Base
	q.Last = p.Close
	q.Change = p.Close - p.Base
	q.Volume = p.Volume
	if p.Base != 0 {
		q.ChangeRate = q.Change / p.Base
	}
}

func applyStockHeader(q *domain.Quote, header stockHeaderResult) {
	for _, section := range header.Sections {
		switch strings.ToUpper(section.Type) {
		case "ETF":
			q.GrossExpenseRatio = section.GrossExpenseRatio
			q.DividendYieldRate = section.DividendYieldRatio
		case "TRADING_AMOUNT":
			q.TradingAmountRank = section.Ranking
		case "TRADING_STRENGTH":
			if section.TradingStrength != 0 && q.TradingStrength == 0 {
				q.TradingStrength = section.TradingStrength
			}
		}
	}
	if q.MarketCap > 0 && q.MarketCapKRW == 0 {
		// Server response sometimes omits MarketCapKRW; downstream KRW columns are
		// best-effort. Leave zero rather than recomputing here.
		q.MarketCapKRW = 0
	}
}

func (c *Client) getStockPriceDetails(ctx context.Context, productCode string) (stockPriceDetailsResult, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v3/stock-prices/details", c.infoBaseURL))
	if err != nil {
		return stockPriceDetailsResult{}, err
	}
	query := endpoint.Query()
	query.Set("productCodes", productCode)
	endpoint.RawQuery = query.Encode()

	var envelope quoteEnvelope[[]stockPriceDetailsResult]
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return stockPriceDetailsResult{}, err
	}
	if len(envelope.Result) == 0 {
		return stockPriceDetailsResult{}, fmt.Errorf("no price-details result for %s", productCode)
	}
	return envelope.Result[0], nil
}

func (c *Client) getStockHeader(ctx context.Context, productCode string) (stockHeaderResult, error) {
	var envelope quoteEnvelope[stockHeaderResult]
	if err := c.getJSON(
		ctx,
		fmt.Sprintf("%s/api/v1/stock-infos/header/%s", c.infoBaseURL, productCode),
		&envelope,
	); err != nil {
		return stockHeaderResult{}, err
	}
	return envelope.Result, nil
}

// ResolveProductCode resolves a user-supplied symbol (e.g. "SOXL", "A005930", or
// a productCode) to the canonical Toss productCode used by the trading endpoints.
// Thin exported wrapper over resolveProductCode for command-layer callers.
func (c *Client) ResolveProductCode(ctx context.Context, symbol string) (string, error) {
	return c.resolveProductCode(ctx, symbol)
}

func (c *Client) resolveProductCode(ctx context.Context, symbol string) (string, error) {
	normalized := normalizeProductCode(symbol)
	if normalized == "" {
		return "", fmt.Errorf("symbol is required")
	}
	if looksLikeProductCode(normalized) {
		return normalized, nil
	}

	var envelope stockSearchEnvelope
	body := []byte(fmt.Sprintf(`{"query":%q}`, normalized))
	if err := c.postJSON(ctx, fmt.Sprintf("%s/api/v2/search/stocks", c.infoBaseURL), body, &envelope); err != nil {
		return "", err
	}
	if len(envelope.Result.Stocks) == 0 {
		return "", fmt.Errorf("no product code result returned for %s", normalized)
	}
	return envelope.Result.Stocks[0].StockCode, nil
}

func (c *Client) getStockInfo(ctx context.Context, productCode string) (stockInfoResult, error) {
	var envelope quoteEnvelope[stockInfoResult]
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/v2/stock-infos/%s", c.infoBaseURL, productCode), &envelope); err != nil {
		return stockInfoResult{}, err
	}
	return envelope.Result, nil
}

func (c *Client) getStockDetailCommon(ctx context.Context, productCode string) (*stockDetailCommonResult, error) {
	var envelope quoteEnvelope[stockDetailCommonResult]
	if err := c.getJSON(
		ctx,
		fmt.Sprintf("%s/api/v1/stock-detail/ui/%s/common", c.infoBaseURL, productCode),
		&envelope,
	); err != nil {
		return nil, err
	}
	return &envelope.Result, nil
}

func (c *Client) getStockPrice(ctx context.Context, productCode string) (stockPriceResult, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v1/product/stock-prices", c.infoBaseURL))
	if err != nil {
		return stockPriceResult{}, err
	}

	query := endpoint.Query()
	query.Set("meta", "true")
	query.Set("productCodes", productCode)
	endpoint.RawQuery = query.Encode()

	var envelope quoteEnvelope[[]stockPriceResult]
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return stockPriceResult{}, err
	}

	if len(envelope.Result) == 0 {
		return stockPriceResult{}, fmt.Errorf("no price result returned for %s", productCode)
	}

	return envelope.Result[0], nil
}

func normalizeProductCode(symbol string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(symbol))
	if trimmed == "" {
		return trimmed
	}

	if len(trimmed) == 6 && trimmed[0] >= '0' && trimmed[0] <= '9' {
		return "A" + trimmed
	}

	return trimmed
}

func looksLikeProductCode(value string) bool {
	if strings.HasPrefix(value, "OPT_") {
		return true
	}
	if len(value) == 7 && value[0] == 'A' {
		return true
	}
	// 3-letter market prefix + digits: NAS0250224006, AMX0260127004, NYS…
	if len(value) >= 4 && isAlpha(value[0]) && isAlpha(value[1]) && isAlpha(value[2]) {
		for i := 3; i < len(value); i++ {
			if value[i] < '0' || value[i] > '9' {
				return false
			}
		}
		return true
	}
	// 2-letter prefix + digits: US20100311002
	if len(value) >= 8 && isAlpha(value[0]) && isAlpha(value[1]) {
		hasDigit := false
		for i := 2; i < len(value); i++ {
			if value[i] >= '0' && value[i] <= '9' {
				hasDigit = true
				continue
			}
			return false
		}
		return hasDigit
	}
	return false
}

func isAlpha(b byte) bool { return b >= 'A' && b <= 'Z' }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// StreamQuoteOptions controls a long-running quote poll.
type StreamQuoteOptions struct {
	Symbol   string
	Interval time.Duration // default 3s
	OnError  func(error)
}

// QuoteStream drives a quote poll loop and emits a snapshot whenever the
// (last, volume) tuple changes. FetchedAt is a client-side timestamp and is
// not part of the dedup key.
type QuoteStream struct {
	client    *Client
	opts      StreamQuoteOptions
	out       chan domain.Quote
	stopOnce  sync.Once
	stop      chan struct{}
	lastPrice float64
	lastVol   float64
	hasLast   bool
}

// StreamQuote builds a QuoteStream; call Run(ctx) to drive it.
func (c *Client) StreamQuote(opts StreamQuoteOptions) (*QuoteStream, error) {
	if strings.TrimSpace(opts.Symbol) == "" {
		return nil, fmt.Errorf("StreamQuote: symbol is required")
	}
	if opts.Interval <= 0 {
		opts.Interval = 3 * time.Second
	}
	return &QuoteStream{
		client: c,
		opts:   opts,
		out:    make(chan domain.Quote, 4),
		stop:   make(chan struct{}),
	}, nil
}

// Quotes returns the channel callers consume.
func (s *QuoteStream) Quotes() <-chan domain.Quote { return s.out }

// Close stops the stream loop. Safe to call multiple times.
func (s *QuoteStream) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Run polls until ctx is cancelled or Close is called.
func (s *QuoteStream) Run(ctx context.Context) error {
	defer close(s.out)
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stop:
			return nil
		case <-timer.C:
			q, err := s.client.GetQuote(ctx, s.opts.Symbol)
			if err != nil {
				if s.opts.OnError != nil {
					s.opts.OnError(err)
				}
			} else {
				s.emit(q)
			}
			timer.Reset(s.opts.Interval)
		}
	}
}

// emit emits the snapshot when (last, volume) differs from the previous
// emission (or is the first emission).
func (s *QuoteStream) emit(q domain.Quote) {
	if s.hasLast && q.Last == s.lastPrice && q.Volume == s.lastVol {
		return
	}
	s.lastPrice = q.Last
	s.lastVol = q.Volume
	s.hasLast = true
	select {
	case s.out <- q:
	case <-s.stop:
	}
}
