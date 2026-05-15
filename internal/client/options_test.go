package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGetOptionInstrumentFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "stock-info-opt-sndk-call.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/stock-infos/OPT_SNDK260515C01395000_20260506" {
			w.Write(body)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	inst, err := c.GetOptionInstrument(context.Background(), "OPT_SNDK260515C01395000_20260506")
	if err != nil {
		t.Fatalf("GetOptionInstrument error: %v", err)
	}
	if inst.UnderlyingSymbol != "SNDK" || inst.StrikePrice != 1395.0 || inst.PutCall != "CALL" {
		t.Fatalf("unexpected option instrument: %+v", inst)
	}
	if inst.OpenInterest != 82 || inst.ContractUnit != 100.0 {
		t.Fatalf("unexpected OI/contract: %+v", inst)
	}
	if inst.LiquidationDisplay != "2시간 46분 후 거래 종료" {
		t.Fatalf("unexpected liquidation display: %s", inst.LiquidationDisplay)
	}
	if !inst.PennyPilot {
		t.Fatalf("expected PennyPilot true")
	}
}

func TestGetOptionInstrumentRejectsNonOPT(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://unused"})
	_, err := c.GetOptionInstrument(context.Background(), "NAS0250224006")
	if err == nil {
		t.Fatalf("expected error for non-OPT_ productCode")
	}
}

func TestGetNearestATMOptionFromFixture(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	body := mustReadFile(t, filepath.Join(root, "option-default-chart-option-sndk.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/search/stocks":
			// NAS-prefixed codes flow through search per existing looksLikeProductCode behavior.
			w.Write([]byte(`{"result":{"stocks":[{"stockCode":"NAS0250224006","stockName":"SNDK","matchType":"EXACT"}]}}`))
		case "/api/v1/option-infos/default-chart-option":
			if r.URL.Query().Get("underlyingGuid") != "NAS0250224006" {
				t.Fatalf("unexpected underlyingGuid: %s", r.URL.Query().Get("underlyingGuid"))
			}
			w.Write(body)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})

	code, err := c.GetNearestATMOption(context.Background(), "NAS0250224006")
	if err != nil {
		t.Fatalf("GetNearestATMOption error: %v", err)
	}
	if code != "OPT_SNDK260515C01395000_20260506" {
		t.Fatalf("unexpected nearest ATM code: %s", code)
	}
}
