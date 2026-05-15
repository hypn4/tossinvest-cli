# Follow-Streaming (Chart / Orderbook / Quote) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--follow` NDJSON streaming to `tossctl chart get`, `tossctl quotes book`, and `tossctl quote get`, matching the polling cadences identified in the Toss web reverse engineering.

**Architecture:** Three concrete `Stream*` types (`ChartStream`, `OrderBookStream`, `QuoteStream`) mirroring the existing `TickStream` (`internal/client/quotes.go`) line-for-line — **no generic `Stream[T]` abstraction**. Each owns its own dedup state because the dedup keys differ structurally:

| Command | Dedup state | Emit condition |
| --- | --- | --- |
| `chart get --follow` | `map[dt] -> {close, volume}` (capped at 32) | new `dt` OR same `dt` with different `(close, volume)` |
| `quotes book --follow` | last hash of `(offerPrices, offerVolumes, bidPrices, bidVolumes)` | hash changes (emit whole snapshot) |
| `quote get --follow` | last `(last, volume, tradeDateTime)` | any of those three changes |

Each stream gets a sibling NDJSON writer (`WriteCandleNDJSON`, `WriteOrderBookNDJSON`, `WriteQuoteNDJSON`) parallel to `WriteTickNDJSON`. All defaults anchor to `docs/reverse-engineering/rpc-catalog.md:148-161`:

```
chart --follow:   --interval 60s   --count 2
quotes book:      --interval 1s
quote get:        --interval 3s
```

**Tech Stack:** Go, cobra (no new dependencies). Reuses `Client.getJSON`, `signal.NotifyContext`, `time.NewTimer` from existing stream pattern.

---

## Reference

- Existing pattern: `internal/client/quotes.go:148-238` (`StreamTicks` / `TickStream`)
- CLI follow wiring: `cmd/tossctl/quotes.go:88-126` (`runTicksFollow`)
- NDJSON writer: `internal/output/quotes.go` (`WriteTickNDJSON`)
- Realtime cadences: `docs/reverse-engineering/rpc-catalog.md:148-161` (Realtime Strategy)
- Domain types: `internal/domain/models.go` — `Candle`, `Chart`, `OrderBook`, `Quote`

Key constraints the implementation must respect:

1. **Toss does not push prices over SSE/WebSocket.** REST polling is the only mechanism. Document this in `--help` long text so users don't expect lower latency than the poll interval.
2. **Market-closed silence is correct behavior.** When the market is closed, dedup keys stop changing and the stream emits nothing. Do **not** add heartbeats — they pollute NDJSON. Note this in `--help`.
3. **All `--follow` modes emit oldest-first NDJSON** so consumers can `tail -f` / pipe to `jq` and see chronological updates.
4. **No generic `Stream[T]` abstraction.** Three concrete types. The reviewer agent may be tempted to factor out a base — preempt with this self-review note.
5. **Tests must NOT rely on real network.** Use `httptest.NewServer` with a programmatic mutator between calls (increment a counter, mutate one field, re-serve).
6. **Fork-only.** Branch `feat/pr5-follow-streaming` is off `feat/order-page-integration`; push to `origin` (fork) only — never `upstream`.

---

## Task 1: Output — `WriteCandleNDJSON`

**Files:**
- Modify: `internal/output/chart.go` (add function at end of file)
- Modify: `internal/output/chart_test.go` (add test)

- [ ] **Step 1: Write the failing test**

Append to `internal/output/chart_test.go`:

```go
func TestWriteCandleNDJSON(t *testing.T) {
	var buf bytes.Buffer
	candles := []domain.Candle{
		{DateTime: "2026-05-15T10:00:00-04:00", Open: 100, High: 102, Low: 99, Close: 101, Volume: 1000},
		{DateTime: "2026-05-15T10:30:00-04:00", Open: 101, High: 103, Low: 100, Close: 102, Volume: 1200},
	}
	for _, c := range candles {
		if err := WriteCandleNDJSON(&buf, c); err != nil {
			t.Fatalf("error: %v", err)
		}
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %q", len(lines), buf.String())
	}
	var first domain.Candle
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("first line not valid JSON: %v", err)
	}
	if first.Close != 101 {
		t.Fatalf("unexpected first candle: %+v", first)
	}
}
```

