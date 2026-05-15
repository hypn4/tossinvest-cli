# Order Page Deep Dive

Verified 2026-05-15 KST from `https://www.tossinvest.com/stocks/US20100311002/order` (SOXL — Direxion Daily Semiconductor Bull 3x, AMEX ETF) using Chrome DevTools MCP with an authenticated session. Extended capture window included the home dashboard (`/`) and the Samsung Electronics (`A005930`) quote endpoints for KR-market comparisons.

This document complements [`rpc-catalog.md`](rpc-catalog.md) and [`../trading/rpc-catalog.md`](../trading/rpc-catalog.md). The focus is the three axes most useful for LLM/agent integrations:

- **Chart candles** — supported timeframes, session filtering, pagination
- **Realtime quote · orderbook · ticks** — REST polling vs SSE, market depth differences (KR 10-level vs US top-of-book)
- **Auxiliary LLM context** — Toss AI signals, news, related stocks, full holdings, indices

## Capture metadata

- Capture window: 2026-05-15 21:09–21:22 KST (US pre-market session)
- Subject: SOXL (`US20100311002`) order page → home (`/`) for portfolio + AI signal capture → Samsung (`A005930`) for KR orderbook
- Tooling: `chrome-devtools` MCP, `evaluate_script` for vocabulary sweeps
- Raw responses (gitignored, kept for reproducibility): `.captures/2026-05-15/us-soxl/*.network-response`
- Validation scope: 90 `stepUnit` combinations enumerated, all `session=` values probed, both KR and US orderbook depth compared

This file is the canonical write-up. The raw `.network-response` payloads live outside git but are referenced inline where the JSON shape matters. Endpoint rows promoted to the catalog live in [`rpc-catalog.md`](rpc-catalog.md).

---

## 1. 전체 구조 한눈에

```
브라우저
  ├─ wts-info-api.tossinvest.com   ← public/익명 가능한 시세·정보 (가장 많이 호출됨)
  ├─ wts-api.tossinvest.com        ← 인증 필요한 사용자/세션/설정
  ├─ wts-cert-api.tossinvest.com   ← 인증 필요한 트레이딩/포지션/주문/분석
  ├─ wts-lc.tossinvest.com         ← 로그/펌프링 (수집용, 무시 가능)
  └─ sse-message.tossinvest.com    ← Server-Sent Events 푸시 (얇은 알림 채널)
```

- **WebSocket 없음.** 모든 푸시는 SSE 1개 채널로 통합되며, 페이로드는 "재조회하라" 신호만 전달함.
- **인증.** `wts-cert-api` 호스트는 세션 쿠키(`SESSION`, `UTK`, `LTK`, `FTK`, `BTK`, `XSRF-TOKEN`, `deviceId`, `browserSessionId`) 가 모두 필요. `wts-info-api`/`wts-api` 일부 경로는 익명 접근 가능 (시세 조회 등).
- **공통 헤더.** `app-version`, `browser-tab-id`, `Origin: https://www.tossinvest.com`, `Accept: application/json`. 일부 인증 API 는 `tossinvest-event-id`, `x-toss-os` 도 함께 전송.

페이지 진입 시 ~80개의 fetch/XHR 가 한 번에 발생하고, 그 중 LLM 컨텍스트로 유의미한 데이터는 ~15개 엔드포인트에 집중됨.

---

## 2. 차트 캔들 (Chart Candles)

### 2.1 엔드포인트 패턴

```
GET https://wts-info-api.tossinvest.com/api/v1/c-chart/{product}/{code}/{stepUnit}
    ?count=<int>
    [&from=<ISO8601 with TZ, URL-encoded>]
    [&useAdjustedRate=true]
```

- `tossinvest-path-pattern: /api/v1/c-chart/{product}/{code}/{stepUnit}` 응답 헤더로 토스 라우팅 패턴이 노출됨.
- 인증 불필요(공개) — 익명 쿠키 없이 호출 시도 검증 필요하나 응답 자체는 사용자 정보 미포함.
- `content-encoding: zstd` (브라우저는 자동 해제. CLI 클라이언트는 `Accept-Encoding: gzip` 만 보내도 무방).

| 파라미터 | 값 (관찰) | 의미 |
| --- | --- | --- |
| `product` | `us-s` (미국 주식/ETF), `kr-s` (한국 주식) | 시장 prefix |
| `code` | `US20100311002` (SOXL) | 종목 GUID |
| `stepUnit` | `day:1`, `min:30` | 단위 (`day`/`min`/`week`/`month` 추정) + `:`+step |
| `count` | 1, 100, 90, 248, 450 | 가져올 캔들 수 |
| `from` | `2026-05-03T23:30:00-04:00` (URL-encoded) | 페이지네이션 — 이 시점보다 과거 캔들 |
| `useAdjustedRate` | `true` | 액면분할/배당 조정 가격 사용 |

관찰된 호출:

- `day:1?count=100&useAdjustedRate=true` — 일봉 초기 로드 100개
- `day:1?count=1&useAdjustedRate=true` — 가장 최근 1개 (현재가 갱신용)
- `min:30?count=450&useAdjustedRate=true` — 30분봉 450개 (~10 거래일치)
- `min:30?count=90&from=2026-05-03T23:30:00-04:00&useAdjustedRate=true` — 페이지네이션
- `min:30?count=450&from=2026-04-30T02:30:00-04:00&useAdjustedRate=true` — 더 과거 페이지

### 2.2 응답 구조

```jsonc
{
  "result": {
    "code": "US20100311002",
    "nextDateTime": "2025-12-19T00:00:00-05:00",   // ← 더 과거 페이지 요청용 커서
    "exchangeRate": 1491.8,                         // ← 응답 시점 USD→KRW 환율
    "candles": [
      // 일봉 (sessionType 없음)
      { "dt": "2026-05-15T00:00:00-04:00", "base": 186.19, "open": 180.61,
        "high": 180.95, "low": 166.3, "close": 168.6,
        "volume": 1731781, "amount": 298116203 },

      // 30분봉 (sessionType 포함)
      { "dt": "2026-05-15T08:30:00-04:00", "sessionType": "pre",
        "base": 186.19, "open": 169.62, "high": 169.84, "low": 168.13,
        "close": 168.6, "volume": 102848, "amount": 0 }
    ]
  }
}
```

| 필드 | 의미 |
| --- | --- |
| `dt` | 캔들 시작 시각 (ISO 8601 + 시장 현지 타임존 — US 는 `-04:00`/`-05:00`) |
| `base` | 전일 종가/기준가 (등락률 산정 기준) |
| `open`/`high`/`low`/`close` | OHLC (시장 현지 통화) |
| `volume` | 거래량 (주) |
| `amount` | 거래대금 (시장 현지 통화 — 일봉은 채워지고 분봉은 `0` 으로 비어 있음) |
| `sessionType` | 분봉/주간봉 한정. 관찰값 분포 (450개 중): `day`=153, `main`=117, `pre`=108, `after`=72 |

**sessionType 의미 추정** (관찰한 시간대를 기준으로):

