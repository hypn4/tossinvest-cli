# PR16 — Stock news + filings (뉴스 · 공시 tab)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add two new commands covering the 뉴스 · 공시 stock-page tab:
- `tossctl stock news <sym> [--count N]` — global news feed (Korean, curated by Toss; works for both KR and US stocks)
- `tossctl stock filings <sym> [--count N]` — KR-only filings (DART + KIND); returns empty for US stocks

Both endpoints use the companyCode (resolved via existing `resolveCompanyCode` helper) and share the same `{pagingParam, body, lastPage}` envelope.

**Architecture:** Two thin client methods + two output writers. Reuses existing patterns from PR9 (`resolveCompanyCode`). Each command fetches N pages of size 20 until N items are collected or `lastPage: true`.

**Tech Stack:** Go, cobra, `getJSON`, encoding/csv.

**Branch:** `feat/order-page-integration`. Base SHA: `5df3a01`.

---

## Endpoint shapes (verified live)

**`GET /api/v2/news/companies/{companyCode}?size=20&orderBy=latest&number=1`**:
```json
{"result": {
  "pagingParam": {"number": 2, "size": 20, "key": null},
  "body": [
    {
      "id": "ajukyung_20260516170346151",
      "title": "[종합] 고개 숙인 이재용 ...",
      "summary": "이 회장 사과 후 ...",
      "contentText": "...",
      "imageUrls": ["https://image.ajunews.com/..."],
      "source": {"code": "ajukyung", "name": "아주경제", "logoImageUrl": "..."},
      "relatedNews": [],
      "stockCodes": null,
      "stockInfo": null,
      "createdAt": "2026-05-16T17:13:25",
      "updatedAt": "2026-05-16T17:13:25"
    }
  ],
  "lastPage": false
}}
```

**`GET /api/v1/stock-detail/companies/{companyCode}/filings?number=1&size=20`** (KR only):
```json
{"result": {
  "pagingParam": {"number": 2, "size": 20, "key": null},
  "body": [
    {
      "id": "DART:A:005930-20260515002181",
      "title": "2026년 3월 확정실적 발표",
      "summary": "영업이익 57조 2,327억원, 작년보다 756% 증가",
      "companyCode": "005930",
      "stockCode": "A005930",
      "form": "EARNINGS",
      "reportId": "DART:A:005930-20260515002181",
      "earningCall": {
        "status": "ENDED",
        "landingUrl": "/earning-call/bridge/219418",
        "title": "26년 1분기 실적발표",
        "reportTitle": "AI 수요와 메모리 성장으로 최대 실적 달성",
        "liveAt": "2026-04-30T10:00:00",
        "zonedLiveAt": "2026-04-30T01:00:00Z"
      },
      "createdAt": "2026-05-15T00:00:00"
    }
  ],
  "lastPage": false
}}
```

`form` values observed: `HTML` (general announcements), `EARNINGS` (earnings releases — includes `earningCall` object).

For US stocks, the filings endpoint returns an empty `body[]`.

---

## File Structure

**New:**
- `internal/client/news.go` + `_test.go` — `ListStockNews` + `ListStockFilings`
- `internal/output/news.go` + `_test.go` — `WriteStockNews` + `WriteStockFilings`
- `fixtures/responses/public/stock-news-samsung.json` — populated KR news
- `fixtures/responses/public/stock-filings-samsung.json` — populated KR filings

**Modified:**
- `internal/domain/models.go` — append `NewsItem`, `NewsSource`, `FilingItem`, `EarningCall` types
- `cmd/tossctl/stock.go` — register `newsCmd` and `filingsCmd`
- `fixtures/responses/public/manifest.json` — append both entries
- `CHANGELOG.md` — add line

---

## Task 1: Domain + client + fixtures + tests

### Step 1: Save fixtures

Create `fixtures/responses/public/stock-news-samsung.json` (trim to 2 items):