If the test file doesn't yet import `bytes`, `strings`, `encoding/json`, add them.

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteCandleNDJSON -v`
Expected: `undefined: WriteCandleNDJSON`.

- [ ] **Step 3: Implement `WriteCandleNDJSON`**

Append to `internal/output/chart.go`:

```go
// WriteCandleNDJSON writes a single candle as one JSON object terminated by '\n'.
// Used by `tossctl chart get --follow` so downstream consumers (jq, LLM
// pipelines) can read each candle update as a stream of independent objects.
func WriteCandleNDJSON(w io.Writer, candle domain.Candle) error {
	data, err := json.Marshal(candle)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
```

If `encoding/json` / `io` are not yet imported in `chart.go`, add them.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/output/ -run TestWriteCandleNDJSON -v`
Expected: PASS.

- [ ] **Step 5: Run the whole output suite**

Run: `go test ./internal/output/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/output/chart.go internal/output/chart_test.go
git commit -m "feat(output): add WriteCandleNDJSON for chart --follow"
```

---

## Task 2: Output — `WriteOrderBookNDJSON`

**Files:**
- Modify: `internal/output/quotes.go`
- Modify: `internal/output/quotes_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/output/quotes_test.go`:

```go
func TestWriteOrderBookNDJSON(t *testing.T) {
	var buf bytes.Buffer
	book := domain.OrderBook{
		ProductCode: "US20100311002",
		Last:        167.10,
		Offers:      []domain.OrderBookLevel{{Price: 167.20, Volume: 30}},
		Bids:        []domain.OrderBookLevel{{Price: 167.05, Volume: 87}},
	}
	if err := WriteOrderBookNDJSON(&buf, book); err != nil {
		t.Fatalf("error: %v", err)
	}
	line := strings.TrimRight(buf.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Fatalf("expected single-line NDJSON, got: %q", buf.String())
	}
	var parsed domain.OrderBook
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed.ProductCode != "US20100311002" {
		t.Fatalf("unexpected book: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteOrderBookNDJSON -v`
Expected: `undefined: WriteOrderBookNDJSON`.

- [ ] **Step 3: Implement `WriteOrderBookNDJSON`**

Append to `internal/output/quotes.go`:

```go
// WriteOrderBookNDJSON writes a single orderbook snapshot as one JSON object
// terminated by '\n'. Used by `tossctl quotes book --follow`.
func WriteOrderBookNDJSON(w io.Writer, book domain.OrderBook) error {
	data, err := json.Marshal(book)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
```

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/output/ -run TestWriteOrderBookNDJSON -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output/quotes.go internal/output/quotes_test.go
git commit -m "feat(output): add WriteOrderBookNDJSON for quotes book --follow"
```

---

## Task 3: Output — `WriteQuoteNDJSON`

**Files:**
- Modify: `internal/output/quote.go`
- Modify: `internal/output/quote_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/output/quote_test.go` (create the file if it does not exist; add `package output` and required imports `bytes`, `encoding/json`, `strings`, `testing`, the domain package):

```go
func TestWriteQuoteNDJSON(t *testing.T) {
	var buf bytes.Buffer
	q := domain.Quote{
		ProductCode: "US20100311002",
		Symbol:      "SOXL",
		Last:        167.10,
		Volume:      1500000,
	}
	if err := WriteQuoteNDJSON(&buf, q); err != nil {
		t.Fatalf("error: %v", err)
	}
	line := strings.TrimRight(buf.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Fatalf("expected single line, got %q", buf.String())
	}
	var parsed domain.Quote
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed.Symbol != "SOXL" {
		t.Fatalf("unexpected quote: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/output/ -run TestWriteQuoteNDJSON -v`
Expected: `undefined: WriteQuoteNDJSON`.

- [ ] **Step 3: Implement `WriteQuoteNDJSON`**

Append to `internal/output/quote.go`:

```go
// WriteQuoteNDJSON writes a single quote snapshot as one JSON object terminated
// by '\n'. Used by `tossctl quote get --follow`.
func WriteQuoteNDJSON(w io.Writer, quote domain.Quote) error {
	data, err := json.Marshal(quote)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}
```

If `encoding/json` / `io` are not yet imported, add them.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/output/ -run TestWriteQuoteNDJSON -v`
Expected: PASS.

- [ ] **Step 5: Run all output tests**

Run: `go test ./internal/output/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/output/quote.go internal/output/quote_test.go
git commit -m "feat(output): add WriteQuoteNDJSON for quote get --follow"
```

---

## Task 4: Client — `ChartStream` + `StreamChart`

**Files:**
- Modify: `internal/client/chart.go`
- Modify: `internal/client/chart_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Append to `internal/client/chart_test.go`. The handler increments `calls` and mutates the fixture body between requests to exercise the three dedup branches: initial emit, same-dt intra-bucket update, new-dt bucket.

```go
func TestStreamChartEmitsIntraBucketAndNewBucket(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	stockInfo := mustReadFile(t, filepath.Join(root, "stock-info.json"))

	// Three synthetic chart responses delivered round-robin by call count.
	bodies := [][]byte{
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":102,"low":99,"close":101,"volume":1000}
		]}}`),
		// same dt, close & volume changed -> intra-bucket update, must emit
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":103,"low":99,"close":102,"volume":1500}
		]}}`),
		// new dt prepended (newest-first), prior bucket unchanged -> emit only the new one
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:30:00+09:00","open":102,"high":104,"low":101,"close":103,"volume":800},
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":103,"low":99,"close":102,"volume":1500}
		]}}`),
		// identical to call 3 -> emit nothing
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:30:00+09:00","open":102,"high":104,"low":101,"close":103,"volume":800},
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":103,"low":99,"close":102,"volume":1500}
		]}}`),
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v2/stock-infos/"):
			w.Write(stockInfo)
		case strings.Contains(r.URL.Path, "/api/v1/c-chart/"):
			idx := int(calls.Add(1)) - 1
			if idx >= len(bodies) {
				idx = len(bodies) - 1
			}
			w.Write(bodies[idx])
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	stream, err := c.StreamChart(StreamChartOptions{
		Symbol:    "A005930",
		Timeframe: "30m",
		Count:     2,
		Interval:  1, // 1ns — effectively immediate; ctx cancels after N emits
	})
	if err != nil {
		t.Fatalf("StreamChart returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	emitted := []domain.Candle{}

	go func() {
		defer cancel()
		// Expect 3 emits: initial(101), intra(102), new bucket(103). After receiving
		// those, sleep briefly to drain at least one more poll cycle (which must add
		// no new emits) then stop.
		for c := range stream.Ticks() {
			emitted = append(emitted, c)
			if len(emitted) == 3 {
				time.Sleep(20 * time.Millisecond)
				stream.Close()
				return
			}
		}
	}()

	if err := stream.Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run returned %v", err)
	}

	if len(emitted) != 3 {
		t.Fatalf("expected 3 candles emitted, got %d: %+v", len(emitted), emitted)
	}
	if emitted[0].Close != 101 || emitted[1].Close != 102 || emitted[2].Close != 103 {
		t.Fatalf("unexpected emit order: %+v", emitted)
	}
}
```