- `pre` — 미국 프리마켓 (US ET 04:00–09:30)
- `main` — 미국 정규장 (US ET 09:30–16:00)
- `after` — 미국 애프터마켓 (US ET 16:00–20:00)
- `day` — 토스 **주간거래** (한국 시간 낮 시간대에 매칭되는 별도 세션, US ET 22:00 다음날 04:00 부근에 활성화되는 토스 데이터임)

LLM 이 캔들을 다룰 때는 `sessionType` 으로 정규장만 필터(`main`)하거나, 전체 세션을 통합한 OHLCV 합성 시리즈를 사용할지 결정해야 한다.

### 2.3 페이지네이션 / 환율

- `result.nextDateTime` = 응답에서 가장 오래된 캔들 시각. 더 과거를 받으려면 `from` 에 이 값을 그대로 URL-encode 해서 다음 요청.
- `result.exchangeRate` = 응답 직전 시점의 USD/KRW. 캔들 자체엔 KRW 환산 필드가 없으므로 KRW 환산이 필요하면 `close * exchangeRate` 로 계산하거나, 별도로 `/api/v3/stock-prices/details` 의 `closeKrw`/`baseKrw` 등을 사용.

---

## 3. 실시간 시세 · 호가 · 체결

### 3.1 채널 모델

토스증권 웹은 **WebSocket을 쓰지 않고**, 다음 두 가지로 "실시간"을 구현한다:

1. **SSE 1개 채널** (`/api/v1/wts-notification`) — "뭔가 바뀌었으니 다시 가져가라" 형식의 thin notification
2. **REST 폴링** — 위 신호를 받았거나 사용자 인터랙션이 발생하면 시세 REST 엔드포인트를 재호출

> 따라서 LLM/봇 입장의 실시간 데이터 = `(SSE 트리거) + (재조회 REST)` 또는 단순 **N초 폴링**.

### 3.2 핵심 시세 REST 엔드포인트

| 메서드 | URL | 갱신 빈도(관찰) | 인증 | 용도 |
| --- | --- | --- | --- | --- |
| GET | `wts-info-api.tossinvest.com/api/v3/stock-prices/details?productCodes={code}` | 페이지 진입 시 4회 호출됨 | 공개 | **단일 종목 풀 시세 (현재가+OHLC+시총+체결강도+pre/after)** |
| GET | `wts-info-api.tossinvest.com/api/v3/stock-prices?meta=true&productCodes={code}` | 1회 | 공개 | 경량 가격 + 거래 세션 시각 |
| GET | `wts-info-api.tossinvest.com/api/v1/product/stock-prices?meta=true&productCodes=c1,c2,...` | 1회 | 공개 | **다중 종목 배치 가격 조회** |
| GET | `wts-info-api.tossinvest.com/api/v3/stock-prices/{code}/quotes` | 1회 | 공개 | **호가창 (Top-of-book)** |
| GET | `wts-info-api.tossinvest.com/api/v2/stock-prices/{code}/ticks?count=<N>` | 페이지 진입 시 50개, 차트 펼친 후 120개 | 공개 | **체결 틱 스트림 (스냅샷)** |

### 3.3 시세 상세 — `/api/v3/stock-prices/details`

```json
{
  "result": [{
    "code": "US20100311002",
    "tradeDateTime": "2026-05-15T12:09:40Z",
    "open": 180.61, "high": 180.95, "low": 166.30, "close": 168.60,
    "volume": 1731781,            // 정규장 누적 (관찰 시점 pre 진행 중이므로 전일 마감 분량으로 추정)
    "value": 298116203,            // 거래대금 (USD)
    "base": 186.19,                // 전일 종가
    "changeType": "DOWN",
    "currency": "USD",
    "high52w": 191.28, "low52w": 15.10,
    "high1y": 191.28,  "low1y": 15.10,
    "marketCap": 20763090000,
    "tradingStrength": 95.99,      // 체결강도 = 누적매수 / 누적매도 × 100
    "preDayVolume": 43318951,
    "afterMarketOpen": 186.19,  "afterMarketHigh": 186.19,
    "afterMarketLow": 186.19,   "afterMarketClose": 186.19,
    "openKrw": 269433, "highKrw": 269941, "lowKrw": 248086, "closeKrw": 251517,
    "baseKrw": 277758, "high52wKrw": 285351, "low52wKrw": 22526,
    "valueKrw": 444729751635,
    "afterMarketOpenKrw": 277758, "afterMarketCloseKrw": 277758,
    "closeKrwDecimal": 251517.48, "baseKrwDecimal": 277758.242
  }]
}
```

- LLM 이 "현재가/등락/거래량/체결강도" 만 알면 되는 경우 이 엔드포인트 하나로 충분.
- `tradingStrength` (체결강도) — 95.99 = 매도세 우위. > 100 이면 매수세 우위. 토스의 시장심리 지표.
- `marketCap` 은 시장통화(USD) 기준.
- KRW 환산 필드가 풍부하지만, 결제일 환율과 실시간 환율의 미세한 시점차로 인해 `*KrwDecimal` 변이가 미세하게 차이남.

### 3.4 호가창 — `/api/v3/stock-prices/{code}/quotes`

```json
{
  "result": {
    "close": 168.60, "closeKrw": 251517,
    "offerPrices": [168.79], "offerPricesKrw": [251800], "offerVolumes": [7],
    "bidPrices":   [168.60], "bidPricesKrw":   [251517], "bidVolumes":   [2],
    "offerVolume": 7,  "bidVolume": 2
  }
}
```

- 관찰 시점 (US 프리마켓 직전) 에는 **Top-of-book 1단계만 반환**. 한국장이나 정규장에선 다단 호가가 노출될 가능성이 높음 (배열 길이로 표현). 기존 docs/trading 캡쳐에선 같은 패턴이지만 다단 발견은 안 됨.
- `offerVolume` / `bidVolume` 은 호가 잔량 합계.

### 3.5 체결 틱 — `/api/v2/stock-prices/{code}/ticks?count=N`

```json
{
  "result": [
    {
      "time": "21:09:40",           // KST(?) HH:MM:SS, 날짜 없음
      "code": "US20100311002",
      "price": 168.60, "priceKrw": 251517,
      "base": 186.19,  "baseKrw": 277758,
      "volume": 2,                   // 해당 체결 수량
      "tradeType": "SELL",            // BUY = 매수 체결, SELL = 매도 체결
      "cumulativeVolume": 1731781    // 누적 거래량 (감소 방향으로 정렬되어 있음)
    }
  ]
}
```

- 응답은 **최근 → 과거** 시간 역순.
- `count` 의 상한은 미확인. 페이지에선 50/120 만 관찰.
- `time` 에 날짜가 없어 자정 경계가 모호함 — `cumulativeVolume` 단조 감소를 이용해 정렬·중복 제거.
- 누적거래량 차분(volume 합산 ≈ cumulativeVolume 차)으로 신뢰성 검증 가능.
- **연속 폴링** 으로 신규 틱만 잘라내려면 `time + price + cumulativeVolume` 조합을 트랜잭션 키로 사용.

