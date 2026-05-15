package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func TestWriteAnalystSnapshotTable(t *testing.T) {
	t.Parallel()
	snap := domain.AnalystSnapshot{
		ProductCode: "NAS0250224006",
		Opinion: domain.AnalystOpinion{
			Type: "BUY", StrongBuy: 6, Buy: 12, Hold: 4, Sell: 0, StrongSell: 0,
			TargetUSD: 1224.42, TargetKRW: 1794999.72,
			Description: "애널리스트 22명 중 18명이 구매 의견을 냈어요.",
		},
		Consensus: domain.ConsensusTarget{
			Mean: 1224.42, High: 2000.0, Low: 250.0,
			MeanKRW: 1794999.72, HighKRW: 2932000.0, LowKRW: 366500.0,
			Currency: "USD", PointDate: "2026-05-15",
			PastCloses: []domain.ConsensusPastClose{
				{Date: "2026-05-15", Price: 1405.85, PriceKRW: 2097247},
				{Date: "2026-04-30", Price: 1096.51, PriceKRW: 1635773},
			},
		},
		Reports:   nil,
		FetchedAt: time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
	}
	var buf bytes.Buffer
	if err := WriteAnalystSnapshot(&buf, FormatTable, snap); err != nil {
		t.Fatalf("WriteAnalystSnapshot error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAS0250224006", "BUY", "strongBuy 6", "buy 12", "hold 4", "$1224.42", "$2000.00", "$250.00", "2026-05-15", "$1405.85", "0건"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\nfull:\n%s", want, out)
		}
	}
}
