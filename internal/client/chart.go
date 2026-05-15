package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// supportedSteps enumerates the (unit, step) pairs that the Toss
// `/api/v1/c-chart/{product}/{code}/{unit}:{step}` endpoint accepts. The list
// was verified by enumerating 90 combinations on 2026-05-15; anything else
// returns 400. See docs/reverse-engineering/order-page-deep-dive.md.
var supportedSteps = map[string]map[int]struct{}{
	"min":   {1: {}, 3: {}, 5: {}, 10: {}, 15: {}, 30: {}, 60: {}},
	"day":   {1: {}},
	"week":  {1: {}},
	"month": {1: {}, 3: {}},
	"year":  {1: {}},
}

var supportedSessions = map[string]struct{}{
	"all":  {},
	"main": {},
	"day":  {},
	"pre":  {},
	"after": {},
}

var supportedInvestModes = map[string]struct{}{
	"integrated": {},
	"regular":    {},
}

// timeframeAliases maps user-friendly tags to the (unit, step) the API uses.
// `1h` collapses to `min:60` since Toss rejects `hour:*` outright.
var timeframeAliases = map[string]struct {
	Unit string
	Step int
}{
	"1m":  {"min", 1},
	"3m":  {"min", 3},
	"5m":  {"min", 5},
	"10m": {"min", 10},
	"15m": {"min", 15},
	"30m": {"min", 30},
	"60m": {"min", 60},
	"1h":  {"min", 60},
	"1d":  {"day", 1},
	"1w":  {"week", 1},
	"1mo": {"month", 1},
	"3mo": {"month", 3},
	"1y":  {"year", 1},
}

// ChartOptions controls a single chart fetch.
type ChartOptions struct {
	Timeframe   string    // user-friendly alias (e.g. "30m"); takes precedence over Unit/Step when set
	Unit        string    // raw "min" / "day" / "week" / "month" / "year"
	Step        int       // step within the unit; must be a supported value
	Count       int       // number of candles to return; defaults to 100
	From        time.Time // optional pagination cursor (older-than)
	Session     string    // "all" (default) / "main" / "day" / "pre" / "after"
	InvestMode  string    // "integrated" (default) / "regular"
	UseAdjusted *bool     // defaults to true
}

// ResolveTimeframe normalizes ChartOptions into (unit, step) and validates.
// Exported so the CLI layer can present a clear error before issuing a fetch.
func (o ChartOptions) ResolveTimeframe() (string, int, error) {
	unit := strings.ToLower(strings.TrimSpace(o.Unit))
	step := o.Step
	if tf := strings.ToLower(strings.TrimSpace(o.Timeframe)); tf != "" {
		alias, ok := timeframeAliases[tf]
		if !ok {
			return "", 0, fmt.Errorf("unsupported timeframe %q (try 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y)", tf)
		}
		unit, step = alias.Unit, alias.Step
	}
	if unit == "" {
		unit, step = "day", 1
	}
	stepsForUnit, ok := supportedSteps[unit]
	if !ok {
		return "", 0, fmt.Errorf("unsupported unit %q (must be one of min/day/week/month/year)", unit)
	}
	if _, ok := stepsForUnit[step]; !ok {
		return "", 0, fmt.Errorf("unsupported step %d for unit %q", step, unit)
	}
	return unit, step, nil
}

func validateSession(value string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "", nil
	}
	if _, ok := supportedSessions[v]; !ok {
		return "", fmt.Errorf("unsupported session %q (must be one of all/main/day/pre/after)", v)
	}
	return v, nil
}

func validateInvestMode(value string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "", nil
	}
	if _, ok := supportedInvestModes[v]; !ok {
		return "", fmt.Errorf("unsupported invest mode %q (must be integrated or regular)", v)
	}
	return v, nil
}

type chartEnvelope struct {
	Result struct {
		Code         string    `json:"code"`
		NextDateTime string    `json:"nextDateTime"`
		ExchangeRate float64   `json:"exchangeRate"`
		Candles      []rawCandle `json:"candles"`
	} `json:"result"`
}

type rawCandle struct {
	DT          string  `json:"dt"`
	SessionType string  `json:"sessionType,omitempty"`
	Base        float64 `json:"base"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume"`
	Amount      float64 `json:"amount"`
}

