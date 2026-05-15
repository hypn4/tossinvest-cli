package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

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
