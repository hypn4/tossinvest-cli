package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteOptionInstrumentTable(t *testing.T) {
	inst := domain.OptionInstrument{
		ProductCode:        "OPT_SNDK260515C01395000_20260506",
		UnderlyingSymbol:   "SNDK",
		PutCall:            "CALL",
		StrikePrice:        1395,
		Last:               31.8,
		Bid:                30.1,
		Ask:                33.9,
		OpenInterest:       82,
		MaturityDate:       "2026-05-15",
		LiquidationDisplay: "2시간 46분 후 거래 종료",
		FetchedAt:          time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteOptionInstrument(&buf, FormatTable, inst); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, needle := range []string{"SNDK", "CALL", "1395", "2026-05-15", "거래 종료"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in table output:\n%s", needle, out)
		}
	}
}

func TestWriteOptionInstrumentJSON(t *testing.T) {
	inst := domain.OptionInstrument{ProductCode: "OPT_X", UnderlyingSymbol: "X", StrikePrice: 10}
	var buf bytes.Buffer
	if err := WriteOptionInstrument(&buf, FormatJSON, inst); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OptionInstrument
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.StrikePrice != 10 {
		t.Fatalf("unexpected JSON roundtrip: %+v", parsed)
	}
}
