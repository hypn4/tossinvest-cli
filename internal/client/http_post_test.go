package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJSONEmptySendsEmptyBody(t *testing.T) {
	t.Parallel()
	var gotMethod, gotCT string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"result":{"ok":true}}`))
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	var env struct {
		Result struct {
			OK bool `json:"ok"`
		} `json:"result"`
	}
	if err := c.postJSONEmpty(context.Background(), server.URL+"/anything", &env); err != nil {
		t.Fatalf("postJSONEmpty error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotCT != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", gotCT)
	}
	if string(gotBody) != "{}" {
		t.Fatalf("expected body {}, got %q", string(gotBody))
	}
	if !env.Result.OK {
		t.Fatalf("expected env.Result.OK to be true")
	}
}
