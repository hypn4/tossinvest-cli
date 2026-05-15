package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// StreamTicksOptions controls a long-running tick stream.
type StreamTicksOptions struct {
	Symbol   string
	Count    int           // request size each poll; default 50
	Interval time.Duration // poll cadence; default 2s
	Since    float64       // resume from this cumulativeVolume (exclusive); 0 emits the initial snapshot
	OnError  func(error)   // optional non-fatal error sink; default discards
}

// TickStream drives a tick poll loop and emits new ticks oldest-first.
type TickStream struct {
	client   *Client
	opts     StreamTicksOptions
	out      chan domain.Tick
	stopOnce sync.Once
	stop     chan struct{}
	cursor   float64
}

// StreamTicks builds a TickStream; call Run(ctx) to drive it.
func (c *Client) StreamTicks(opts StreamTicksOptions) (*TickStream, error) {
	if strings.TrimSpace(opts.Symbol) == "" {
		return nil, fmt.Errorf("StreamTicks: symbol is required")
	}
	if opts.Count <= 0 {
		opts.Count = 50
	}
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	return &TickStream{
		client: c,
		opts:   opts,
		out:    make(chan domain.Tick, opts.Count),
		stop:   make(chan struct{}),
		cursor: opts.Since,
	}, nil
}

// Ticks returns the channel callers consume.
func (s *TickStream) Ticks() <-chan domain.Tick { return s.out }

// Close stops the stream loop. Safe to call multiple times.
func (s *TickStream) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Run polls until ctx is cancelled or Close is called. It blocks; emit loops
// should run it in a goroutine.
func (s *TickStream) Run(ctx context.Context) error {
	defer close(s.out)
	timer := time.NewTimer(0) // fire immediately for first poll
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stop:
			return nil
		case <-timer.C:
			ticks, err := s.client.GetTicks(ctx, s.opts.Symbol, s.opts.Count)
			if err != nil {
				if s.opts.OnError != nil {
					s.opts.OnError(err)
				}
			} else {
				s.emit(ticks)
			}
			timer.Reset(s.opts.Interval)
		}
	}
}

// emit converts the newest-first snapshot into chronological NDJSON-friendly
// order and advances the cumulativeVolume cursor.
func (s *TickStream) emit(snapshot []domain.Tick) {
	// Snapshot is newest→oldest; iterate in reverse for chronological emit.
	for i := len(snapshot) - 1; i >= 0; i-- {
		tick := snapshot[i]
		if tick.CumulativeVolume <= s.cursor {
			continue
		}
		select {
		case s.out <- tick:
			s.cursor = tick.CumulativeVolume
		case <-s.stop:
			return
		}
	}
}

// StreamOrderBookOptions controls a long-running orderbook poll.
type StreamOrderBookOptions struct {
	Symbol   string
	Interval time.Duration // default 1s
	OnError  func(error)
}

// OrderBookStream drives an orderbook poll loop and emits a snapshot whenever
// the hash of (offerPrices, offerVolumes, bidPrices, bidVolumes) changes.
type OrderBookStream struct {
	client   *Client
	opts     StreamOrderBookOptions
	out      chan domain.OrderBook
	stopOnce sync.Once
	stop     chan struct{}
	lastHash uint64
	hasLast  bool
}

// StreamOrderBook builds an OrderBookStream; call Run(ctx) to drive it.
func (c *Client) StreamOrderBook(opts StreamOrderBookOptions) (*OrderBookStream, error) {
	if strings.TrimSpace(opts.Symbol) == "" {
		return nil, fmt.Errorf("StreamOrderBook: symbol is required")
	}
	if opts.Interval <= 0 {
		opts.Interval = 1 * time.Second
	}
	return &OrderBookStream{
		client: c,
		opts:   opts,
		out:    make(chan domain.OrderBook, 4),
		stop:   make(chan struct{}),
	}, nil
}

// Books returns the channel callers consume.
func (s *OrderBookStream) Books() <-chan domain.OrderBook { return s.out }

// Close stops the stream loop. Safe to call multiple times.
func (s *OrderBookStream) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Run polls until ctx is cancelled or Close is called.
func (s *OrderBookStream) Run(ctx context.Context) error {
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
			book, err := s.client.GetOrderBook(ctx, s.opts.Symbol)
			if err != nil {
				if s.opts.OnError != nil {
					s.opts.OnError(err)
				}
			} else {
				s.emit(book)
			}
			timer.Reset(s.opts.Interval)
		}
	}
}

// emit hashes the orderbook levels and emits the snapshot when the hash differs
// from the previous emission (or is the first emission).
func (s *OrderBookStream) emit(book domain.OrderBook) {
	h := hashOrderBook(book)
	if s.hasLast && h == s.lastHash {
		return
	}
	s.lastHash = h
	s.hasLast = true
	select {
	case s.out <- book:
	case <-s.stop:
	}
}

// hashOrderBook produces a stable hash of the price/volume vectors. FNV-1a 64
// keeps the implementation dependency-free; collisions are negligible at this
// payload size. Levels are sorted by price before hashing so the hash is
// invariant under any server-side reordering of the same book.
func hashOrderBook(book domain.OrderBook) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	hashLevels := func(levels []domain.OrderBookLevel) {
		sorted := append([]domain.OrderBookLevel(nil), levels...)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Price < sorted[j].Price
		})
		for _, lvl := range sorted {
			binary.BigEndian.PutUint64(buf[:], math.Float64bits(lvl.Price))
			h.Write(buf[:])
			binary.BigEndian.PutUint64(buf[:], math.Float64bits(lvl.Volume))
			h.Write(buf[:])
		}
		h.Write([]byte{0xff})
	}
	hashLevels(book.Offers)
	hashLevels(book.Bids)
	return h.Sum64()
}
