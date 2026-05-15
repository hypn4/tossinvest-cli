package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/junghoonkye/tossinvest-cli/internal/session"
)

func orderableFixtureRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "fixtures", "responses", "public")
}

func TestGetOrderableSummary(t *testing.T) {
	t.Parallel()

	root := orderableFixtureRoot(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/dashboard/common/cached-orderable-amount", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "orderable-amount.json"))
	})
	mux.HandleFunc("/api/v3/my-assets/transactions/markets/us/overview", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "transactions-overview-us.json"))
	})
	mux.HandleFunc("/api/v3/my-assets/transactions/markets/kr/overview", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(root, "transactions-overview-kr.json"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	sess := session.Session{Cookies: map[string]string{"SESSION": "test"}}
	c := New(Config{
		HTTPClient:  server.Client(),
		APIBaseURL:  server.URL,
		CertBaseURL: server.URL,
		Session:     &sess,
	})
	summary, err := c.GetOrderableSummary(context.Background())
	if err != nil {
		t.Fatalf("GetOrderableSummary returned error: %v", err)
	}
	if summary.OrderableKR.KRW != 110421 {
		t.Fatalf("unexpected KR orderable KRW: %v", summary.OrderableKR.KRW)
	}
	if summary.OrderableUS.USD != 508.71 {
		t.Fatalf("unexpected US orderable USD: %v", summary.OrderableUS.USD)
	}
	if summary.US.Market != "us" {
		t.Fatalf("US overview missing")
	}
	if summary.KR.Market != "kr" {
		t.Fatalf("KR overview missing")
	}
}
