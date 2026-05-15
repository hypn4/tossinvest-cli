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

func TestGetStockValuationFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	bundle := struct {
		Evaluation json.RawMessage `json:"evaluation"`
		Comparison json.RawMessage `json:"comparison"`
	}{}
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, "stock-valuation-sndk.json")), &bundle); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/evaluation/NAS0250224006":
			w.Write(bundle.Evaluation)
		case "/api/v2/stock-infos/evaluation-comparison/NAS0250224006":
			w.Write(bundle.Comparison)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	val, err := c.GetStockValuation(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetStockValuation error: %v", err)
	}
	if val.PER != 45.4 {
		t.Fatalf("expected PER 45.4, got %v", val.PER)
	}
	if val.Position != "HIGH" {
		t.Fatalf("expected HIGH, got %q", val.Position)
	}
	if val.Industry != "컴퓨터와 주변기기" {
		t.Fatalf("unexpected industry: %q", val.Industry)
	}
	if len(val.Peers) != 5 {
		t.Fatalf("expected 5 peers, got %d", len(val.Peers))
	}
	var found bool
	for _, p := range val.Peers {
		if p.IsSelf && p.ProductCode == "NAS0250224006" && p.Value == 45.4 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected self peer row with NAS0250224006/45.4; peers=%+v", val.Peers)
	}
}

func TestGetStockValuationSkipsPeersWithEmptyGraph(t *testing.T) {
	t.Parallel()
	eval := `{"result":{"per":1,"pbr":1,"psr":1,"median":1,"position":"NORMAL"}}`
	cmp := `{"result":{"selectedFactor":{"code":"PER"},"selectedTics":{"displayName":"X"},"stockGraphs":[
		{"code":"NAS0250224006","name":"self","graph":[{"period":"Q1","value":10}]},
		{"code":"PEER_EMPTY","name":"empty","graph":[]}
	]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "evaluation-comparison") {
			w.Write([]byte(cmp))
		} else {
			w.Write([]byte(eval))
		}
	}))
	defer server.Close()
	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	val, err := c.GetStockValuation(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatal(err)
	}
	if len(val.Peers) != 1 {
		t.Fatalf("expected 1 peer (empty-graph skipped), got %d: %+v", len(val.Peers), val.Peers)
	}
	if val.Peers[0].ProductCode != "NAS0250224006" {
		t.Fatalf("expected self peer kept, got %q", val.Peers[0].ProductCode)
	}
}