// GetChart fetches OHLCV candles for the resolved product.
func (c *Client) GetChart(ctx context.Context, symbol string, opts ChartOptions) (domain.Chart, error) {
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return domain.Chart{}, err
	}

	unit, step, err := opts.ResolveTimeframe()
	if err != nil {
		return domain.Chart{}, err
	}
	session, err := validateSession(opts.Session)
	if err != nil {
		return domain.Chart{}, err
	}
	investMode, err := validateInvestMode(opts.InvestMode)
	if err != nil {
		return domain.Chart{}, err
	}

	count := opts.Count
	if count <= 0 {
		count = 100
	}

	info, _ := c.getStockInfo(ctx, productCode)
	product := chartProductPrefix(info.Market.Code, productCode)

	endpoint, err := url.Parse(fmt.Sprintf(
		"%s/api/v1/c-chart/%s/%s/%s:%d",
		c.infoBaseURL, product, productCode, unit, step,
	))
	if err != nil {
		return domain.Chart{}, err
	}
	q := endpoint.Query()
	q.Set("count", strconv.Itoa(count))
	useAdjusted := true
	if opts.UseAdjusted != nil {
		useAdjusted = *opts.UseAdjusted
	}
	q.Set("useAdjustedRate", strconv.FormatBool(useAdjusted))
	if !opts.From.IsZero() {
		q.Set("from", opts.From.Format(time.RFC3339))
	}
	if session != "" {
		q.Set("session", session)
	}
	if investMode != "" {
		q.Set("investMode", investMode)
	}
	endpoint.RawQuery = q.Encode()

	var envelope chartEnvelope
	if err := c.getJSON(ctx, endpoint.String(), &envelope); err != nil {
		return domain.Chart{}, err
	}

	candles := make([]domain.Candle, 0, len(envelope.Result.Candles))
	for _, raw := range envelope.Result.Candles {
		candles = append(candles, domain.Candle{
			DateTime:    raw.DT,
			SessionType: raw.SessionType,
			Base:        raw.Base,
			Open:        raw.Open,
			High:        raw.High,
			Low:         raw.Low,
			Close:       raw.Close,
			Volume:      raw.Volume,
			Amount:      raw.Amount,
		})
	}

	return domain.Chart{
		ProductCode:  envelope.Result.Code,
		Symbol:       info.Symbol,
		Name:         info.Name,
		Market:       info.Market.DisplayName,
		Currency:     info.Currency,
		Unit:         unit,
		Step:         step,
		Session:      session,
		InvestMode:   investMode,
		UseAdjusted:  useAdjusted,
		ExchangeRate: envelope.Result.ExchangeRate,
		NextDateTime: envelope.Result.NextDateTime,
		Candles:      candles,
		FetchedAt:    time.Now().UTC(),
	}, nil
}

// chartProductPrefix maps a market code to the c-chart path prefix.
// Toss uses `kr-s` / `us-s` / `amx-s` / `nas-s` / `nys-s` etc. — the lower-case
// first two letters of the market code followed by `-s` for stocks/ETFs.
func chartProductPrefix(marketCode, productCode string) string {
	if strings.HasPrefix(productCode, "OPT_") {
		return "us-o"
	}
	mc := strings.ToLower(strings.TrimSpace(marketCode))
	switch mc {
	case "ksp", "ksq", "krx", "knx":
		return "kr-s"
	case "nys", "nas", "amx":
		return "us-s"
	}
	// Fallback: infer from the product code prefix.
	if strings.HasPrefix(productCode, "A") {
		return "kr-s"
	}
	return "us-s"
}

// StreamChartOptions controls a long-running chart poll.
type StreamChartOptions struct {
	Symbol    string
	Timeframe string        // alias: 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y
	Count     int           // candles to request per poll; default 2 (current + previous bucket)
	Interval  time.Duration // poll cadence; default 60s
	Session   string        // optional session filter
	OnError   func(error)   // non-fatal error sink
}

// ChartStream drives a chart poll loop and emits new or updated candles
// oldest-first. Dedup state: per-dt (close, volume); identical (dt,close,volume)
// across polls produces no emit.
type ChartStream struct {
	client   *Client
	opts     StreamChartOptions
	out      chan domain.Candle
	stopOnce sync.Once
	stop     chan struct{}
	seen     map[string]candleHash
}

type candleHash struct {
	close  float64
	volume float64
}

const chartSeenCap = 32

// StreamChart builds a ChartStream; call Run(ctx) to drive it.
func (c *Client) StreamChart(opts StreamChartOptions) (*ChartStream, error) {
	if strings.TrimSpace(opts.Symbol) == "" {
		return nil, fmt.Errorf("StreamChart: symbol is required")
	}
	if opts.Count <= 0 {
		opts.Count = 2
	}
	if opts.Interval <= 0 {
		opts.Interval = 60 * time.Second
	}
	return &ChartStream{
		client: c,
		opts:   opts,
		out:    make(chan domain.Candle, opts.Count*4),
		stop:   make(chan struct{}),
		seen:   make(map[string]candleHash, chartSeenCap),
	}, nil
}

// Ticks returns the channel callers consume. Method name kept symmetrical with
// TickStream so consumer patterns transfer; the channel element type signals
// what's actually flowing.
func (s *ChartStream) Ticks() <-chan domain.Candle { return s.out }

// Close stops the stream loop. Safe to call multiple times.
func (s *ChartStream) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Run polls until ctx is cancelled or Close is called.
func (s *ChartStream) Run(ctx context.Context) error {
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
			chart, err := s.client.GetChart(ctx, s.opts.Symbol, ChartOptions{
				Timeframe: s.opts.Timeframe,
				Count:     s.opts.Count,
				Session:   s.opts.Session,
			})
			if err != nil {
				if s.opts.OnError != nil {
					s.opts.OnError(err)
				}
			} else {
				s.emit(chart)
			}
			timer.Reset(s.opts.Interval)
		}
	}
}

// emit iterates candles oldest-first and emits any whose (dt, close, volume)
// hash differs from the previously-seen value (or is new). The seen map is
// pruned to the current snapshot whenever it grows beyond chartSeenCap.
func (s *ChartStream) emit(chart domain.Chart) {
	for i := len(chart.Candles) - 1; i >= 0; i-- {
		c := chart.Candles[i]
		h := candleHash{close: c.Close, volume: c.Volume}
		if prev, ok := s.seen[c.DateTime]; ok && prev == h {
			continue
		}
		s.seen[c.DateTime] = h
		select {
		case s.out <- c:
		case <-s.stop:
			return
		}
	}
	if len(s.seen) > chartSeenCap {
		keep := make(map[string]bool, len(chart.Candles))
		for _, c := range chart.Candles {
			keep[c.DateTime] = true
		}
		for dt := range s.seen {
			if !keep[dt] {
				delete(s.seen, dt)
			}
		}
	}
}
