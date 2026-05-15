package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestChartOptionsResolveTimeframe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		opts     ChartOptions
		wantUnit string
		wantStep int
		wantErr  bool
	}{
		{"alias 30m", ChartOptions{Timeframe: "30m"}, "min", 30, false},
		{"alias 1h maps to min:60", ChartOptions{Timeframe: "1h"}, "min", 60, false},
		{"alias 3mo", ChartOptions{Timeframe: "3mo"}, "month", 3, false},
		{"raw unit/step", ChartOptions{Unit: "day", Step: 1}, "day", 1, false},
		{"empty defaults to day:1", ChartOptions{}, "day", 1, false},
		{"unsupported alias", ChartOptions{Timeframe: "2m"}, "", 0, true},
		{"unsupported unit", ChartOptions{Unit: "hour", Step: 1}, "", 0, true},
		{"unsupported step", ChartOptions{Unit: "min", Step: 20}, "", 0, true},
		{"unsupported month step", ChartOptions{Unit: "month", Step: 6}, "", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unit, step, err := tc.opts.ResolveTimeframe()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %+v", tc.opts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if unit != tc.wantUnit || step != tc.wantStep {
				t.Fatalf("got %s:%d, want %s:%d", unit, step, tc.wantUnit, tc.wantStep)
			}
		})
	}
}

func TestValidateSessionInvestMode(t *testing.T) {
	t.Parallel()

	for _, v := range []string{"", "all", "main", "day", "pre", "after"} {
		if _, err := validateSession(v); err != nil {
			t.Fatalf("validateSession(%q) returned error: %v", v, err)
		}
	}
	if _, err := validateSession("regular"); err == nil {
		t.Fatal("validateSession should reject 'regular'")
	}

	for _, v := range []string{"", "integrated", "regular"} {
		if _, err := validateInvestMode(v); err != nil {
			t.Fatalf("validateInvestMode(%q) returned error: %v", v, err)
		}
	}
	if _, err := validateInvestMode("crazy"); err == nil {
		t.Fatal("validateInvestMode should reject unknown value")
	}
}

func TestGetChartFromFixture(t *testing.T) {
	t.Parallel()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	fixtureRoot := filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/stock-infos/A005930":
			http.ServeFile(w, r, filepath.Join(fixtureRoot, "stock-info.json"))
		case strings.HasPrefix(r.URL.Path, "/api/v1/c-chart/kr-s/A005930/day:1"):
			http.ServeFile(w, r, filepath.Join(fixtureRoot, "chart-day-1.json"))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient:  server.Client(),
		InfoBaseURL: server.URL,
	})

	chart, err := client.GetChart(context.Background(), "005930", ChartOptions{Timeframe: "1d", Count: 30})
	if err != nil {
		t.Fatalf("GetChart returned error: %v", err)
	}
	if chart.ProductCode != "A005930" {
		t.Fatalf("unexpected product code: %s", chart.ProductCode)
	}
	if chart.Unit != "day" || chart.Step != 1 {
		t.Fatalf("unexpected timeframe: %s:%d", chart.Unit, chart.Step)
	}
	if len(chart.Candles) == 0 {
		t.Fatal("expected at least one candle from the fixture")
	}
	first := chart.Candles[0]
	if first.Open == 0 || first.Close == 0 {
		t.Fatalf("first candle missing OHLC fields: %+v", first)
	}
}

func TestChartProductPrefix_OptionRoutesToUSO(t *testing.T) {
	cases := []struct {
		name        string
		marketCode  string
		productCode string
		want        string
	}{
		{"OPT_ with NSQ market", "NSQ", "OPT_SNDK260515C01395000_20260506", "us-o"},
		{"OPT_ with empty market", "", "OPT_SNDK260515C01395000_20260506", "us-o"},
		{"NAS stock unaffected", "NSQ", "NAS0250224006", "us-s"},
		{"KR stock unaffected", "KSP", "A005930", "kr-s"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := chartProductPrefix(c.marketCode, c.productCode)
			if got != c.want {
				t.Fatalf("chartProductPrefix(%q,%q)=%q; want %q", c.marketCode, c.productCode, got, c.want)
			}
		})
	}
}

func TestStreamChartEmitsIntraBucketAndNewBucket(t *testing.T) {
	t.Parallel()

	root := fixtureRoot(t)
	stockInfo := mustReadFile(t, filepath.Join(root, "stock-info.json"))

	bodies := [][]byte{
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":102,"low":99,"close":101,"volume":1000}
		]}}`),
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":103,"low":99,"close":102,"volume":1500}
		]}}`),
		[]byte(`{"result":{"code":"A005930","exchangeRate":1,"candles":[
			{"dt":"2026-05-15T10:30:00+09:00","open":102,"high":104,"low":101,"close":103,"volume":800},
			{"dt":"2026-05-15T10:00:00+09:00","open":100,"high":103,"low":99,"close":102,"volume":1500}
		]}}`),
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
		Interval:  1,
	})
	if err != nil {
		t.Fatalf("StreamChart returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	emitted := []domain.Candle{}

	go func() {
		defer cancel()
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
