package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var sampleSignals = []domain.Signal{
	{ProductCode: "US20100311002", ReasoningDescription: "차익실현 매도"},
	{ProductCode: "US20040819002", ReasoningDescription: "AI 투자 확대에 따른 재무 부담"},
}

func TestWriteSignalsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSignals(&buf, FormatJSON, sampleSignals); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.Signal
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(parsed))
	}
}

func TestWriteSignalsCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSignals(&buf, FormatCSV, sampleSignals); err != nil {
		t.Fatalf("error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d", len(lines))
	}
}

func TestWriteSignalsTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSignals(&buf, FormatTable, sampleSignals); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "차익실현 매도") {
		t.Fatalf("expected Korean reasoning in output: %s", out)
	}
}

// Reference imports to keep them in scope for follow-up tests
// (timestamps used in detail/event fixtures).
var _ = time.Now
