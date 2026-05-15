package client

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type orderBookEnvelope struct {
	Result struct {
		Close           float64   `json:"close"`
		CloseKrw        float64   `json:"closeKrw"`
		OfferPrices     []float64 `json:"offerPrices"`
		OfferPricesKrw  []float64 `json:"offerPricesKrw"`
		OfferVolumes    []float64 `json:"offerVolumes"`
		BidPrices       []float64 `json:"bidPrices"`
		BidPricesKrw    []float64 `json:"bidPricesKrw"`
		BidVolumes      []float64 `json:"bidVolumes"`
		OfferVolume     float64   `json:"offerVolume"`
		BidVolume       float64   `json:"bidVolume"`
		SinglePrice     bool      `json:"singlePrice"`
		EstimatedPrice  float64   `json:"estimatedPrice"`
		EstimatedVolume float64   `json:"estimatedVolume"`
	} `json:"result"`
}

// GetOrderBook fetches the latest orderbook snapshot for a symbol. KR markets
// return up to 10 levels per side; US returns only top-of-book.
func (c *Client) GetOrderBook(ctx context.Context, symbol string) (domain.OrderBook, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.OrderBook{}, err
	}

	info, _ := c.getStockInfo(ctx, productCode)

	endpoint := fmt.Sprintf("%s/api/v3/stock-prices/%s/quotes", c.infoBaseURL, productCode)
	var envelope orderBookEnvelope
	if err := c.getJSON(ctx, endpoint, &envelope); err != nil {
		return domain.OrderBook{}, err
	}

	offers := assembleLevels(envelope.Result.OfferPrices, envelope.Result.OfferPricesKrw, envelope.Result.OfferVolumes)
	bids := assembleLevels(envelope.Result.BidPrices, envelope.Result.BidPricesKrw, envelope.Result.BidVolumes)

	// Offers come back highest→lowest; normalize to lowest→highest for table rendering.
	sort.SliceStable(offers, func(i, j int) bool { return offers[i].Price < offers[j].Price })
	// Bids come back highest→lowest already, which matches our presentation expectation.
	sort.SliceStable(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })

	return domain.OrderBook{
		ProductCode:     productCode,
		Symbol:          info.Symbol,
		Name:            info.Name,
		Market:          info.Market.DisplayName,
		Currency:        info.Currency,
		Last:            envelope.Result.Close,
		LastKRW:         envelope.Result.CloseKrw,
		Offers:          offers,
		Bids:            bids,
		OfferVolumeSum:  envelope.Result.OfferVolume,
		BidVolumeSum:    envelope.Result.BidVolume,
		SinglePrice:     envelope.Result.SinglePrice,
		EstimatedPrice:  envelope.Result.EstimatedPrice,
		EstimatedVolume: envelope.Result.EstimatedVolume,
		FetchedAt:       time.Now().UTC(),
	}, nil
}

func assembleLevels(prices, pricesKrw, volumes []float64) []domain.OrderBookLevel {
	n := len(prices)
	out := make([]domain.OrderBookLevel, 0, n)
	for i := 0; i < n; i++ {
		level := domain.OrderBookLevel{Price: prices[i]}
		if i < len(pricesKrw) {
			level.PriceKRW = pricesKrw[i]
		}
		if i < len(volumes) {
			level.Volume = volumes[i]
		}
		out = append(out, level)
	}
	return out
}

type tickEnvelope struct {
	Result []struct {
		Time             string  `json:"time"`
		Code             string  `json:"code"`
		Price            float64 `json:"price"`
		PriceKrw         float64 `json:"priceKrw"`
		Base             float64 `json:"base"`
		BaseKrw          float64 `json:"baseKrw"`
		Volume           float64 `json:"volume"`
		TradeType        string  `json:"tradeType"`
		CumulativeVolume float64 `json:"cumulativeVolume"`
	} `json:"result"`
}

// GetTicks returns up to count recent trade ticks newest-first.
func (c *Client) GetTicks(ctx context.Context, symbol string, count int) ([]domain.Tick, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 50
	}

	endpoint, err := url.Parse(fmt.Sprintf("%s/api/v2/stock-prices/%s/ticks", c.infoBaseURL, productCode))
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("count", strconv.Itoa(count))
	endpoint.RawQuery = q.Encode()

	var envelope tickEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.Tick, 0, len(envelope.Result))
	fetchedAt := time.Now().UTC()
	for _, raw := range envelope.Result {
		out = append(out, domain.Tick{
			Time:             raw.Time,
			ProductCode:      raw.Code,
			Price:            raw.Price,
			PriceKRW:         raw.PriceKrw,
			Base:             raw.Base,
			Volume:           raw.Volume,
			TradeType:        raw.TradeType,
			CumulativeVolume: raw.CumulativeVolume,
			FetchedAt:        fetchedAt,
		})
	}
	return out, nil
}
