# Stock deep-tab CLI — phased rollout (PR9 → PR12)

Author: hypn4
Date: 2026-05-16
Branch: `feat/order-page-integration`

## Goal

Wire the remaining 종목정보 deep-tab endpoints (already documented in
`docs/reverse-engineering/rpc-catalog.md`) to first-class `tossctl stock <subcmd>`
commands. After PR8 closed the company-overview gap, eight planned commands
remain. This spec rolls them out in four phases, each its own PR.

Out of scope (per `memory/options-scope-read-only.md`):
- Any option ordering or account-creation flow.
- Authenticated comment / memo / like / watch list mutation.

## PR sequencing

| PR | Command(s) | Endpoint(s) | Capture status |
|---|---|---|---|
| **PR9** | `stock indicators` | `/api/v1/stock-detail/ui/wts/{code}/investment-indicators` | captured |
| | `stock valuation` | `/api/v2/stock-infos/evaluation/{code}` (POST) + `/api/v2/stock-infos/evaluation-comparison/{code}` (POST) | captured |
| | `stock revenue` | `/api/v1/companies/{companyCode}/sales-compositions` | captured |
| | `stock peers` | `/api/v2/companies/{companyCode}/tics` | captured |
| | `stock analyst` | `/api/v1/stock-detail/ui/wts/{code}/analyst-opinion` + `/api/v2/stock-infos/consensus/{code}` + `/api/v1/stock-detail/ui/wts/{code}/analyst-reports` | captured |
| **PR10** | `stock financials` | `/api/v2/stock-infos/revenue-and-net-profit/{code}` (POST) + `/api/v2/stock-infos/operating-income/{code}` (POST) + `/api/v2/stock-infos/stability/{code}` (POST) + `/api/v2/companies/{code}/financial-statements/comprehensive` (POST) + `/api/v2/companies/{code}/financial-statement-records` (POST) | **fresh capture needed** |
| **PR11** | `stock dividends` | `/api/v1/stock-infos/dividend/{code}/years` + `/api/v1/stock-infos/dividend/{code}/summary` + `/api/v1/stock-infos/{code}/dividends/yield-ratio/histories` | **fresh capture needed** |
| **PR12** | `stock estimates` | `/api/v2/companies/{code}/financial/estimate/date` + `/api/v2/companies/{code}/financial/estimate/revenue` (POST) + `/api/v2/companies/{code}/financial/estimate/eps` (POST) + `/api/v2/companies/{code}/financial/estimate/operating-income` (POST) | **fresh capture needed** |

PRs land in order. Each PR is one merge into `feat/order-page-integration`,
pushed to `origin` (fork) only — no upstream push.

## PR9 detail (captured groups)

### 9a — `tossctl stock indicators <sym>`

Endpoint: `GET /api/v1/stock-detail/ui/wts/{code}/investment-indicators`

Response: `result.indicatorSections[]` with `sectionName ∈ {가치평가, 수익, 배당, 안정성}` and per-section `data` dict.

Domain type: `StockIndicators { Valuation, Earnings, Dividend, Stability *IndicatorSection }` where each `IndicatorSection` is the raw key/value map for that section.

Table layout:

```
샌디스크 — SNDK (NAS0250224006)
=== 가치평가 ===
PER  45.4배    PBR  14.9배    PSR  15.5배

=== 수익 ===
EPS  $28.76 (₩42,904)
BPS  $93.08 (₩138,856)
ROE  39.3%

=== 배당 ===
배당 주기      —
배당 수익률    0.00%
연간 배당금    —

=== 안정성 ===
(captured fields, e.g. liabilityToEquityDisplay)
```

### 9b — `tossctl stock valuation <sym>`

Two endpoints, POST with empty body `{}`.

Aggregated domain type:
```go
type StockValuation struct {
    PER, PBR, PSR    float64
    Median           float64   // industry median for `selectedFactor`
    Position         string    // HIGH | LOW | NORMAL
    PeerFactor       string    // PER (default)
    Peers            []PeerValuation
}
type PeerValuation struct { Code, Name string; PER, PBR, PSR float64 }
```

Table:

