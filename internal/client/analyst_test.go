package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetAnalystSnapshotFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Opinion   json.RawMessage `json:"opinion"`
		Consensus json.RawMessage `json:"consensus"`
		Reports   json.RawMessage `json:"reports"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "analyst-snapshot-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/stock-detail/ui/wts/NAS0250224006/analyst-opinion":
			w.Write(bundle.Opinion)
		case "/api/v2/stock-infos/consensus/NAS0250224006":
			w.Write(bundle.Consensus)
		case "/api/v1/stock-detail/ui/wts/NAS0250224006/analyst-reports":
			w.Write(bundle.Reports)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	snap, err := c.GetAnalystSnapshot(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetAnalystSnapshot error: %v", err)
	}
	if snap.Opinion.Type != "BUY" {
		t.Fatalf("expected BUY, got %q", snap.Opinion.Type)
	}
	if snap.Opinion.Buy != 12 || snap.Opinion.StrongBuy != 6 {
		t.Fatalf("unexpected opinion counts: %+v", snap.Opinion)
	}
	if snap.Consensus.Mean != 1224.42 {
		t.Fatalf("expected mean 1224.42, got %v", snap.Consensus.Mean)
	}
	if len(snap.Consensus.PastCloses) != 3 {
		t.Fatalf("expected 3 past closes, got %d", len(snap.Consensus.PastCloses))
	}
	if len(snap.Reports) != 0 {
		t.Fatalf("expected 0 reports, got %d", len(snap.Reports))
	}
}

func TestGetAnalystSnapshotFlattensReportGroups(t *testing.T) {
	t.Parallel()
	opinion := `{"result":{"type":"BUY","strongBuy":1,"buy":2,"hold":0,"sell":0,"strongSell":0,"targetPrice":{"USD":100,"KRW":140000},"description":"x"}}`
	consensus := `{"result":{"targetPrice":{"mean":100,"meanKrw":140000,"high":120,"highKrw":168000,"low":80,"lowKrw":112000,"currency":"USD"},"pointDate":"2026-05-15","pastClosePrices":[]}}`
	reports := `{"result":{"analystReportGroups":[
        {"date":"2026-05-10","reports":[
            {"title":"Buy thesis","source":"Morgan Stanley","url":"https://example/r1"},
            {"title":"Earnings preview","source":"Goldman","url":"https://example/r2"}
        ]},
        {"date":"2026-04-22","reports":[
            {"title":"Initiation","source":"JPM","url":"https://example/r3"}
        ]}
    ]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/analyst-opinion"):
			w.Write([]byte(opinion))
		case strings.HasSuffix(r.URL.Path, "/consensus/NAS0250224006"):
			w.Write([]byte(consensus))
		case strings.HasSuffix(r.URL.Path, "/analyst-reports"):
			w.Write([]byte(reports))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	snap, err := c.GetAnalystSnapshot(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetAnalystSnapshot error: %v", err)
	}
	if len(snap.Reports) != 3 {
		t.Fatalf("expected 3 flattened reports, got %d", len(snap.Reports))
	}
	// Verify group.Date propagation: first two reports get the 2026-05-10 date
	if snap.Reports[0].Date != "2026-05-10" || snap.Reports[1].Date != "2026-05-10" {
		t.Fatalf("expected first two reports to inherit 2026-05-10, got %q / %q", snap.Reports[0].Date, snap.Reports[1].Date)
	}
	if snap.Reports[2].Date != "2026-04-22" {
		t.Fatalf("expected third report to inherit 2026-04-22, got %q", snap.Reports[2].Date)
	}
	if snap.Reports[0].Title != "Buy thesis" || snap.Reports[0].Source != "Morgan Stanley" {
		t.Fatalf("unexpected first report: %+v", snap.Reports[0])
	}
}