### 3.6 SSE 푸시 채널 — `/api/v1/wts-notification`

```
GET https://sse-message.tossinvest.com/api/v1/wts-notification
Accept: text/event-stream
Cache-Control: no-cache
Cookie: (전체 인증 쿠키 셋)
```

- 응답 `content-type: text/event-stream;charset=UTF-8`, `connection: close` (HTTP/1.1 chunked).
- 페이지 진입 시 2회 연결되었는데, 첫 연결은 서버가 `event: connection-close` 핸드오프 후 클라이언트가 재연결 — 이 거동은 `docs/reverse-engineering/push-events.md` 에 이미 문서화.
- 이미 알려진 이벤트 타입: `pending-order-refresh`, `purchase-price-refresh`, `share-holdings`, `web-push` — 모두 "재조회하라" 신호.
- **체결/시세 실시간 푸시 자체는 없음.** 시세는 폴링 + 사용자 인터랙션 기반.

### 3.7 옵션 실시간 틱 (별도)

`GET wts-cert-api.tossinvest.com/api/v1/member-subscriptions/get-option-real-time-tick`

- 옵션 종목 한정의 유료 구독 플래그 조회. **주식/ETF 에는 영향 없음** (이번 SOXL 페이지에서도 호출되지만 옵션 차트 모듈에서 의도된 것). 옵션 라이브 틱은 별도 구독 모델로 차단되어 있을 가능성.

---

## 4. LLM 컨텍스트에 유용한 부가 데이터

### 4.1 종목 메타데이터 (정적/준정적)

| URL | 정보 | LLM 활용 |
| --- | --- | --- |
| `wts-info-api/api/v2/stock-infos/{code}` | symbol, isinCode, name(한/영/일), market, group(ETF/주식/SPAC), currency, sharesOutstanding, listDate, **leverageFactor**, derivativeEtf, optionSupported, riskLevel, purchasePrerequisite | 종목 식별·분류·리스크 등급. SOXL 의 경우 `leverageFactor=3.0`, `derivativeEtp=true`, `riskLevel="2"`. |
| `wts-info-api/api/v1/stock-infos/header/{code}` | sections[]: ETF(grossExpenseRatio, dividendYieldRatio), PRICE(close, high, low, 52w high/low), TRADING_AMOUNT(ranking, delta), TRADING_STRENGTH | 종목 페이지 헤더에 보이는 요약. 1요청으로 LLM 에 풍부한 컨텍스트 제공. |
| `wts-info-api/api/v1/stock-detail/ui/{code}/common` | badges, notices, memoCount | 위험경고/공시 배지. 변동성 ETF 에 대한 토스의 경고 노출. |
| `wts-info-api/api/v1/stock-infos/{code}/wts-badges` | UI 배지 | 종목 페이지 상단 배지 |
| `wts-cert-api/api/v1/stock-infos/{code}/red-flags` | 인증 필요. 위험 플래그 모음 | 매매 전 경고 |
| `wts-info-api/api/v1/option-infos/default-chart-option?underlyingGuid={code}` | 옵션 차트 기본값 | 옵션 페이지 진입 시 |

### 4.2 사용자/거래 컨텍스트 (인증 필요)

| URL | 정보 |
| --- | --- |
| `wts-cert-api/api/v3/trading/order/{code}/trading-status` | 매매 가능/제한 상태 |
| `wts-cert-api/api/v2/trading/order/{code}/prerequisite` | 매매 전 동의·약관 체크 (레버리지 ETP 동의 등) |
| `wts-cert-api/api/v1/trading/orders/calculate/{code}/orderable-quantity/sell` | 매도 가능 수량 |
| `wts-cert-api/api/v2/trading/orders/calculate/{code}/cost-basis-elements` | 평단·매도 원가 |
| `wts-cert-api/api/v1/trading/orders/calculate/{code}/average-price` | 평균 매수가 |
| `wts-cert-api/api/v1/trading/orders/histories/PENDING?stockCode={code}&...` | 종목별 미체결 주문 |
| `wts-cert-api/api/v1/trading/orders/histories/COMPLETED?stockCode={code}&...` | 종목별 체결 주문 |
| `wts-cert-api/api/v3/trading/orders/histories/compact/executed?productCode={code}&timeUnit=thirty_minute` | **본인 체결 내역을 30분 버킷으로 집계** — 차트 위에 오버레이용 |
| `wts-cert-api/api/v1/trading/analysis/productCode/{code}` | 트레이딩 분석 (SOXL 은 `null` 반환 — 데이터 없음 또는 대상 외) |
| `wts-cert-api/api/v4/trading/auto-trading?productCode={code}&...` | 자동매매 설정 |
| `wts-cert-api/api/v1/user-price-alimy/{code}` | 사용자 가격 알림 |

`compact/executed` 의 응답은 본인의 30분 버킷 체결 평단/총량 — LLM 에게는 일반 시장 데이터로 노출하지 말고 사용자 컨텍스트로 분리해야 함.

### 4.3 시장 환경 (전역)

| URL | 정보 |
| --- | --- |
| `wts-info-api/api/v1/dashboard/wts/overview/exchange-rates` | 환율 |
| `wts-cert-api/api/v1/dashboard/wts/overview/indicator/index?market=us` | 미국 시장 인덱스 |
| `wts-api/api/v2/system/trading-hours/integrated` | 거래시간 |
| `wts-api/api/v1/exchange/usd/base-exchange-rate` | USD 기준환율 |
| `wts-info-api/api/v1/usa-market/get-option-biz-day-by-overtime?overtimeFlag=false` | 미국 영업일 |
| `wts-info-api/api/v1/rankings/realtime/stock?size=10` | 실시간 인기 종목 랭킹 |
| `wts-info-api/api/v1/dashboard/intelligences/all` (POST) | 대시보드 인텔리전스 카드 |
| `wts-cert-api/api/v1/dashboard/common/stocks/mini-chart` (POST) | 미니차트 일괄 조회 |

### 4.4 커뮤니티

`wts-cert-api/api/v4/comments?subjectType=STOCK&subjectId={code}&commentSortType=POPULAR|RECENT`

- 종목 커뮤니티 댓글. 토스가 신원·모더레이션 측면에서 민감하게 다루는 영역이므로 기존 `docs/reverse-engineering/rpc-catalog.md` 에선 첫 릴리스 제외 처리됨.
- LLM 에 노출 시 의견 출처·사용자 식별 정보 제거 필요.

---

## 5. LLM 통합 제안 (실용 가이드)

LLM 에게 "이 종목을 지금 사야 하는지 의견 줘" 정도의 컨텍스트를 채우려면 다음 5개만 묶어도 충분히 풍부함:

1. **종목 메타 (1회)** — `GET /api/v2/stock-infos/{code}` + `GET /api/v1/stock-infos/header/{code}`
2. **현재가/등락/체결강도 (폴링 5~10s)** — `GET /api/v3/stock-prices/details?productCodes={code}`
3. **차트 (스냅샷)** — `GET /api/v1/c-chart/us-s/{code}/day:1?count=60&useAdjustedRate=true` + `GET .../min:30?count=120&useAdjustedRate=true`
4. **호가/체결 (스냅샷 or 짧은 폴링)** — `GET /api/v3/stock-prices/{code}/quotes` + `GET /api/v2/stock-prices/{code}/ticks?count=50`
5. **이벤트 신호** — SSE `/api/v1/wts-notification` 구독 (옵션) — 단, 시세 자체는 안 옴

