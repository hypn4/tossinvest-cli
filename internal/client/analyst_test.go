package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
