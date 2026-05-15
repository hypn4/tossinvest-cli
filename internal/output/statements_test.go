package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func statementsTestFixture() domain.StockStatements {
	v := func(f float64) *float64 { return &f }
	return domain.StockStatements{
		ProductCode: "NAS0250224006",
		Factor:      domain.StatementFactor{Code: "INC", DisplayName: "손익계산서"},
		Period:      "Q",
		IsKr:        false,
		Periods: []domain.StatementPeriod{
			{
				Period: "2025-12",
				Items: []domain.StatementLineItem{
					{Item: "RTLR", NameKor: "매출액", NameEng: "Total Revenue", Unit: "USD", Value: v(2306)},
					{Item: "SREV", ParentItem: "RTLR", NameKor: "매출", NameEng: "Revenue", Unit: "USD", Value: v(2306)},
					{Item: "TIAT", NameKor: "당기순이익", NameEng: "Net Income", Unit: "USD", Value: v(887)},
				},
			},
			{
				Period: "2026-03",
				Items: []domain.StatementLineItem{
					{Item: "RTLR", NameKor: "매출액", NameEng: "Total Revenue", Unit: "USD", Value: v(3092)},
					{Item: "SREV", ParentItem: "RTLR", NameKor: "매출", NameEng: "Revenue", Unit: "USD", Value: v(3092)},
					{Item: "TIAT", NameKor: "당기순이익", NameEng: "Net Income", Unit: "USD", Value: v(3615)},
				},
			},
		},
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestWriteStockStatementsTable(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatTable, st); err != nil {
		t.Fatalf("WriteStockStatements error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"NAS0250224006",
		"손익계산서", "(Q)",
		"ITEM", "NAME", "2025-12", "2026-03",
		"RTLR", "매출액", "2,306", "3,092",
		"  SREV", // child indent
		"TIAT", "당기순이익", "3,615",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockStatementsTableNilValues(t *testing.T) {
	t.Parallel()
	v := func(f float64) *float64 { return &f }
	st := domain.StockStatements{
		ProductCode: "X",
		Factor:      domain.StatementFactor{Code: "INC", DisplayName: "손익계산서"},
		Period:      "Q",
		Periods: []domain.StatementPeriod{
			{Period: "2026-03", Items: []domain.StatementLineItem{
				{Item: "RTLR", NameKor: "매출액", Unit: "USD", Value: v(100)},
				{Item: "SORE", ParentItem: "RTLR", NameKor: "기타 수익", Unit: "", Value: nil},
			}},
		},
	}
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatTable, st); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "—") {
		t.Fatalf("expected — for nil value; got:\n%s", buf.String())
	}
}

func TestWriteStockStatementsCSV(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatCSV, st); err != nil {
		t.Fatalf("CSV error: %v", err)
	}
	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if rows[0][0] != "item" || rows[0][1] != "parent_item" || rows[0][2] != "name_kor" {
		t.Fatalf("unexpected CSV header: %v", rows[0])
	}
	// Header + 3 items (we dedupe — each line item appears once even though we have 2 periods)
	// Actually CSV pivots periods as columns too, so we'll have item rows with period columns
	if len(rows) < 4 {
		t.Fatalf("expected at least 4 rows, got %d", len(rows))
	}
}

func TestWriteStockStatementsJSON(t *testing.T) {
	t.Parallel()
	st := statementsTestFixture()
	var buf bytes.Buffer
	if err := WriteStockStatements(&buf, FormatJSON, st); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	var got domain.StockStatements
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if got.Factor.Code != "INC" {
		t.Fatalf("factor roundtrip mismatch")
	}
	if !strings.Contains(buf.String(), `"name_kor"`) {
		t.Fatalf("expected snake_case name_kor in JSON")
	}
}