### 5.1 가벼운 폴링 전략 권장값

| 대상 | 추천 주기 | 비고 |
| --- | --- | --- |
| `stock-prices/details` | 3–5초 | 정규장; pre/after 는 10–15초 |
| `quotes` | 1–2초 | 매매 의사결정용. 호가 jitter 가 작으므로 캐시 가능 |
| `ticks?count=20` | 2–3초 | `cumulativeVolume` 으로 신규 틱만 잘라내기 |
| `c-chart .../min:1 or min:5` | 1분에 1회 | 마지막 캔들이 진행 중인 시점엔 1회 더 호출해서 갱신 |

### 5.2 인증 부담

- 차트·시세·호가·틱은 모두 `wts-info-api` 호스트의 공개 엔드포인트. **세션 쿠키 없이도 동작**할 가능성 (검증 필요). CLI 의 비인증 데이터 패스로 분리 가능.
- 본인 평단/주문/체결은 `wts-cert-api` — `tossctl` 의 기존 로그인 플로우 (브라우저 쿠키 임포트) 그대로 사용.

### 5.3 KRW 환산 일관성

- 캔들 응답엔 KRW 환산 없음 → `result.exchangeRate` 사용
- `stock-prices/details` 엔 `*Krw`/`*KrwDecimal` 동시 제공 → **데시멀 필드 우선**
- 환율은 응답 시점 기준이므로 분 단위로 갱신될 수 있음. LLM 에 "USD 기준" 그대로 노출이 가장 깨끗함.

---

## 6. 기존 문서 대비 새로 확인한 사실

| 항목 | 기존 문서 상태 | 본 캡쳐로 추가 확인 |
| --- | --- | --- |
| 일봉 캔들 | `c-chart/kr-s/{code}/day:1` (한국 사례) | `c-chart/us-s/{code}/day:1` 미국 사례 동일 응답 스키마 확인 |
| 분봉 캔들 | 미문서화 | **`day:1` 외에 `min:30` 도 동일 패턴, 추가 필드 `sessionType` (pre/main/day/after) 발견** |
| 차트 페이지네이션 | 미문서화 | **`from` 파라미터로 과거 페이지, 응답의 `nextDateTime` 이 다음 커서** |
| 체결 틱 | 미문서화 | **`/api/v2/stock-prices/{code}/ticks?count=N` 발견** — 호가창과 짝, `tradeType`+`cumulativeVolume` 포함 |
| 호가 | 미문서화 | **`/api/v3/stock-prices/{code}/quotes` 발견** — Top-of-book, 다단 가능성 있음 |
| 풀 시세 | 부분 (`/api/v1/product/stock-prices`) | **`/api/v3/stock-prices/details` 가 훨씬 풍부 (52w/1y, marketCap, tradingStrength, preDay, afterMarket)** |
| 종목 헤더 | 미문서화 | **`/api/v1/stock-infos/header/{code}` 발견** — ETF 비용/배당, 거래대금 랭킹, 체결강도 |
| 본인 체결 30분 버킷 | 미문서화 | **`/api/v3/trading/orders/histories/compact/executed?...&timeUnit=thirty_minute`** |
| 체결강도 (`tradingStrength`) | 미문서화 | 의미 추정: `매수체결합 / 매도체결합 × 100` |
| sessionType 분포 | — | `main`/`day`/`pre`/`after` 4종 확인 |
| 옵션 실시간 틱 구독 | 미문서화 | `/api/v1/member-subscriptions/get-option-real-time-tick` 존재 (옵션 한정 추정) |
| 트레이딩 분석 (`/api/v1/trading/analysis/productCode/{code}`) | 일부 종목 capture | **SOXL 은 `null` 반환** — 분석 미제공 종목 존재 확인 |

---

## 7. 보안·법적 유의사항

- 데이터는 사용자가 토스증권에 정상 로그인한 세션에서 자기 자신의 화면을 통해 받은 것 — 본 분석은 토스증권의 **개인 사용** 범위 내.
- 시세 데이터의 **재배포·상업적 이용**은 토스증권/거래소 약관 위반 소지. CLI 가 단일 사용자에게만 데이터를 노출(개인 분석용)하는 한도에서 사용해야 함.
- `wts-cert-api/api/v4/comments` 의 사용자 식별 정보는 LLM 컨텍스트에 직접 넣지 말 것.
- SSE 의 `web-push` 페이로드는 사용자에게 보여주는 알림 텍스트이므로 PII 가 들어 있을 수 있음.

---

## 8. 후속 작업 (제안)

1. `docs/reverse-engineering/rpc-catalog.md` 의 **Quote and Symbol Detail** 섹션에 위 신규 엔드포인트 5개 추가.
2. `internal/client/quote.go` 의 `GetQuote` 를 `/api/v3/stock-prices/details` 기반으로 업그레이드해서 52w/marketCap/tradingStrength 노출.
3. CLI 에 `tossctl chart {symbol} --tf 30m --count 200` 형태의 차트 명령 추가 후보 — `c-chart/{product}/{code}/{stepUnit}` 매핑.
4. `tossctl quotes {symbol}` 명령 후보 — 호가창 + 최근 체결 N건 묶음.
5. SSE 리스너 (`tossctl push listen`) 에 시세 폴링 트리거 옵션 추가 (사용자가 알림 받을 때 자동 재조회).
6. 본인 체결 30분 버킷 (`compact/executed`) 을 `tossctl my fills --tf 30m` 로 노출.

---

## Appendix — Extended findings (분봉 vocabulary · AI 시그널 · 깊은 호가 · 보유 주식)

1차 보고서 작성 후 추가 탐사한 4개 토픽 — 차트 시간단위 전체 도메인, 시그널/뉴스/관련종목, KR 종목 풀호가, 포트폴리오 — 의 검증 결과를 모았다. 캡쳐는 2026-05-15 21:14–21:22 KST 동안 SOXL 주문페이지 → 홈(`/`) → 삼성전자(KR) 흐름에서 수집했고, 일부는 `evaluate_script` 로 직접 호출해 stepUnit/세션 vocabulary 를 전수 확정했다.

## A. 차트 시간단위 — 전체 도메인

`evaluate_script` 로 9개 unit × 10개 step 조합 90개를 전수 시도한 결과:

| stepUnit | 응답 | 비고 |
| --- | --- | --- |
| `min:1`, `min:3`, `min:5`, `min:10`, `min:15`, `min:30`, `min:60` | 200 | **분봉 7종 모두 지원** — `60` 분봉도 `hour:1` 이 아니라 `min:60` 으로 호출 |
| `day:1` | 200 | 일봉 |
| `week:1` | 200 | 주봉 |
| `month:1`, `month:3` | 200 | 월봉 + **분기봉 (`month:3`)** |
| `year:1` | 200 | 연봉 |
| 그 외 모든 조합 (`sec:*`, `hour:*`, `minute:*`, `tick:*`, `min:20`, `min:120`, `day:3`, `week:3`, `month:5`, `year:3` 등) | **400 `statusCode:400, code:"400"`** | 잘못된 vocabulary |

