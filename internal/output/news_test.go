package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteStockNewsTable(t *testing.T) {
	t.Parallel()
	items := []domain.NewsItem{
		{ID: "a", Title: "삼성전자 노사 대화 재개", Summary: "18일 노사 사후조정",
			Source: domain.NewsSource{Code: "ajukyung", Name: "아주경제"}, CreatedAt: "2026-05-16T17:13:25"},
		{ID: "b", Title: "분기 실적 발표", Summary: "메모리 성장으로 최대 실적",
			Source: domain.NewsSource{Code: "fn", Name: "파이낸셜뉴스"}, CreatedAt: "2026-05-15T10:00:00"},
	}
	var buf bytes.Buffer
	if err := WriteStockNews(&buf, FormatTable, items); err != nil {
		t.Fatalf("WriteStockNews error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"2026-05-16", "아주경제", "삼성전자 노사 대화 재개",
		"2026-05-15", "파이낸셜뉴스", "분기 실적 발표",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockNewsCSV(t *testing.T) {
	t.Parallel()
	items := []domain.NewsItem{
		{ID: "a", Title: "T, with comma", Summary: "S",
			Source: domain.NewsSource{Code: "x", Name: "Y, Inc"}, CreatedAt: "2026-05-16T17:13:25"},
	}
	var buf bytes.Buffer
	if err := WriteStockNews(&buf, FormatCSV, items); err != nil {
		t.Fatalf("CSV error: %v", err)
	}
	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header + 1 row, got %d", len(rows))
	}
	if rows[1][2] != "T, with comma" {
		t.Fatalf("comma not preserved; got %q", rows[1][2])
	}
}

func TestWriteStockNewsJSON(t *testing.T) {
	t.Parallel()
	items := []domain.NewsItem{
		{ID: "a", Title: "X", Source: domain.NewsSource{Name: "Y"}, CreatedAt: "2026-05-16T17:13:25"},
	}
	var buf bytes.Buffer
	if err := WriteStockNews(&buf, FormatJSON, items); err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	var got []domain.NewsItem
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got[0].Source.Name != "Y" {
		t.Fatalf("roundtrip mismatch")
	}
	if !strings.Contains(buf.String(), `"created_at"`) {
		t.Fatalf("expected snake_case created_at")
	}
}

func TestWriteStockFilingsTable(t *testing.T) {
	t.Parallel()
	items := []domain.FilingItem{
		{ID: "k", Title: "파생상품시장 안내", Summary: "주식선물ㆍ주식옵션",
			Form: "HTML", CreatedAt: "2026-05-15T00:00:00"},
		{ID: "d", Title: "2026년 3월 확정실적 발표", Summary: "영업이익 57조",
			Form: "EARNINGS", CreatedAt: "2026-05-15T00:00:00",
			EarningCall: &domain.EarningCall{Status: "ENDED", Title: "26년 1분기 실적발표"}},
	}
	var buf bytes.Buffer
	if err := WriteStockFilings(&buf, FormatTable, items); err != nil {
		t.Fatalf("WriteStockFilings error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"HTML", "파생상품시장 안내", "EARNINGS",
		"확정실적 발표", "↳ 어닝콜", "26년 1분기 실적발표",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestWriteStockFilingsEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := WriteStockFilings(&buf, FormatTable, nil); err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "No filings") {
		t.Fatalf("expected 'No filings' message")
	}
}
