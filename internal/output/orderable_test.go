package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

var sampleOrderable = domain.OrderableSummary{
	OrderableKR: domain.Money{KRW: 110421},
	OrderableUS: domain.Money{KRW: 758893, USD: 508.71},
	KR:          domain.TransactionOverview{Market: "kr", OrderableKRW: 110421, Withdrawable: []domain.SettlementBucket{{Date: "2026-05-15", KRW: 110421}}},
	US:          domain.TransactionOverview{Market: "us", OrderableUSD: 508.71, Withdrawable: []domain.SettlementBucket{{Date: "2026-05-15", USD: 508.71}}},
	FetchedAt:   time.Now(),
}

func TestWriteOrderableJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderable(&buf, FormatJSON, sampleOrderable); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.OrderableSummary
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.OrderableUS.USD != 508.71 {
		t.Fatalf("USD round-trip failed: %+v", parsed.OrderableUS)
	}
}

func TestWriteOrderableTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteOrderable(&buf, FormatTable, sampleOrderable); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "110,421") || !strings.Contains(out, "$508.71") {
		t.Fatalf("expected KR/US amounts in output: %s", out)
	}
}