```
샌디스크 — SNDK
PER 45.4배 vs 업종 중앙값 26.5배   →  HIGH
PBR 14.9 / PSR 15.5 동반 표기

동종업계 비교 (PER 기준)
SYMBOL  PER    PBR    PSR
SNDK   45.4   14.9   15.5  ← 본인
…
```

### 9c — `tossctl stock revenue <sym>`

Endpoint: `GET /api/v1/companies/{companyCode}/sales-compositions`.

`companyCode` (e.g. `NAS116LTR-E0`) must be resolved from the `overview` payload first — store it on the Client or pass through. Add a private helper `(c *Client) resolveCompanyCode(ctx, sym) (string, error)` that internally calls `GetCompanyOverview` and returns `result.code` (the company-level code, distinct from productCode).

Domain type:
```go
type SalesComposition struct {
    CompanyCode, DataSource string
    FiscalYear              int
    EndDate                 string
    Items                   []SalesCompositionItem
}
type SalesCompositionItem struct { Business, Product string; Ratio float64 }
```

Table:

```
샌디스크 — SNDK
매출 구성 (FY2025, ending 2025-06-30)
BUSINESS         RATIO
클라이언트       56.11%
소비자           30.84%
클라우드 서비스  13.05%
출처: 연합인포맥스 및 기업 IR자료
```

### 9d — `tossctl stock peers <sym>`

Endpoint: `GET /api/v2/companies/{companyCode}/tics`. Same companyCode resolution as 9c.

Domain:
```go
type TICSIndustry struct {
    BaseDate string
    Major    []TICSEntry   // depth-1 industries
    Minor    []TICSEntry   // depth-2 industries (subset of major)
}
type TICSEntry struct {
    ID                       int
    Title, Description       string
    CompanyCount             int
    Representative           bool
    Rankings                 []TICSPeerRanking
}
type TICSPeerRanking struct { BaseDate string; ProductCode, Name string; Rank int; Score float64 }
```

Table (default = major industry only, top 10 peers):

```
샌디스크 — SNDK
TICS 산업: 컴퓨터와 주변기기 (id 209, 85개사)
"sd카드, usb 메모리 등 판매"

랭킹 (2026-05-16 기준)
RANK   SYMBOL    NAME           ...
1      SNDK      샌디스크       ...
2      ...
```

`--all` flag to also dump minorList. JSON output always dumps everything.

### 9e — `tossctl stock analyst <sym>`

Three endpoints stitched:
- `/api/v1/stock-detail/ui/wts/{code}/analyst-opinion` (counts + target price)
- `/api/v2/stock-infos/consensus/{code}` (mean/high/low target + past closes)
- `/api/v1/stock-detail/ui/wts/{code}/analyst-reports` (report list)

Domain:
```go
type AnalystSnapshot struct {
    Opinion AnalystOpinion
    Consensus ConsensusTarget
    Reports []AnalystReport
}
type AnalystOpinion struct {
    Type                                      string // BUY|HOLD|SELL
    StrongBuy, Buy, Hold, Sell, StrongSell    int
    TargetUSD, TargetKRW                      float64
    Description                               string
}
type ConsensusTarget struct {
    Mean, High, Low, MeanKrw, HighKrw, LowKrw float64
    Currency, PointDate                       string
    PastCloses                                []ConsensusPastClose
}
type ConsensusPastClose struct { Date string; Price, PriceKrw float64 }
type AnalystReport struct { Title, Source, Date string; URL string }
```

Table:

```
샌디스크 — SNDK
의견: BUY  (애널리스트 22명 중 18명이 구매 의견)
strongBuy 6  buy 12  hold 4  sell 0  strongSell 0

컨센서스 목표가
mean    $1,224.42  (₩1,794,999)
high    $2,000.00
low       $250.00

목표가 vs 과거 종가
2026-05-15  $1,405.85
2026-04-30  $1,096.51

애널리스트 보고서: 0건 (이번 캡쳐 시점)
```

## PR10–12 detail (fresh-capture required)

Each follows the same pattern as PR9 commands but is preceded by a
chrome-devtools session:

1. Open Toss web with the stock-info deep tab open for SNDK.
2. Trigger the relevant section (scroll into view, click expand if any).
3. Dump network requests, save responses to `.captures/2026-05-16/sndk-stock-info/`.
4. Update `rpc-catalog.md` with verified shapes.
5. Trim and commit fixture JSON to `fixtures/responses/public/`.
6. Run TDD per command spec.

