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

// Suppress "imported and not used" errors while later tasks add GetTicks; remove later.
var _ = strconv.Itoa
var _ = url.Parse
