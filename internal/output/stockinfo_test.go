package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockInfoDetailTable(t *testing.T) {
	detail := domain.StockInfoDetail{
		ProductCode: "NAS0250224006",
		Sections: []domain.StockInfoSection{
			{Type: "OVERVIEW", Data: json.RawMessage(`{"name":"샌디스크","description":"메모리 회사"}`)},
			{Type: "INDICATORS", Data: json.RawMessage(`{"values":[{"label":"PER","displayValue":"45.4배"}]}`)},
		},
		FetchedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := WriteStockInfoDetail(&buf, FormatTable, detail); err != nil {
		t.Fatalf("error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "OVERVIEW") || !strings.Contains(s, "INDICATORS") {
		t.Fatalf("expected section type names in table: %q", s)
	}
}

func TestWriteStockInfoDetailJSON(t *testing.T) {
	detail := domain.StockInfoDetail{
		ProductCode: "NAS0250224006",
		Sections:    []domain.StockInfoSection{{Type: "OVERVIEW", Data: json.RawMessage(`{"name":"샌디스크"}`)}},
	}
	var buf bytes.Buffer
	if err := WriteStockInfoDetail(&buf, FormatJSON, detail); err != nil {
		t.Fatalf("error: %v", err)
	}
	var roundtrip domain.StockInfoDetail
	if err := json.Unmarshal(buf.Bytes(), &roundtrip); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if roundtrip.ProductCode != "NAS0250224006" {
		t.Fatalf("unexpected roundtrip: %+v", roundtrip)
	}
}