### PR10 — `stock financials <sym>`

Combines five POST endpoints. Output sections:
- **Revenue & Net profit** (quarterly + annual)
- **Operating income** (quarterly + annual)
- **Stability** (부채비율 / 유동비율 by year + quarter)
- **Financial statements (comprehensive)** — multi-period IS/BS/CF summary
- **Financial-statement records** — raw line items (`--format json` only by default; suppress in table view)

Flag: `--period quarter|annual|both` (default `both`).

### PR11 — `stock dividends <sym>`

Three GET endpoints. Output:
- **Summary** card: yield, frequency, last payout date, annual cash
- **Per-year payouts** table
- **Yield ratio history** sparkline (optional / `--with-history`)

### PR12 — `stock estimates <sym>`

`estimate/date` (GET) gates the three POST endpoints. Output:
- **Consensus pointDate**
- **Revenue estimates** by `valueType` (31=annual, 32/42=quarterly variants)
- **EPS estimates**
- **Operating-income estimates**

Flag: `--years N` to truncate.

## Implementation pattern (per command)

Mirrors PR8:

1. `internal/domain/models.go` — append types.
2. `internal/client/<name>.go` — new file with one or more public methods + private envelope structs. `httptest.Server` + fixture-driven test in `<name>_test.go`.
3. `internal/output/<name>.go` — `Write<Name>(w, format, value)`. Three branches: Table / JSON / CSV. Table golden via `_test.go`.
4. `cmd/tossctl/stock.go` — append `<name>Cmd := &cobra.Command{...}` and wire into `cmd.AddCommand(...)`.
5. `fixtures/responses/public/` — add response JSON, update `manifest.json`.
6. `CHANGELOG.md` — one line under `## Unreleased`.

Each command is committed as a single PR-internal merge to keep PR-history clean.

## Output conventions

- `--output table` (default): human-readable, mirrors the relevant Toss web tab.
- `--output json`: full domain struct, suitable for LLM pipelines.
- `--output csv`: flat tabular subset (skip nested sections; emit primary table only).

Common numeric formatting:
- USD: `$1,224.42` (2 decimals, comma separators)
- KRW: `₩1,794,999` (no decimals)
- Percent: `45.4%`

## Resolver helpers

Two new private helpers on `Client`:
- `resolveCompanyCode(ctx, sym) (string, error)` — calls `GetCompanyOverview` and returns `result.code` (company code, NOT productCode). Used by `stock revenue`, `stock peers`, and PR10/PR12 financial-statements/records endpoints.
- `postJSONEmpty(ctx, url, dst)` — helper for the many `POST {}` endpoints (evaluation, revenue-and-net-profit, …). Same signature as `getJSON`.

These join existing `resolveProductCode` (which converts symbol → productCode).

## Testing strategy

Per-PR:
- Unit: `httptest.Server` for client, golden file for output writer.
- Build + vet + test: `go build ./... && go vet ./... && go test ./...` must pass.
- Live verification: run the new command against the real API for SNDK after binary build; record output in CHANGELOG/PR description.

Cross-PR: PR9 establishes the test pattern; PR10–12 follow it without
introducing new test infrastructure.

## Migration / breaking changes

None. All new commands; no existing command surface or output format is
modified. CHANGELOG entries are pure `### Added`.

## Risks

- **`companyCode` ≠ `productCode`** — easy to mix up. Mitigate via the
  dedicated `resolveCompanyCode` helper and explicit doc comments on
  affected client methods.
- **POST endpoints with empty body** — verify they 200 with `{}` and not
  `nil`/no body via real captures before shipping.
- **Field availability varies by listing** — KR vs US stocks expose
  different indicator subsets. Output writers must tolerate missing
  sections (return "—" rather than panic).
- **`analyst-reports` returned empty** in PR9 capture. Real test data may
  have entries; design must handle non-empty arrays gracefully.

## Open questions

None at spec time. PR10–12 endpoint shapes will be confirmed during fresh
capture (assumed valid based on rpc-catalog documentation but unverified).
