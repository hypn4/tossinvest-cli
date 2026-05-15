package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var sampleFills = []domain.CompactExecution{
	{
		ProductCode:             "US20100311002",
		TradeType:               "buy",
		ExecutionAvgKRWPrice:    277505.50,
		ExecutionAvgLocalPrice:  185.61,
		ExecutionTotalKRWAmount: 5550110,
		ExecutionTotalLocal:     3712.20,
		Quantity:                20.0,
		BucketStart:             time.Date(2026, 5, 15, 2, 0, 0, 0, time.UTC),
	},
}

func TestWriteFillsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatJSON, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.CompactExecution
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed[0].Quantity != 20 {
		t.Fatalf("unexpected qty: %v", parsed[0].Quantity)
	}
}

func TestWriteFillsTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatTable, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"buy", "20", "185.61"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output: %s", want, out)
		}
	}
}

func TestWriteFillsCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFills(&buf, FormatCSV, sampleFills); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d", len(lines))
	}
}