If `chart_test.go` does not yet have helpers `fixtureRoot` / `mustReadFile`, copy them from `internal/client/quotes_test.go`. Add imports: `context`, `net/http`, `net/http/httptest`, `path/filepath`, `strings`, `sync/atomic`, `testing`, `time`, and `domain`.

The channel name `stream.Ticks()` is intentional — see Step 3 for why we keep the existing method name across all stream types.

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/client/ -run TestStreamChartEmitsIntraBucketAndNewBucket -v`
Expected: `undefined: Client.StreamChart` or `undefined: StreamChartOptions`.

- [ ] **Step 3: Implement `StreamChart`**

Append to `internal/client/chart.go`:

```go
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

// Ticks returns the channel callers consume. (Name kept symmetrical with the
// other Stream* types so a generic consumer loop can be substituted in tests.)
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
// pruned to the current snapshot whenever it grows beyond chartSeenCap so it
// can't grow unbounded across day boundaries.
func (s *ChartStream) emit(chart domain.Chart) {
	// chart.Candles is newest-first from GetChart; iterate in reverse.
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
```

If `sync` is not imported in `chart.go`, add it.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/client/ -run TestStreamChartEmitsIntraBucketAndNewBucket -v`
Expected: `--- PASS`.

- [ ] **Step 5: Run whole client suite**

Run: `go test ./internal/client/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/client/chart.go internal/client/chart_test.go
git commit -m "feat(client): add StreamChart with per-dt (close,volume) dedup"
```

---

## Task 5: Client — `OrderBookStream` + `StreamOrderBook`

**Files:**
- Modify: `internal/client/quotes.go`
- Modify: `internal/client/quotes_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/client/quotes_test.go`. Three handler responses: initial, identical (no emit), then mutated top-of-book (emit). Final assertion: 2 emits across at least 3 polls.

```go
func TestStreamOrderBookEmitsOnHashChange(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	stockInfo := mustReadFile(t, filepath.Join(root, "stock-info.json"))

	bodies := [][]byte{
		// initial
		[]byte(`{"result":{"close":167.10,"closeKrw":249000,
			"offerPrices":[167.20],"offerPricesKrw":[249100],"offerVolumes":[30],
			"bidPrices":[167.05],"bidPricesKrw":[248950],"bidVolumes":[87],
			"offerVolume":30,"bidVolume":87}}`),
		// identical -> no emit
		[]byte(`{"result":{"close":167.10,"closeKrw":249000,
			"offerPrices":[167.20],"offerPricesKrw":[249100],"offerVolumes":[30],
			"bidPrices":[167.05],"bidPricesKrw":[248950],"bidVolumes":[87],
			"offerVolume":30,"bidVolume":87}}`),
		// offerVolumes changed -> emit
		[]byte(`{"result":{"close":167.10,"closeKrw":249000,
			"offerPrices":[167.20],"offerPricesKrw":[249100],"offerVolumes":[40],
			"bidPrices":[167.05],"bidPricesKrw":[248950],"bidVolumes":[87],
			"offerVolume":40,"bidVolume":87}}`),
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v2/stock-infos/"):
			w.Write(stockInfo)
		case strings.HasSuffix(r.URL.Path, "/quotes"):
			idx := int(calls.Add(1)) - 1
			if idx >= len(bodies) {
				idx = len(bodies) - 1
			}
			w.Write(bodies[idx])
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	stream, err := c.StreamOrderBook(StreamOrderBookOptions{
		Symbol:   "US20100311002",
		Interval: 1, // 1ns
	})
	if err != nil {
		t.Fatalf("StreamOrderBook error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	emitted := []domain.OrderBook{}

	go func() {
		defer cancel()
		for book := range stream.Books() {
			emitted = append(emitted, book)
			if len(emitted) == 2 {
				time.Sleep(20 * time.Millisecond)
				stream.Close()
				return
			}
		}
	}()

	if err := stream.Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run returned %v", err)
	}

	if len(emitted) != 2 {
		t.Fatalf("expected 2 emits, got %d: %+v", len(emitted), emitted)
	}
	if emitted[0].Offers[0].Volume != 30 || emitted[1].Offers[0].Volume != 40 {
		t.Fatalf("unexpected emit volumes: %+v", emitted)
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/client/ -run TestStreamOrderBookEmitsOnHashChange -v`
Expected: `undefined: Client.StreamOrderBook`.

- [ ] **Step 3: Implement `StreamOrderBook`**

Append to `internal/client/quotes.go`:

```go
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
// keeps the implementation dependency-free and collisions are negligible at this
// payload size.
func hashOrderBook(book domain.OrderBook) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	hashLevels := func(levels []domain.OrderBookLevel) {
		for _, lvl := range levels {
			binary.BigEndian.PutUint64(buf[:], math.Float64bits(lvl.Price))
			h.Write(buf[:])
			binary.BigEndian.PutUint64(buf[:], math.Float64bits(lvl.Volume))
			h.Write(buf[:])
		}
		h.Write([]byte{0xff}) // section separator
	}
	hashLevels(book.Offers)
	hashLevels(book.Bids)
	return h.Sum64()
}
```

Add imports if missing: `encoding/binary`, `hash/fnv`, `math`.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/client/ -run TestStreamOrderBookEmitsOnHashChange -v`
Expected: `--- PASS`.

- [ ] **Step 5: Run whole client suite**

Run: `go test ./internal/client/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/client/quotes.go internal/client/quotes_test.go
git commit -m "feat(client): add StreamOrderBook with hash-based change detection"
```

---

## Task 6: Client — `QuoteStream` + `StreamQuote`

**Files:**
- Modify: `internal/client/quote.go`
- Modify: `internal/client/quote_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Append to `internal/client/quote_test.go`:

```go
func TestStreamQuoteEmitsOnPriceOrVolumeChange(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	stockInfo := mustReadFile(t, filepath.Join(root, "stock-info.json"))

	// Each "details" body is a separate snapshot. Volume changes between 1 and 2,
	// then stays put; price changes between 2 and 3.
	bodies := [][]byte{
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:00","open":166,"high":172,"low":161,"close":167.10,"volume":1500000,"base":186.19}]}`),
		// volume change -> emit
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:01","open":166,"high":172,"low":161,"close":167.10,"volume":1501000,"base":186.19}]}`),
		// identical to previous -> no emit
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:01","open":166,"high":172,"low":161,"close":167.10,"volume":1501000,"base":186.19}]}`),
		// price change -> emit
		[]byte(`{"result":[{"code":"US20100311002","currency":"USD","tradeDateTime":"2026-05-15T15:00:02","open":166,"high":172,"low":161,"close":167.20,"volume":1501000,"base":186.19}]}`),
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v2/stock-infos/"):
			w.Write(stockInfo)
		case strings.Contains(r.URL.Path, "/api/v3/stock-prices/details"):
			idx := int(calls.Add(1)) - 1
			if idx >= len(bodies) {
				idx = len(bodies) - 1
			}
			w.Write(bodies[idx])
		case strings.HasPrefix(r.URL.Path, "/api/v1/stock-infos/header/"):
			w.Write([]byte(`{"result":{"sections":[]}}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/stock-detail/ui/"):
			w.Write([]byte(`{"result":{"badges":[],"notices":[]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	stream, err := c.StreamQuote(StreamQuoteOptions{
		Symbol:   "US20100311002",
		Interval: 1, // 1ns
	})
	if err != nil {
		t.Fatalf("StreamQuote error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	emitted := []domain.Quote{}

	go func() {
		defer cancel()
		for q := range stream.Quotes() {
			emitted = append(emitted, q)
			if len(emitted) == 3 {
				time.Sleep(20 * time.Millisecond)
				stream.Close()
				return
			}
		}
	}()

	if err := stream.Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run returned %v", err)
	}

	if len(emitted) != 3 {
		t.Fatalf("expected 3 emits, got %d: %+v", len(emitted), emitted)
	}
	// 1st: initial. 2nd: volume change. 3rd: price change.
	if emitted[0].Last != 167.10 || emitted[1].Volume != 1501000 || emitted[2].Last != 167.20 {
		t.Fatalf("unexpected emits: %+v", emitted)
	}
}
```

- [ ] **Step 2: Run the test; it must fail**

Run: `go test ./internal/client/ -run TestStreamQuoteEmitsOnPriceOrVolumeChange -v`
Expected: `undefined: Client.StreamQuote`.

- [ ] **Step 3: Implement `StreamQuote`**

Append to `internal/client/quote.go`:

```go
// StreamQuoteOptions controls a long-running quote poll.
type StreamQuoteOptions struct {
	Symbol   string
	Interval time.Duration // default 3s
	OnError  func(error)
}

// QuoteStream drives a quote poll loop and emits a snapshot whenever
// (last, volume, tradeDateTime — surfaced via Quote.FetchedAt sub-second flips)
// changes. Concretely we compare (last, volume) tuples; FetchedAt is a client
// timestamp and not part of the dedup key.
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

// emit emits the snapshot when (last, volume) differs from the previous emission
// (or is the first emission).
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
```

If `sync` is missing from the import block, add it.

- [ ] **Step 4: Run the test; it must pass**

Run: `go test ./internal/client/ -run TestStreamQuoteEmitsOnPriceOrVolumeChange -v`
Expected: `--- PASS`.

- [ ] **Step 5: Run whole client suite**

Run: `go test ./internal/client/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/client/quote.go internal/client/quote_test.go
git commit -m "feat(client): add StreamQuote with (last,volume) change detection"
```

---

## Task 7: CLI — `chart get --follow`

**Files:**
- Modify: `cmd/tossctl/chart.go`

- [ ] **Step 1: Modify chart.go to add follow flags and follow path**

Replace the body of `newChartCmd` in `cmd/tossctl/chart.go` so the `get` subcommand grows `--follow` / `--interval`. Full updated function below; preserve the package, imports, and any pre-existing variables. Add imports `context`, `errors`, `io`, `os`, `os/signal`, `syscall`.

```go
func newChartCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chart",
		Short: "Read OHLCV chart candles",
	}

	var (
		timeframe  string
		count      int
		session    string
		investMode string
		from       string
		noAdjust   bool
		follow     bool
		interval   time.Duration
	)

	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch chart candles for a symbol; --follow for NDJSON stream",
		Long: `Fetch OHLCV candles from Toss Securities.

Supported --tf values: 1m, 3m, 5m, 10m, 15m, 30m, 1h (= 60m), 1d, 1w, 1mo, 3mo, 1y.
Supported --session values: all (default), main, day, pre, after.

Without --follow this prints the requested candles once.

With --follow this becomes a long-running stream: every --interval (default 60s)
the latest two buckets are polled, and any new or updated candle is emitted as
NDJSON. Dedup key is (datetime, close, volume) — intra-bucket updates and new
buckets both emit. Use Ctrl-C to stop.

Toss exposes no WebSocket/SSE for prices; --follow is REST polling. During
closed market hours the dedup keys stop changing so no lines are emitted; this
is intentional and not a broken stream.

Examples:
  tossctl chart get SOXL --tf 1m --count 60
  tossctl chart get SOXL --tf 1m --follow --interval 60s
  tossctl chart get A005930 --tf 5m --follow`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			chartOpts := client.ChartOptions{
				Timeframe:  timeframe,
				Count:      count,
				Session:    session,
				InvestMode: investMode,
			}
			if noAdjust {
				adjusted := false
				chartOpts.UseAdjusted = &adjusted
			}
			if from != "" {
				parsed, err := time.Parse(time.RFC3339, from)
				if err != nil {
					return fmt.Errorf("--from must be an RFC3339 timestamp with timezone: %w", err)
				}
				chartOpts.From = parsed
			}

			if !follow {
				chart, err := app.client.GetChart(cmd.Context(), args[0], chartOpts)
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteChart(cmd.OutOrStdout(), app.format, chart)
			}
			return runChartFollow(cmd.Context(), app, args[0], timeframe, count, session, interval, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	getCmd.Flags().StringVar(&timeframe, "tf", "1d", "Timeframe alias: 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y")
	getCmd.Flags().IntVar(&count, "count", 100, "Number of candles (1..); when --follow is set, defaults to 2 if unspecified")
	getCmd.Flags().StringVar(&session, "session", "", "Session filter: all/main/day/pre/after (default: server default)")
	getCmd.Flags().StringVar(&investMode, "invest-mode", "", "Invest mode: integrated/regular (default: server default)")
	getCmd.Flags().StringVar(&from, "from", "", "Pagination cursor — RFC3339 timestamp; older-than this point")
	getCmd.Flags().BoolVar(&noAdjust, "no-adjust", false, "Disable split/dividend adjustment (default: enabled)")
	getCmd.Flags().BoolVar(&follow, "follow", false, "Stream candle updates as NDJSON until Ctrl-C")
	getCmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "Poll interval when --follow is set")

	cmd.AddCommand(getCmd)
	return cmd
}

func runChartFollow(ctx context.Context, app *appContext, symbol, timeframe string, count int, session string, interval time.Duration, stdout, stderr io.Writer) error {
	pollCount := count
	if pollCount <= 0 || pollCount > 5 {
		// Tight default per advisor — the stream's purpose is "what just changed",
		// not "give me 100 buckets every minute". Cap at 5 even if user passed more.
		pollCount = 2
	}
	stream, err := app.client.StreamChart(client.StreamChartOptions{
		Symbol:    symbol,
		Timeframe: timeframe,
		Count:     pollCount,
		Interval:  interval,
		Session:   session,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "chart poll error: %v\n", err)
		},
	})
	if err != nil {
		return userFacingCommandError(err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Run(sigCtx)
	}()

	for {
		select {
		case candle, ok := <-stream.Ticks():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteCandleNDJSON(stdout, candle); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Verify help text**

Run: `go run ./cmd/tossctl chart get --help`
Expected: `--follow` and `--interval` rows appear; example list mentions `--follow`.

- [ ] **Step 4: Run the cmd suite**

Run: `go test ./cmd/tossctl/`
Expected: ok.

- [ ] **Step 5: Commit**

```bash
git add cmd/tossctl/chart.go
git commit -m "feat(cli): chart get --follow (NDJSON candle stream)"
```

---

## Task 8: CLI — `quotes book --follow`

**Files:**
- Modify: `cmd/tossctl/quotes.go`

- [ ] **Step 1: Modify quotes.go to add --follow to book**

Replace the `bookCmd` block inside `newQuotesCmd` with one that adds `--follow` / `--interval`, and add `runBookFollow` at the bottom of the file. Full updated block below.

Within `newQuotesCmd`, replace the local variable declarations near the top and the `bookCmd` definition so the function reads:

```go
func newQuotesCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quotes",
		Short: "Read orderbook and tick data",
	}

	var (
		bookFollow   bool
		bookInterval time.Duration
	)
	bookCmd := &cobra.Command{
		Use:   "book <symbol>",
		Short: "Show the latest orderbook (호가창); --follow for NDJSON stream",
		Long: `Show the latest orderbook for a symbol.

Without --follow this prints a snapshot.

With --follow this becomes a long-running stream: every --interval (default 1s)
the orderbook is re-fetched and a snapshot is emitted as NDJSON whenever any
price or volume across all levels changes. Use Ctrl-C to stop.

Toss exposes no WebSocket/SSE for the orderbook; --follow is REST polling.
During closed market hours the orderbook does not change so no lines are
emitted; this is intentional.

Examples:
  tossctl quotes book SOXL
  tossctl quotes book SOXL --follow --interval 1s
  tossctl quotes book A005930 --follow`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if !bookFollow {
				book, err := app.client.GetOrderBook(cmd.Context(), args[0])
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteOrderBook(cmd.OutOrStdout(), app.format, book)
			}
			return runBookFollow(cmd.Context(), app, args[0], bookInterval, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	bookCmd.Flags().BoolVar(&bookFollow, "follow", false, "Stream orderbook updates as NDJSON until Ctrl-C")
	bookCmd.Flags().DurationVar(&bookInterval, "interval", 1*time.Second, "Poll interval when --follow is set")

	// ... existing ticksCmd definition stays as-is below this point ...
```

Keep the existing `ticksCmd` block and the closing `cmd.AddCommand(bookCmd, ticksCmd)` line.

Append at the bottom of the file:

```go
func runBookFollow(ctx context.Context, app *appContext, symbol string, interval time.Duration, stdout, stderr io.Writer) error {
	stream, err := app.client.StreamOrderBook(client.StreamOrderBookOptions{
		Symbol:   symbol,
		Interval: interval,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "orderbook poll error: %v\n", err)
		},
	})
	if err != nil {
		return userFacingCommandError(err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Run(sigCtx)
	}()

	for {
		select {
		case book, ok := <-stream.Books():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteOrderBookNDJSON(stdout, book); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Verify help text**

Run: `go run ./cmd/tossctl quotes book --help`
Expected: `--follow` and `--interval` rows appear; example list mentions `--follow`.

- [ ] **Step 4: Run the cmd suite**

Run: `go test ./cmd/tossctl/`
Expected: ok.

- [ ] **Step 5: Commit**

```bash
git add cmd/tossctl/quotes.go
git commit -m "feat(cli): quotes book --follow (NDJSON orderbook stream)"
```

---

## Task 9: CLI — `quote get --follow`

**Files:**
- Modify: `cmd/tossctl/quote.go`

- [ ] **Step 1: Modify quote.go to add --follow to get**

Read the current `cmd/tossctl/quote.go` first. The file already has `newQuoteCmd` with a `get` subcommand that calls `GetQuote`. Add `--follow` / `--interval` flags and a `runQuoteFollow` helper. The pattern below preserves the snapshot path and adds the streaming branch:

```go
func newQuoteCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote",
		Short: "Read quote data",
	}

	var (
		follow   bool
		interval time.Duration
	)
	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch the latest quote for a symbol; --follow for NDJSON stream",
		Long: `Show the latest quote (price, OHLC, 52w/1y range, marketCap, trading
strength, ETF expense ratio, dividend yield, ranking).

Without --follow this prints a snapshot.

With --follow this becomes a long-running stream: every --interval (default 3s)
the quote is re-fetched and a snapshot is emitted as NDJSON whenever last price
or volume changes. Use Ctrl-C to stop.

Toss exposes no WebSocket/SSE for prices; --follow is REST polling. During
closed market hours last and volume do not change so no lines are emitted;
this is intentional.

Examples:
  tossctl quote get SOXL
  tossctl quote get SOXL --follow --interval 3s
  tossctl quote get A005930 --follow`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if !follow {
				quote, err := app.client.GetQuote(cmd.Context(), args[0])
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteQuote(cmd.OutOrStdout(), app.format, quote)
			}
			return runQuoteFollow(cmd.Context(), app, args[0], interval, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	getCmd.Flags().BoolVar(&follow, "follow", false, "Stream quote updates as NDJSON until Ctrl-C")
	getCmd.Flags().DurationVar(&interval, "interval", 3*time.Second, "Poll interval when --follow is set")

	cmd.AddCommand(getCmd)
	return cmd
}

func runQuoteFollow(ctx context.Context, app *appContext, symbol string, interval time.Duration, stdout, stderr io.Writer) error {
	stream, err := app.client.StreamQuote(client.StreamQuoteOptions{
		Symbol:   symbol,
		Interval: interval,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "quote poll error: %v\n", err)
		},
	})
	if err != nil {
		return userFacingCommandError(err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Run(sigCtx)
	}()

	for {
		select {
		case q, ok := <-stream.Quotes():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteQuoteNDJSON(stdout, q); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
```

Add imports if missing: `context`, `errors`, `io`, `os`, `os/signal`, `syscall`, `time`, and the `client` / `output` packages already in use.

If the existing function name for the snapshot writer is `output.WriteQuote` but the file uses a different exported name, adapt the call site (read the file first to confirm).

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 3: Verify help text**

Run: `go run ./cmd/tossctl quote get --help`
Expected: `--follow` and `--interval` rows appear; example list mentions `--follow`.

- [ ] **Step 4: Run the cmd suite**

Run: `go test ./cmd/tossctl/`
Expected: ok.

- [ ] **Step 5: Commit**

```bash
git add cmd/tossctl/quote.go
git commit -m "feat(cli): quote get --follow (NDJSON quote stream)"
```

---

## Task 10: Docs — CHANGELOG + rpc-catalog CLI mapping

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/reverse-engineering/rpc-catalog.md`

- [ ] **Step 1: Update CHANGELOG.md**

Add an Unreleased / next-version section (mirror the existing style):

```markdown
### Added
- `tossctl chart get --follow` — NDJSON candle stream with per-`dt` `(close, volume)` dedup. Default `--interval 60s`, `--count 2`.
- `tossctl quotes book --follow` — NDJSON orderbook stream emitting on any change to the (offer/bid prices, volumes) hash. Default `--interval 1s`.
- `tossctl quote get --follow` — NDJSON quote stream emitting on `(last, volume)` change. Default `--interval 3s`.

### Changed
- `cmd/tossctl/chart.go`, `cmd/tossctl/quotes.go`, `cmd/tossctl/quote.go` long help text now documents that `--follow` is REST polling (no SSE/WebSocket from Toss) and that closed-market silence is correct behavior.
```

- [ ] **Step 2: Update `docs/reverse-engineering/rpc-catalog.md` Realtime Strategy section**

In the `## Realtime Strategy` table near `rpc-catalog.md:148-161`, append a column or footnote indicating which CLI command consumes each surface for streaming. Minimum: add a short paragraph after the table:

```markdown
**CLI mappings (added in PR5):**
- `Last/OHLC/체결강도/marketCap` → `tossctl quote get --follow` (default 3s)
- `Orderbook` → `tossctl quotes book --follow` (default 1s)
- `Recent ticks` → `tossctl quotes ticks --follow` (default 2s)
- `Intraday candle` → `tossctl chart get --tf <1m|5m|15m|...> --follow` (default 60s)
- `Push triggers` → `tossctl push listen`
```

- [ ] **Step 3: Run the whole test suite as a regression**

Run: `go vet ./... && go test ./... -count=1`
Expected: all packages pass.

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md docs/reverse-engineering/rpc-catalog.md
git commit -m "docs: note --follow streaming additions for chart/book/quote"
```

---

## Self-review notes

- **No generic abstraction.** Three concrete `ChartStream` / `OrderBookStream` / `QuoteStream` types. Reviewer agents may be tempted to extract a base — resist. Dedup logic differs structurally per type; sharing a base would tangle three otherwise-readable files. (Confirmed with advisor before drafting.)
- **All defaults anchored to `rpc-catalog.md:148-161`.** Chart 60s, book 1s, quote 3s. Don't bikeshed these — they came from the reverse engineering.
- **Tests use `httptest` with a programmatic mutator pattern.** No new fixture files needed. Mutation is in-memory between served responses.
- **Method name `Ticks()` reused for ChartStream** so a generic consumer loop could be substituted in tests. `Books()` and `Quotes()` differ because the channel element type is the primary signal in the code reading the stream. This is a deliberate stylistic split, not an oversight.
- **No heartbeat for closed markets.** Plan documents closed-market silence in `--help` long text only. Adding heartbeats would pollute NDJSON for the LLM consumer.
- **Snapshot semantics for `quotes book --follow`.** We emit the whole orderbook every change (not a delta) because Toss returns the whole book each poll; deltas would require expensive cross-snapshot diffing that the consumer can do trivially.
- **`--count` cap on chart follow** is 5. Higher values waste bandwidth — the stream's job is "what just changed", not "give me history". Snapshot mode still honors the full `--count`.
- **`-h` short flag** is reserved by cobra for help. Don't try to add `-f` for follow — keep flags long-only to match the existing `--follow` on `quotes ticks`.

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-05-16-follow-streaming.md`. Execute with **superpowers:subagent-driven-development** — fresh subagent per task, spec compliance review then code quality review per task.

After Task 10, merge `--no-ff` back into `feat/order-page-integration`, push to fork (`origin`) only, never to `upstream`.