전체 stepUnit vocabulary 정리:

```
min:1, min:3, min:5, min:10, min:15, min:30, min:60,
day:1, week:1, month:1, month:3, year:1
```

(12종이 토스가 공식적으로 지원하는 전부)

### A.1 차트 필터 파라미터

홈 페이지의 추천 차트 호출에서 새로 발견된 쿼리 파라미터를 전수 검증:

```
GET /api/v1/c-chart/us-s/{code}/{stepUnit}
    ?count=N
    [&from=<ISO8601 with TZ>]
    [&session=all|main|day|pre|after]      # 신규
    [&investMode=integrated|regular]        # 신규
    [&useAdjustedRate=true|false]
```

`session` 파라미터를 `["all","main","day","pre","after","regular","intraday"]` 로 전수 시도한 결과:

| session 값 | 결과 | 의미 |
| --- | --- | --- |
| `all` (또는 미지정) | 200 | 모든 세션 통합 (기본값) |
| `main` | 200 — `sessionType: "main"` 만 | 정규장만 |
| `day` | 200 — `sessionType: "day"` 만 | 토스 주간거래 (Korean daytime) |
| `pre` | 200 — `sessionType: "pre"` 만 | 미국 프리마켓 |
| `after` | 200 — `sessionType: "after"` 만 | 미국 애프터마켓 |
| `regular`, `intraday` 등 그 외 | 400 | 미지원 |

`investMode` 는 `integrated` / `regular` 두 값만 허용. SOXL(미국 ETF)에서는 두 값이 동일한 결과를 반환했지만, **NXT(대체거래소) 가 활성화된 KR 종목** 에서 `regular` 와 `integrated` 가 갈라질 가능성이 있음 (`stock-infos` 응답의 `nxtSupported`, `daytimePriceSupported` 플래그와 연동될 것으로 추정).

### A.2 미니 차트 (일중 배치)

`POST https://wts-cert-api.tossinvest.com/api/v1/dashboard/common/stocks/mini-chart`

```json
// Request
{ "codes": ["NAS0241010007", "AMX0260127004", "US20100311002", "US20100311003"] }

// Response (요약)
{
  "result": {
    "baseDateTime": "2026-05-15T12:21:26Z",
    "baseRange": "1d",
    "baseStep": "10m",
    "miniCharts": [
      {
        "code": "US20100311002",
        "timezone": "America/New_York",
        "tradingStart": "2026-05-15T00:00:00Z",
        "tradingEnd": "2026-05-15T13:30:00Z",
        "candles": [
          { "startDate": "2026-05-15T00:00:00Z", "endDate": "2026-05-15T00:30:00Z",
            "open": 185.13, "close": 187.69, "low": 185.13, "high": 187.81,
            "base": 186.19, "sessionType": "day" },
          // ... 25 candles for the day (30분 버킷 — `baseStep: 10m` 라벨과 실제 버킷 폭이 다름)
        ]
      }
    ]
  }
}
```

- 인증 필요 (`wts-cert-api`).
- **`/c-chart` 와 필드명이 다름**: `dt` → `startDate`/`endDate`, 그 외 OHLCV 동일.
- 멀티 종목 일중 미니 그래프 노출용 — 차트 정밀도가 낮은 페이지(자산 페이지, 워치리스트 등)에서 효율적으로 사용 가능.
- `sessionType` 값 매핑이 `/c-chart` 와 약간 다름 (`day` 가 주간장+after 통합 등) — 정합성에 의존하면 안 됨.

---

## B. 보조지표 / 기술적 분석 / 시그널

### B.1 결론 — 기술적 지표는 클라이언트 사이드 계산

다음 후보 엔드포인트를 모두 시도했으나 모두 404/400:

- `/api/v1/indicators/{code}/{name}` — bollinger, rsi 등
- `/api/v1/c-chart/.../indicators`
- `/api/v1/signals/stock/{code}`
- `/api/v1/stock-detail/ui/{code}/indicator` (UI 서브경로도 `common` 외에는 모두 404)

저장된 사용자 설정만 존재함:

- `GET /api/v1/properties/member/prochart-setting` — 프로 차트 레이아웃/그린 도형 등 (사용자별 JSON)
- `GET /api/v1/properties/member/multi-prochart-setting`
- `GET /api/v1/properties/member/mts-prochart-setting`

→ **MA, RSI, MACD, 볼린저밴드, 스토캐스틱 등은 서버가 계산해 주지 않음.** 클라이언트가 `/c-chart` 의 OHLCV 시리즈를 받아 TradingView 호환 라이브러리로 계산. CLI/LLM 에서 사용하려면 `/c-chart` 결과를 받아 직접 (또는 Go `markcheno/go-talib` 같은 라이브러리로) 계산해야 함.

### B.2 토스 자체 "AI 시그널" — 보조지표의 대체재

토스는 클래식 기술적 지표 대신 자체 NLP/이슈 큐레이션 기반 **AI 시그널** 을 풍부하게 제공:

| URL | 메서드 | 본문/쿼리 | 인증 | 용도 |
| --- | --- | --- | --- | --- |
| `wts-info-api/api/v1/dashboard/wts/overview/ai-signals` | POST | `{"productCodes":[...]}` (최대 ~100개) | 공개 | **종목별 1줄 사유** — `reasoningDescription` (예: `"실적 개선 효과"`, `"차익실현 매도"`, `"AI 투자 확대에 따른 재무 부담"`) |
| `wts-info-api/api/v1/dashboard/wts/overview/ai-signals/detail?productCode={code}&productType=STOCKS` | GET | — | 공개 | **종목별 풀 시그널 상세** — 직전 응답의 한 줄 사유에 대한 자세한 근거 (아래 구조 참고) |
| `wts-info-api/api/v2/dashboard/wts/overview/signals` | POST | `{"productCodes":[...], "filters":[]}` | 공개 | **이벤트 시그널** — 실적 발표/공시 알림 등 |
| `wts-info-api/api/v1/dashboard/intelligences/all` | POST | `{"productCode":"..."}` (선택) | 공개 | 인텔리전스 카드 (대시보드용) |

#### B.2.1 ai-signals 배치 응답 예

```json
{
  "result": {
    "signals": [
      { "productCode": "US20190226001", "reasoningDescription": "실적 개선 효과" },
      { "productCode": "US20100311002", "reasoningDescription": "차익실현 매도" },
      { "productCode": "US20040819002", "reasoningDescription": "AI 투자 확대에 따른 재무 부담" },
      { "productCode": "US20000627001", "reasoningDescription": "인플레이션 우려" }
      // ...
    ]
  }
}
```

#### B.2.2 ai-signals/detail 응답 예 (LLM 컨텍스트의 보석)

