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

func TestWriteOptionExpiriesTable(t *testing.T) {
	exps := []domain.OptionExpiry{
		{MaturityDate: "2026-05-15", DisplayLiquidationDateTime: "24분 후 거래 종료"},
		{MaturityDate: "2026-05-22", DisplayLiquidationDateTime: "7일 후 거래 종료"},
	}
	var buf bytes.Buffer
	if err := WriteOptionExpiries(&buf, FormatTable, exps); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "2026-05-15") || !strings.Contains(s, "거래 종료") {
		t.Fatalf("expected expiry rows in table:\n%s", s)
	}
}

func TestWriteOptionChainTable(t *testing.T) {
	rows := []domain.OptionChainRow{
		{StrikePrice: 1395, CallGuid: "OPT_C", PutGuid: "OPT_P", CallOpenInterest: 82, PutOpenInterest: 41},
	}
	var buf bytes.Buffer
	if err := WriteOptionChain(&buf, FormatTable, rows); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "1395") || !strings.Contains(s, "82") {
		t.Fatalf("expected strike row in table:\n%s", s)
	}
}

func TestWriteOptionPricesJSON(t *testing.T) {
	prices := []domain.OptionPrice{{Code: "OPT_X", Close: 12.3, Volume: 100}}
	var buf bytes.Buffer
	if err := WriteOptionPrices(&buf, FormatJSON, prices); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed []domain.OptionPrice
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed[0].Close != 12.3 {
		t.Fatalf("unexpected roundtrip: %+v", parsed)
	}
}