```json
{
  "result": {
    "pagingParam": {"number": 2, "size": 20, "key": null},
    "body": [
      {
        "id": "ajukyung_20260516170346151",
        "title": "[종합] 고개 숙인 이재용 \"모두 제 탓\"...18일 노사 사후조정 재개",
        "summary": "이 회장 사과 후 노사 사후조정 재개 합의",
        "imageUrls": ["https://image.ajunews.com/content/image/2026/05/16/x.jpg"],
        "source": {"code": "ajukyung", "name": "아주경제", "logoImageUrl": "https://static.tossinvest.com/assets/image/press/ajunews-full.png"},
        "createdAt": "2026-05-16T17:13:25",
        "updatedAt": "2026-05-16T17:13:25"
      },
      {
        "id": "financial_202605161658440981",
        "title": "삼성전자 노사 대화 재개...18일 중노위 사후조정 '분수령'",
        "summary": "노사는 오는 18일 오전부터 세종시 중앙노동위원회에서 2차 사후조정 회의를 열 예정이다.",
        "imageUrls": [],
        "source": {"code": "financial", "name": "파이낸셜뉴스", "logoImageUrl": ""},
        "createdAt": "2026-05-16T16:58:44",
        "updatedAt": "2026-05-16T16:58:44"
      }
    ],
    "lastPage": false
  }
}
```

Create `fixtures/responses/public/stock-filings-samsung.json`:

```json
{
  "result": {
    "pagingParam": {"number": 2, "size": 20, "key": null},
    "body": [
      {
        "id": "KIND:12001:005930-2026057484",
        "title": "파생상품시장 안내",
        "summary": "주식선물ㆍ주식옵션 2단계 가격제한폭 확대요건 도달(하락)",
        "companyCode": "005930",
        "stockCode": "A005930",
        "form": "HTML",
        "reportId": "KIND:12001:005930-2026057484",
        "earningCall": null,
        "createdAt": "2026-05-15T00:00:00"
      },
      {
        "id": "DART:A:005930-20260515002181",
        "title": "2026년 3월 확정실적 발표",
        "summary": "영업이익 57조 2,327억원, 작년보다 756% 증가",
        "companyCode": "005930",
        "stockCode": "A005930",
        "form": "EARNINGS",
        "reportId": "DART:A:005930-20260515002181",
        "earningCall": {
          "status": "ENDED",
          "landingUrl": "/earning-call/bridge/219418",
          "title": "26년 1분기 실적발표",
          "reportTitle": "AI 수요와 메모리 성장으로 최대 실적 달성",
          "liveAt": "2026-04-30T10:00:00",
          "zonedLiveAt": "2026-04-30T01:00:00Z"
        },
        "createdAt": "2026-05-15T00:00:00"
      }
    ],
    "lastPage": false
  }
}
```

### Step 2: Update manifest

Append:

```json
{
  "file": "stock-news-samsung.json",
  "url": "https://wts-info-api.tossinvest.com/api/v2/news/companies/005930?size=20&orderBy=latest",
  "method": "GET"
},
{
  "file": "stock-filings-samsung.json",
  "url": "https://wts-info-api.tossinvest.com/api/v1/stock-detail/companies/005930/filings?number=1&size=20",
  "method": "GET"
}
```

### Step 3: Add domain types

Append to `internal/domain/models.go`:

```go
type NewsItem struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Summary   string     `json:"summary"`
	ImageURLs []string   `json:"image_urls,omitempty"`
	Source    NewsSource `json:"source"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at,omitempty"`
}

type NewsSource struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	LogoImageURL string `json:"logo_image_url,omitempty"`
}

// FilingItem is one row from the KR filings endpoint. EarningCall is set
// only when form == "EARNINGS".
type FilingItem struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary,omitempty"`
	CompanyCode string       `json:"company_code"`
	StockCode   string       `json:"stock_code"`
	Form        string       `json:"form"`     // HTML|EARNINGS|...
	ReportID    string       `json:"report_id"`
	EarningCall *EarningCall `json:"earning_call,omitempty"`
	CreatedAt   string       `json:"created_at"`
}