```jsonc
{
  "result": {
    "terms": { "personalizedServiceAgreed": true, "serviceAgreed": true },
    "createdAt": "2026-05-15T11:54:26Z",
    "signalId": "STOCKS:NAS0FR6ZB-E0:20260515",
    "signalDirection": 1,                          // 1 = bullish, -1 = bearish (추정)
    "reasoning": {
      "description": "실적 개선 효과",
      "issue": {
        "assetName": "슈퍼 리그 엔터프라이즈",
        "assetType": "STOCKS",
        "assetCode": "NAS0FR6ZB-E0",
        "profitLossRate": 0.4054,                  // 시그널 시점부터의 수익률
        "description": {
          "data": [
            "슈퍼 리그 엔터프라이즈는 1분기 매출이 전년 대비 10.49% 증가하고, ...",
            "Misfits Ads Business 인수 효과로 ...",
            "애널리스트들은 실적 개선과 인수 효과를 반영해 '매수' 의견을 ..."
          ]
        },
        "investmentType": "RECOMMEND",
        "originCodes": ["US20190226001"]
      },
      "news": {
        "data": [
          { "id": "benzinga_52592661", "source": "benzinga", "agencyName": "벤징가",
            "title": "슈퍼리그 엔터프라이즈, 1분기 실적 전망치 상회…조정 EPS·매출 모두 기대치 초과",
            "createdAt": "2026-05-15T11:50:38Z" },
          { "source": "reuter", "title": "...", "createdAt": "..." }
        ]
      },
      "keywords": ["실적 개선", "매출 증가", "인수 효과"]
    },
    "relatedReasoning": {
      "details": [
        { "assetName": "피스클노트 홀딩스",
          "description": { "data": ["슈퍼 리그 엔터프라이즈와 피스클노트 홀딩스 두 기업 모두 광고 산업군에 속해 있어요.", ...] },
          "relationship": { "subjectName": "슈퍼 리그 엔터프라이즈", "relation": "같은산업", "objectName": "피스클노트 홀딩스" }
        }
      ]
    },
    "hasRelatedReasoning": true
  }
}
```

이 한 호출에서 **(1) 가격 움직임 사유 3줄 요약, (2) 최근 뉴스 헤드라인 + 출처, (3) 동일 산업군 관련 종목 추천 + 관계** 가 한꺼번에 옴 — LLM 컨텍스트로 그대로 직렬화하기에 완성도가 매우 높음. 한국어 native.

#### B.2.3 이벤트 시그널 응답 예

```json
{
  "result": {
    "exposeSignals": true,
    "signalsList": [
      {
        "productCode": "US20190226001",
        "primarySignal": { "signalLabel": "소식", "signalInfo": "실적 발표",
                           "signalId": 6000000, "datetime": "2026-05-15T20:30:00" },
        "signals": [
          { "signalLabel": "소식",
            "signalInfo": "실적이 발표됐어요. 2026년 3월 매출 $300.3만, 1주당 순이익 -$0.98",
            "signalId": 6000000, "datetime": "2026-05-15T20:30:00" }
        ]
      }
    ]
  }
}
```

`signalId` 6000000 대는 실적 카테고리로 보이며 다른 ID 대역에 공시/배당/임원 매매 등이 매핑될 것으로 추정. 다회 호출 후 ID 분포를 수집해 카탈로그를 만들 수 있음.

### B.3 시장 인덱스/시그널 (전역)

| URL | 용도 |
| --- | --- |
| `wts-cert-api/api/v3/dashboard/wts/overview/indicator` | KOSPI/KOSDAQ/NASDAQ/S&P500 등 주요 인덱스 + 환율 + 야간선물 등 헤드라인 카드 (~12KB 응답) |
| `wts-cert-api/api/v2/dashboard/wts/overview/ranking` POST | **랭킹 카드** — `{id, filters[], duration, tag}` 본문. 관찰 본문: `{"id":"biggest_total_amount","filters":["KRX_MANAGEMENT_STOCK","MARKET_CAP_GREATER_THAN_50M","STOCKS_PRICE_GREATER_THAN_ONE_DOLLAR"],"duration":"realtime","tag":"all"}`. `id` 후보(추정): biggest_total_amount(거래대금), biggest_volume(거래량), most_rising/most_falling(상승/하락률) 등 |
| `wts-info-api/api/v1/rankings/realtime/stock?size=10` | 실시간 인기 종목 |

---

## C. 호가창 (Orderbook) — 시장별 깊이 차이

### C.1 같은 path 의 v1/v2/v3 가 다른 스키마를 반환

| Version | URL | dt 필드 | 영문/한글 키 | 추가 필드 |
| --- | --- | --- | --- | --- |
| v3 | `/api/v3/stock-prices/{code}/quotes` | 없음 | `offerPrices`, `offerVolumes`, `bidPrices`, `bidVolumes` | `midPrices`, `midOfferVolumes`, `midBidVolumes`, `singlePrice`, `estimatedPrice`, `estimatedVolume` |
| v2 | `/api/v2/stock-prices/{code}/quotes` | 있음 | `sellPrices`, `sellQuantities`, `buyPrices`, `buyQuantities` | `isSinglePrice`, `estimatedPrice`, `estimatedVolume`, `sumOfSellQuantities`, `sumOfBuyQuantities` |
| v1 | `/api/v1/stock-prices/{code}/quotes` | 있음 | (v2 와 동일) | (v2 와 동일) |

LLM/CLI 에서 사용 시 **`v3` 가 의미적으로 가장 명확** 하나 `dt` 가 없어 폴링 동기화엔 `v2` 가 유리. 하나만 골라야 한다면 v3 + `Date` 헤더 사용 또는 `/stock-prices/details.tradeDateTime` 동시 호출 권장.

### C.2 깊이 차이 — KR 10단계 vs US 1단계

**US 종목 (SOXL, AMEX)** 프리마켓 시점 v3 응답:

```json
{ "result": { "close": 168.36, "offerPrices": [168.38], "offerVolumes": [935],
              "bidPrices":  [168.04], "bidVolumes":  [5],
              "offerVolume": 935, "bidVolume": 5 } }
```

→ **Top-of-book 1단계만.** 미국 시장 데이터 라이센스 제약으로 추정. `depth=10` 등 쿼리 파라미터 추가해도 효과 없음 (검증 완료).

**KR 종목 (삼성전자 A005930, KOSPI)** v3 응답:

```json
{ "result": {
  "close": 273500,
  "offerPrices":  [278500, 278000, 277500, 277000, 276500, 276000, 275500, 275000, 274500, 274000],
  "offerVolumes": [42254, 65779, 31428, 28978, 18195, 59953, 21617, 39916, 34249, 28041],
  "bidPrices":    [273500, 273000, 272500, 272000, 271500, 271000, 270500, 270000, 269500, 269000],
  "bidVolumes":   [10069, 31985, 23652, 23670, 27173, 42680, 48332, 75745, 40444, 93203],
  "midPrices": [0, 0], "midOfferVolumes": [0, 0], "midBidVolumes": [0, 0],
  "singlePrice": false, "estimatedPrice": 0, "estimatedVolume": 0,
  "offerVolume": 370410, "bidVolume": 506978
}}
```

