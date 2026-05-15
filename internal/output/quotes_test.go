package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var sampleBook = domain.OrderBook{
	ProductCode:    "US20100311002",
	Symbol:         "SOXL",
	Market:         "AMEX",
	Currency:       "USD",
	Last:           168.60,
	Offers:         []domain.OrderBookLevel{{Price: 168.79, Volume: 7}},
	Bids:           []domain.OrderBookLevel{{Price: 168.60, Volume: 2}},
	OfferVolumeSum: 7,
	BidVolumeSum:   2,
	FetchedAt:      time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
}

func TestWriteOrderBookJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatJSON, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OrderBook
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if parsed.Symbol != "SOXL" || len(parsed.Offers) != 1 {
		t.Fatalf("unexpected parse: %+v", parsed)
	}
}

func TestWriteOrderBookTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatTable, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "SOXL") {
		t.Fatalf("expected SOXL in output: %s", out)
	}
	if !strings.Contains(out, "168.79") || !strings.Contains(out, "168.6") {
		t.Fatalf("expected prices in output: %s", out)
	}
}

func TestWriteOrderBookCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderBook(&buf, FormatCSV, sampleBook); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header + at least one offer + one bid line, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "side,price,volume") {
		t.Fatalf("unexpected header: %s", lines[0])
	}
}