type EarningCall struct {
	Status      string `json:"status"`              // ENDED|UPCOMING|LIVE|...
	LandingURL  string `json:"landing_url,omitempty"`
	Title       string `json:"title,omitempty"`
	ReportTitle string `json:"report_title,omitempty"`
	LiveAt      string `json:"live_at,omitempty"`
	ZonedLiveAt string `json:"zoned_live_at,omitempty"`
}
```

### Step 4: Failing client tests

Create `internal/client/news_test.go`:

```go
package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestListStockNewsSamsung(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	news := mustReadFile(t, filepath.Join(root, "stock-news-samsung.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v2/news/companies/NAS116LTR-E0":
			w.Write(news)
		default:
			t.Fatalf("unexpected path: %s (query=%s)", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	items, err := c.ListStockNews(context.Background(), "NAS0250224006", 20)
	if err != nil {
		t.Fatalf("ListStockNews error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Source.Name != "아주경제" {
		t.Fatalf("unexpected first source: %q", items[0].Source.Name)
	}
}

func TestListStockFilingsSamsung(t *testing.T) {
	t.Parallel()
	root := fixtureRoot(t)
	overview := mustReadFile(t, filepath.Join(root, "stock-overview-sndk.json"))
	filings := mustReadFile(t, filepath.Join(root, "stock-filings-samsung.json"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/stock-infos/NAS0250224006/overview":
			w.Write(overview)
		case "/api/v1/stock-detail/companies/NAS116LTR-E0/filings":
			w.Write(filings)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(Config{HTTPClient: server.Client(), InfoBaseURL: server.URL})
	items, err := c.ListStockFilings(context.Background(), "NAS0250224006", 20)
	if err != nil {
		t.Fatalf("ListStockFilings error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[1].Form != "EARNINGS" {
		t.Fatalf("expected second item form=EARNINGS, got %q", items[1].Form)
	}
	if items[1].EarningCall == nil || items[1].EarningCall.Status != "ENDED" {
		t.Fatalf("expected earningCall.status=ENDED, got %+v", items[1].EarningCall)
	}
	if items[0].EarningCall != nil {
		t.Fatalf("expected first item (HTML) to have nil earningCall")
	}
}

func TestListStockNewsRejectsBadCount(t *testing.T) {
	t.Parallel()
	c := New(Config{InfoBaseURL: "http://localhost"})
	_, err := c.ListStockNews(context.Background(), "NAS0250224006", 0)
	if err == nil {
		t.Fatalf("expected error for count=0")
	}
}
```

### Step 5: Run test (FAIL)

Run: `go test ./internal/client/ -run "TestListStock(News|Filings)" -v`

### Step 6: Implement client

Create `internal/client/news.go`:

```go
package client

import (
	"context"
	"fmt"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type newsEnvelope struct {
	Result struct {
		Body []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Summary   string   `json:"summary"`
			ImageURLs []string `json:"imageUrls"`
			Source    struct {
				Code         string `json:"code"`
				Name         string `json:"name"`
				LogoImageURL string `json:"logoImageUrl"`
			} `json:"source"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
		} `json:"body"`
		LastPage bool `json:"lastPage"`
	} `json:"result"`
}

type filingsEnvelope struct {
	Result struct {
		Body []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			CompanyCode string `json:"companyCode"`
			StockCode   string `json:"stockCode"`
			Form        string `json:"form"`
			ReportID    string `json:"reportId"`
			EarningCall *struct {
				Status      string `json:"status"`
				LandingURL  string `json:"landingUrl"`
				Title       string `json:"title"`
				ReportTitle string `json:"reportTitle"`
				LiveAt      string `json:"liveAt"`
				ZonedLiveAt string `json:"zonedLiveAt"`
			} `json:"earningCall"`
			CreatedAt string `json:"createdAt"`
		} `json:"body"`
		LastPage bool `json:"lastPage"`
	} `json:"result"`
}

// ListStockNews returns the latest news items for a stock. Underlying endpoint
// pages 20 items at a time; this method auto-pages until `count` items are
// collected or the server reports lastPage. Accepts symbol or productCode.
func (c *Client) ListStockNews(ctx context.Context, symbol string, count int) ([]domain.NewsItem, error) {
	if count <= 0 {
		return nil, fmt.Errorf("ListStockNews: count must be > 0 (got %d)", count)
	}
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	// Pass productCode (not symbol) to skip the redundant search round-trip.
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return nil, err
	}

	const pageSize = 20
	out := make([]domain.NewsItem, 0, count)
	for page := 1; len(out) < count; page++ {
		endpoint := fmt.Sprintf("%s/api/v2/news/companies/%s?size=%d&orderBy=latest&number=%d", c.infoBaseURL, companyCode, pageSize, page)
		// First page omits number= to match Toss web; do that here for parity
		if page == 1 {
			endpoint = fmt.Sprintf("%s/api/v2/news/companies/%s?size=%d&orderBy=latest", c.infoBaseURL, companyCode, pageSize)
		}
		var env newsEnvelope
		if err := c.getJSON(ctx, endpoint, &env); err != nil {
			return nil, err
		}
		for _, n := range env.Result.Body {
			if len(out) >= count {
				break
			}
			out = append(out, domain.NewsItem{
				ID: n.ID, Title: n.Title, Summary: n.Summary, ImageURLs: n.ImageURLs,
				Source: domain.NewsSource{
					Code: n.Source.Code, Name: n.Source.Name, LogoImageURL: n.Source.LogoImageURL,
				},
				CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
			})
		}
		if env.Result.LastPage || len(env.Result.Body) == 0 {
			break
		}
	}
	return out, nil
}

// ListStockFilings returns the latest KR filings (DART + KIND) for a stock.
// Returns an empty slice for US stocks. Pagination behavior matches news.
func (c *Client) ListStockFilings(ctx context.Context, symbol string, count int) ([]domain.FilingItem, error) {
	if count <= 0 {
		return nil, fmt.Errorf("ListStockFilings: count must be > 0 (got %d)", count)
	}
	productCode, err := c.resolveProductCode(ctx, symbol)
	if err != nil {
		return nil, err
	}
	companyCode, err := c.resolveCompanyCode(ctx, productCode)
	if err != nil {
		return nil, err
	}

	const pageSize = 20
	out := make([]domain.FilingItem, 0, count)
	for page := 1; len(out) < count; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/stock-detail/companies/%s/filings?number=%d&size=%d", c.infoBaseURL, companyCode, page, pageSize)
		var env filingsEnvelope
		if err := c.getJSON(ctx, endpoint, &env); err != nil {
			return nil, err
		}
		for _, f := range env.Result.Body {
			if len(out) >= count {
				break
			}
			it := domain.FilingItem{
				ID: f.ID, Title: f.Title, Summary: f.Summary,
				CompanyCode: f.CompanyCode, StockCode: f.StockCode,
				Form: f.Form, ReportID: f.ReportID, CreatedAt: f.CreatedAt,
			}
			if f.EarningCall != nil {
				it.EarningCall = &domain.EarningCall{
					Status: f.EarningCall.Status,
					LandingURL: f.EarningCall.LandingURL,
					Title: f.EarningCall.Title,
					ReportTitle: f.EarningCall.ReportTitle,
					LiveAt: f.EarningCall.LiveAt,
					ZonedLiveAt: f.EarningCall.ZonedLiveAt,
				}
			}
			out = append(out, it)
		}
		if env.Result.LastPage || len(env.Result.Body) == 0 {
			break
		}
	}
	return out, nil
}
```

### Step 7: Run tests (PASS — all 3 subtests)

### Step 8: Commit

```bash
git add internal/domain/models.go internal/client/news.go internal/client/news_test.go fixtures/responses/public/stock-news-samsung.json fixtures/responses/public/stock-filings-samsung.json fixtures/responses/public/manifest.json
git commit -m "feat(client): add ListStockNews + ListStockFilings (paginated)"
```

---

## Task 2: Output writers + cobra + CHANGELOG

### Step 1: Failing output tests

Create `internal/output/news_test.go`:

```go
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
```

### Step 2: Run test (FAIL)

### Step 3: Implement output writers

Create `internal/output/news.go`:

```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

const dateColumnWidth = 10 // YYYY-MM-DD

// WriteStockNews renders a news feed. Table mode shows date / source / title
// (truncated to 80 runes per row). CSV emits the full structure (one row per
// item). JSON dumps the full slice.
func WriteStockNews(w io.Writer, format Format, items []domain.NewsItem) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"id", "created_at", "title", "summary", "source_code", "source_name"}); err != nil {
			return err
		}
		for _, n := range items {
			if err := cw.Write([]string{n.ID, n.CreatedAt, n.Title, n.Summary, n.Source.Code, n.Source.Name}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if len(items) == 0 {
			_, err := fmt.Fprintln(w, "No news found.")
			return err
		}
		headers := []string{"DATE", "SOURCE", "TITLE"}
		rows := make([][]string, len(items))
		for i, n := range items {
			date := n.CreatedAt
			if len(date) > dateColumnWidth {
				date = date[:dateColumnWidth]
			}
			rows[i] = []string{date, n.Source.Name, truncateName(n.Title, 80)}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteStockFilings renders KR filings (DART + KIND). EARNINGS form items get
// a follow-on row showing the earning-call title and status.
func WriteStockFilings(w io.Writer, format Format, items []domain.FilingItem) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case FormatCSV:
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"id", "created_at", "form", "title", "summary", "earning_call_status", "earning_call_title"}); err != nil {
			return err
		}
		for _, f := range items {
			callStatus, callTitle := "", ""
			if f.EarningCall != nil {
				callStatus = f.EarningCall.Status
				callTitle = f.EarningCall.Title
			}
			if err := cw.Write([]string{f.ID, f.CreatedAt, f.Form, f.Title, f.Summary, callStatus, callTitle}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case FormatTable:
		if len(items) == 0 {
			_, err := fmt.Fprintln(w, "No filings found. (US stocks typically have no Korean filings; check news.)")
			return err
		}
		headers := []string{"DATE", "FORM", "TITLE"}
		rows := make([][]string, 0, len(items)*2)
		for _, f := range items {
			date := f.CreatedAt
			if len(date) > dateColumnWidth {
				date = date[:dateColumnWidth]
			}
			rows = append(rows, []string{date, f.Form, truncateName(f.Title, 80)})
			if f.EarningCall != nil {
				rows = append(rows, []string{"", "↳ 어닝콜", fmt.Sprintf("%s (%s)", f.EarningCall.Title, f.EarningCall.Status)})
			}
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
```

### Step 4: Run test (PASS)

### Step 5: Register cobra subcommands

In `cmd/tossctl/stock.go`, after `ratiosCmd`:

```go
var newsCount int
newsCmd := &cobra.Command{
    Use:   "news <symbol>",
    Short: "Latest news for a stock (Korean, curated by Toss)",
    Long: `Fetch the latest news items from /api/v2/news/companies/{companyCode}.

Returns Korean-language news from Toss's curated press partners (아주경제,
파이낸셜뉴스, 이데일리, Benzinga via translation, etc.). Works for both KR
and US stocks. Page size is 20; --count controls total items returned.

Examples:
  tossctl stock news A005930
  tossctl stock news SNDK --count 50
  tossctl stock news AAPL --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        items, err := app.client.ListStockNews(cmd.Context(), args[0], newsCount)
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockNews(cmd.OutOrStdout(), app.format, items)
    },
}
newsCmd.Flags().IntVar(&newsCount, "count", 20, "Number of news items to return (max ~100; auto-paginates)")

var filingsCount int
filingsCmd := &cobra.Command{
    Use:   "filings <symbol>",
    Short: "KR filings (DART + KIND) for a stock; empty for US stocks",
    Long: `Fetch the latest KR regulatory filings from
/api/v1/stock-detail/companies/{companyCode}/filings.

Includes DART disclosures (form=EARNINGS includes 어닝콜 metadata),
KIND announcements (form=HTML, 파생상품시장 안내 etc.), and similar
filings. US stocks return an empty list (no Korean filings); use 'stock
news' for international press coverage.

Examples:
  tossctl stock filings A005930
  tossctl stock filings 005930 --count 50
  tossctl stock filings A005930 --output json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        app, err := newAppContext(opts)
        if err != nil {
            return err
        }
        items, err := app.client.ListStockFilings(cmd.Context(), args[0], filingsCount)
        if err != nil {
            return userFacingCommandError(err)
        }
        return output.WriteStockFilings(cmd.OutOrStdout(), app.format, items)
    },
}
filingsCmd.Flags().IntVar(&filingsCount, "count", 20, "Number of filings to return (auto-paginates)")
```

Update `cmd.AddCommand`:

```go
cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd, financialsCmd, dividendsCmd, estimatesCmd, statementsCmd, ratiosCmd, newsCmd, filingsCmd)
```

### Step 6: CHANGELOG entry

In `CHANGELOG.md`, after the `stock ratios` line, add:

```markdown
- `tossctl stock news <symbol> [--count N]` — latest news for a stock (Korean, curated by Toss) via `GET /api/v2/news/companies/{companyCode}`. Auto-paginates 20/page.
- `tossctl stock filings <symbol> [--count N]` — KR filings (DART disclosures + KIND announcements + 어닝콜 metadata for EARNINGS form) via `GET /api/v1/stock-detail/companies/{companyCode}/filings`. US stocks return empty.
```

### Step 7: Full build + test

Run: `go build ./... && go vet ./... && go test ./...`

### Step 8: Commit

```bash
git add internal/output/news.go internal/output/news_test.go cmd/tossctl/stock.go CHANGELOG.md
git commit -m "feat(cli): add tossctl stock news + filings (뉴스 · 공시 tab)"
```

---

## Task 3: Live verification + push

- [ ] `go build -o /tmp/tossctl ./cmd/tossctl` succeeds.
- [ ] `/tmp/tossctl stock news SNDK --count 5` prints 5 news items.
- [ ] `/tmp/tossctl stock news A005930` prints Korean Samsung news.
- [ ] `/tmp/tossctl stock filings A005930 --count 10` prints DART/KIND filings; at least one EARNINGS form with 어닝콜 row.
- [ ] `/tmp/tossctl stock filings SNDK` prints `No filings found.` message.
- [ ] `/tmp/tossctl stock news A005930 --output json | head -20` confirms snake_case keys (`created_at`, `logo_image_url`).
- [ ] `git push origin feat/order-page-integration`.

---

## Self-Review

**Spec coverage:**
- 2 endpoints (news + filings), both paginated ✅
- snake_case JSON tags ✅
- encoding/csv for CSV ✅
- Auto-pagination via `count` flag ✅
- Empty-case handling (filings on US stocks) ✅
- EarningCall nested object surfaced ✅
- CHANGELOG entry ✅

**Placeholder scan:** none.

**Type consistency:** Domain types snake_case, wire types camelCase. EarningCall is `*EarningCall` (nullable) — only set when form=EARNINGS.