→ **10단계 풀호가.** 또한 KR `stock-prices/details` 는 `upperLimit`/`lowerLimit` (상/하한가), `nxtSinglePrice` 등 추가 필드 노출:

```json
{ "code": "A005930", "exchange": "integrated", "open": 295500, "high": 296500,
  "low": 266000, "close": 273500, "volume": 72786490, "value": 20389193061529,
  "base": 296000, "marketCap": 1598957199288000, "tradingStrength": 65.00,
  "upperLimit": 384500, "lowerLimit": 207500, "preDayVolume": 69704516,
  "nxtTradingSuspended": false, "nxtSinglePrice": false, ... }
```

`midPrices`/`midOfferVolumes`/`midBidVolumes` 는 **장중 단일가(예: 동시호가/시간외 단일가)** 가 활성화될 때 채워질 것으로 추정 (`singlePrice: true` + `estimatedPrice` 와 짝).

### C.3 정리표

| 시장 | 호가 단계 | 단일가 추정 | 상하한가 | 동시호가 필드 |
| --- | --- | --- | --- | --- |
| KR 정규 (KOSPI/KOSDAQ) | **10단계** | `estimatedPrice`/`estimatedVolume` | `upperLimit`/`lowerLimit` (±30%) | `midPrices[]` (장 외 단일가 시) |
| US (NYSE/NASDAQ/AMEX) | 1단계 (Top-of-book) | 미사용 | 없음 | 없음 |

---

## D. 보유 주식 (Holdings / Portfolio)

### D.1 핵심 엔드포인트 — `dashboard/asset/sections/all`

```
POST https://wts-cert-api.tossinvest.com/api/v2/dashboard/asset/sections/all
Content-Type: application/json
X-XSRF-TOKEN: <token>
X-Tossinvest-Account: 1
(쿠키 인증)

Body: {"types": ["SORTED_OVERVIEW"]}
```

- 인증 필수 (`wts-cert-api` + `X-Tossinvest-Account` 헤더 + XSRF 토큰).
- `types` 배열의 다른 관찰값: `MIDDLE` (v1, 빈 응답), `SORTED_OVERVIEW` (v2, 본 응답).
- 응답에 `usePolling`, `pollIntervalMillis: 3000` 포함 → 토스 웹도 3초 폴링 패턴 사용.

### D.2 응답 스키마 (요약)

```jsonc
{
  "result": {
    "sections": [{
      "type": "SORTED_OVERVIEW",
      "data": {
        // 전체 합산
        "principalAmount":            { "krw": 155919294, "usd": 106129.27 },   // 원금
        "evaluatedAmount":            { "krw": 119848392, "usd": 80338.11 },    // 평가금액 (수수료 전)
        "evaluatedAmountAfterFees":   { "krw": 119572751, "usd": 80151.74 },    // 평가금액 (수수료 후)
        "profitLossAmount":           { "krw": -36070901, "usd": -25791.16 },
        "profitLossAmountAfterFees":  { "krw": -36346542, "usd": -25977.53 },
        "dailyProfitLossAmount":      { "krw":  -3295117, "usd":  -2208.82 },
        "profitLossRate":             { "krw": -0.2313, "usd": -0.2430 },
        "profitLossRateAfterFees":    { "krw": -0.2331, "usd": -0.2447 },
        "dailyProfitLossRate":        { "krw": -0.0211, "usd": -0.0208 },

        // marketType 별 (US_OPTION / US_STOCK / KR_STOCK / US_BOND)
        "products": [
          {
            "marketType": "US_STOCK",
            "items": [
              {
                "key": "11101500497::US20100311002::US25459W4583::SOXL",  // {계좌번호}::{guid}::{isin}::{symbol}
                "stockCode": "US20100311002", "stockIsin": "US25459W4583",
                "stockSymbol": "SOXL", "stockName": "SOXL",
                "logoImageUrl": "https://static.toss.im/png-icons/securities/icn-sec-fill-SOXL.png?20240409",
                "quantity": 20.0, "tradableQuantity": 20.0, "unsettledQuantity": 0,
                "currentPrice":     { "krw": 251726, "usd": 168.74 },
                "basePrice":        { "krw": 277758, "usd": 186.19 },     // 전일종가
                "closeWithoutAfter":{ "krw": 251726, "usd": 168.74 },     // 정규장 종가 (애프터 제외)
                "baseWithoutAfter": { "krw": 277758, "usd": 186.19 },
                "purchasePrice":    { "krw": 277505, "usd": 185.61 },     // 평단
                "purchaseAmount":   { "krw": 5550110, "usd": 3712.20 },   // 매수원금
                "evaluatedAmount":  { "krw": 5034526, "usd": 3374.80 },
                "evaluatedAmountAfterFees": { "krw": 5023946, "usd": 3367.72 },
                "profitLossAmount": { "krw": -515583, "usd": -337.40 },
                "profitLossAmountAfterFees":{ "krw": -526163, "usd": -344.48 },
                "dailyProfitLossAmount":    { "krw": -520638, "usd": -349.00 },
                "profitLossRate":         { "krw": -0.0928, "usd": -0.0908 },
                "profitLossRateAfterFees":{ "krw": -0.0948, "usd": -0.0927 },
                "dailyProfitLossRate":    { "krw": -0.0938, "usd": -0.0940 },
                "commission":     { "krw": 10580, "usd": 7.08 },
                "commissionRate": 0.001,
                "buyCommission":  { "krw": 5546, "usd": 3.71 },
                "sellCommission": { "krw": 5034, "usd": 3.37 },
                "tax": null, "taxRate": null,
                "delisting": false, "unlisting": false, "archiving": false,
                "errorPricing": false, "nxtSupported": false,
                "domesticExchange": "integrated", "marketCode": "AMX",
                "stockGroupCode": "", "stockWarrants": false,
                "shortSellingQuantity": 0,
                "rightExpectedQuantity": 0, "rightEvaluatedAmount": 0,
                "notice": { "splitMerge": false, "earningsAnnouncement": false },
                "shareHoldingsType": "us"
              }
              // ... 다른 종목 (MUU, SNXX, SOXS 등)
            ],
            // marketType 단위 합산
            "principalAmount": {...}, "evaluatedAmount": {...},
            "profitLossAmount": {...}, "dailyProfitLossAmount": {...}, ...
          },
          { "marketType": "KR_STOCK", "items": [], ... },
          { "marketType": "US_BOND",  "items": [], ...,
            "expectedMaturityProfitLoss": {...},          // 채권 한정
            "expectedMaturityCommission": {...},
            "expectedMaturityTax": {...},
            "expectedMaturityAmount": {...}  },
          { "marketType": "US_OPTION", "items": [], ... }
        ],
        "hiddenStock": { "count": 0, "all": false, "amount": 0 },
        "usePolling": false,
        "stockNudge": null, "bondNudge": null,
        "pricingErrorMessage": null
      }
    }],
    "pollIntervalMillis": 3000
  }
}
```

