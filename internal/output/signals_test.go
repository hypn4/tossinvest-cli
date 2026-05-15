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

var sampleDetail = domain.SignalDetail{
	ProductCode:      "US20190226001",
	AssetName:        "슈퍼 리그 엔터프라이즈",
	SignalDirection:  1,
	Description:      "실적 개선 효과",
	DescriptionItems: []string{"매출 10.49% 증가", "Misfits Ads 인수 효과", "애널리스트 매수 의견"},
	Keywords:         []string{"실적 개선", "매출 증가", "인수 효과"},
	News: []domain.SignalNews{
		{AgencyName: "벤징가", Title: "1분기 실적 전망치 상회", CreatedAt: time.Now()},
	},
	Related: []domain.RelatedSignal{
		{AssetName: "피스클노트 홀딩스", Relation: "같은산업", Description: []string{"두 기업 모두 광고 산업군에 속해 있어요."}},
	},
	FetchedAt: time.Now(),
}

func TestWriteSignalDetailJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSignalDetail(&buf, FormatJSON, sampleDetail); err != nil {
		t.Fatalf("error: %v", err)
	}
	var parsed domain.SignalDetail
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.AssetName != "슈퍼 리그 엔터프라이즈" {
		t.Fatalf("asset name decoded incorrectly: %s", parsed.AssetName)
	}
}

func TestWriteSignalDetailTable(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSignalDetail(&buf, FormatTable, sampleDetail); err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"실적 개선 효과", "매출 10.49% 증가", "벤징가", "피스클노트 홀딩스", "같은산업"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in table output, got:\n%s", want, out)
		}
	}
}