### D.3 보조 엔드포인트

| URL | 인증 | 용도 |
| --- | --- | --- |
| `wts-api/api/v1/account/list` | yes | 계좌 목록 (`accountNo`, `name`, `type`, `markets[]`) |
| `wts-api/api/v1/account/detail` | yes | 계좌 상세 (`status`, `openDate`, `lastTradeDate`, `accountName`) |
| `wts-cert-api/api/v1/dashboard/common/cached-orderable-amount` | yes | **주문가능금액** (KR/US 합산, `orderableAmountKr.krw`, `orderableAmountUs.usd`) |
| `wts-cert-api/api/v1/dashboard/wts/overview/margin` | yes | 신용/마진 메시지 (현재는 모두 null) |
| `wts-api/api/v3/my-assets/transactions/markets/{kr\|us}/overview` | yes | **출금가능액 + 다음 결제일별 정산금** (`orderableAmount`, `withdrawableAmount.amount0~3`, `depositAmount`, `estimateSettlementAmount.day1~2`) |
| `wts-api/api/v3/my-assets/transactions/markets/{kr\|us}` | yes | 거래원장 (이미 문서화) |
| `wts-cert-api/api/v1/new-watchlists?includePrice=true&lazyLoad=false` | yes | 워치리스트 + 현재가 (RECENT_WATCH, 사용자 폴더 등) |

### D.4 정합성 메모

- **`tradableQuantity`** ≠ `quantity` 일 때는 미수/미결제 또는 권리주식 있음 (`unsettledQuantity` > 0 또는 `rightExpectedQuantity` > 0).
- **`evaluatedAmount`** 은 수수료 전 평가액. CLI/LLM 이 사용자에게 "내 손익이 얼마인가" 보일 때는 **`profitLossAmountAfterFees`** 와 **`evaluatedAmountAfterFees`** 가 더 정확.
- **`dailyProfitLossRate`** 는 전일 대비 — `currentPrice` vs `basePrice` 비율과 같음.
- `commission` 은 추정 매매 수수료 (매수+매도 합산), `commissionRate` 는 율 (0.001 = 0.1%).
- 종목 페이지의 `/api/v3/stock-prices/details` 와 holdings 의 `currentPrice` 가 동일한 시점에서 미세하게 차이날 수 있음 (별도 캐시) — 일관성을 원하면 holdings 한 호출로 끝내는 것을 권장.

---

## E. 종합 — LLM 통합 시 데이터 매트릭스

| 카테고리 | 권장 엔드포인트 | 인증 | 폴링 | 시장 |
| --- | --- | --- | --- | --- |
| 종목 메타 | `GET /api/v2/stock-infos/{code}` | 공개 | 변경 시만 | KR/US |
| 종목 헤더 (ETF 비용, 52주, 체결강도, 거래대금 랭킹) | `GET /api/v1/stock-infos/header/{code}` | 공개 | 1-5분 | KR/US |
| 풀 시세 | `GET /api/v3/stock-prices/details?productCodes={...}` | 공개 | 3-5초 | KR/US |
| 호가 | `GET /api/v3/stock-prices/{code}/quotes` | 공개 | 1-2초 | KR=10단계, US=1단계 |
| 체결 틱 | `GET /api/v2/stock-prices/{code}/ticks?count=N` | 공개 | 2-3초 | KR/US |
| 차트 (12종 stepUnit, 5종 session) | `GET /api/v1/c-chart/{us-s\|kr-s}/{code}/{stepUnit}?count=N&session=...` | 공개 | 봉 단위 | KR/US |
| 미니 차트 (배치 일중) | `POST /api/v1/dashboard/common/stocks/mini-chart` | 인증 | 페이지당 1회 | KR/US |
| AI 시그널 1줄 (배치) | `POST /api/v1/dashboard/wts/overview/ai-signals` | 공개 | 5-10분 | KR/US |
| AI 시그널 상세 + 뉴스 + 관련종목 | `GET /api/v1/dashboard/wts/overview/ai-signals/detail?productCode=...&productType=STOCKS` | 공개 | 시그널 변경 시 | KR/US |
| 이벤트 시그널 (실적/공시) | `POST /api/v2/dashboard/wts/overview/signals` | 공개 | 5-10분 | KR/US |
| 시장 인덱스 | `GET /api/v3/dashboard/wts/overview/indicator` | 인증 | 30초-1분 | 전역 |
| 랭킹 | `POST /api/v2/dashboard/wts/overview/ranking` | 인증 | 1분 | KR/US |
| 보유 포트폴리오 | `POST /api/v2/dashboard/asset/sections/all` `{"types":["SORTED_OVERVIEW"]}` | 인증 | 3초 (서버 권장) | All |
| 계좌/주문가능금액 | `GET /api/v1/dashboard/common/cached-orderable-amount` | 인증 | 거래 후 | KR/US |
| 출금가능액/정산 | `GET /api/v3/my-assets/transactions/markets/{kr\|us}/overview` | 인증 | 일 1회 | KR/US |
| 거래원장 | `GET /api/v3/my-assets/transactions/markets/{kr\|us}` | 인증 | 필요 시 | KR/US |
| 워치리스트 | `GET /api/v1/new-watchlists` | 인증 | 변경 시 | All |
| 본인 체결 30분 버킷 | `GET /api/v3/trading/orders/histories/compact/executed?productCode={code}&timeUnit=thirty_minute` | 인증 | 거래 후 | 종목별 |
| SSE 푸시 | `GET https://sse-message.tossinvest.com/api/v1/wts-notification` | 인증 | 상시 (재연결 자동) | 전역 |

---

## F. 신규/갱신 후속 작업

기존 §8 에 더해:

7. `tossctl chart` 명령에 `--tf` 옵션 도메인을 **12종 stepUnit (min:1/3/5/10/15/30/60, day:1, week:1, month:1/3, year:1)** + `--session main|day|pre|after|all` 로 매핑.
8. `tossctl holdings` (또는 `tossctl portfolio`) 명령 — `POST /api/v2/dashboard/asset/sections/all {"types":["SORTED_OVERVIEW"]}` 래핑. 마켓 필터(`--market us|kr`), 손익 정렬 등.
9. `tossctl orderable` — 한/미 주문가능금액 + 출금가능액 (정산일 포함) 조합.
10. `tossctl signals {symbol}` — AI 시그널 상세 (한 줄 사유 + 3줄 풀 설명 + 뉴스 헤드라인 + 관련종목) 노출. LLM 컨텍스트로 그대로 흘리기 좋음.
11. `tossctl quotes --depth 10 {kr-symbol}` — KR 종목은 10단계 호가, US 는 자동으로 1단계 fallback.
12. 기술적 지표는 서버에 없음을 명시하고, OHLCV 를 받아 클라이언트에서 계산하는 헬퍼 (`internal/indicators/`) 를 추가 — `talib` 호환 함수 시그니처가 자연스러움.
13. `docs/reverse-engineering/rpc-catalog.md` 의 **Quote and Symbol Detail** + 신규 섹션 **Chart / Holdings / Signals** 추가.
