# US Options Reverse Engineering

**Source:** Image #4 (SNDK 옵션 탭, 2026-05-16 캡쳐) + probe `.captures/2026-05-16/sndk-options-stockinfo/`.

The page tab at `https://www.tossinvest.com/stocks/{productCode}/option` displays the option chain UI: expiry ladder + strike chain + per-row OI/volume/등락률/option price + ITM marker + per-option mini chart + 호가 panel + 주문 panel.

## Key Finding — Options Are First-Class `productCode`s

Toss treats each option contract as an independent product with an OCC-style productCode:

```
OPT_<rootSymbol><YYMMDD><C|P><strike*1000 zero-padded to 8 digits>_<listDateYYYYMMDD>
```

Examples:

| Underlying | Expiry | Type | Strike | Toss productCode |
| --- | --- | --- | --- | --- |
| SNDK | 2026-05-15 | Call | $1,395.00 | `OPT_SNDK260515C01395000_20260506` |
| SNDK | 2026-05-15 | Put | $1,395.00 | `OPT_SNDK260515P01395000_<listDate>` |

Once you have an `OPT_…` productCode, **almost every existing price/orderbook/tick/chart endpoint works unchanged** — because Toss's product abstraction wraps stocks, ETFs, bonds, and options into one identifier system. This means we get a huge head start: no new client family needed; `OPT_…` flows through `GetQuote` / `GetOrderBook` / `GetTicks` / `GetChart` with one small caveat (`us-o` chart prefix).

## Confirmed Endpoints

| URL | Returns | Same as stocks? |
| --- | --- | --- |
| `GET /api/v1/option-infos/default-chart-option?underlyingGuid={underlying-code}` | `{"result":"OPT_…"}` — nearest expiry's at-the-money option productCode | new |
| `GET /api/v2/stock-infos/{OPT_…}` | full single-option metadata incl. `optionInstrument` block (see below) | yes |
| `GET /api/v3/stock-prices/details?productCodes={OPT_…}` | OHLCV + 52w + KRW pair (option-specific fields zeroed: marketCap=0, value=0, high1y/low1y often 0) | yes |
| `GET /api/v3/stock-prices/{OPT_…}/quotes` | top-of-book offer/bid prices + volumes | yes |
| `GET /api/v2/stock-prices/{OPT_…}/ticks?count=N` | option trade ticks (same shape as stock ticks) | yes |
| `GET /api/v1/c-chart/us-o/{OPT_…}/min:{N}?count=…&useAdjustedRate=true` | OHLCV candles — **note `us-o` product prefix** (vs `us-s` for stocks) | yes (different prefix) |
| `GET /api/v1/stock-detail/ui/{OPT_…}/common` | badges/notices for the option | yes |
| (auth) `GET /api/v1/member-subscriptions/get-option-real-time-tick` | option realtime-tick subscription flag | new |
| (auth) `GET /api/v1/usa-market/get-option-biz-day-by-overtime?overtimeFlag=false` | US option business day | new (mentioned in earlier RE) |

## `optionInstrument` Block

`/api/v2/stock-infos/{OPT_…}.result.optionInstrument` returns:

```json
{
  "marketCode": "CBOE",
  "rootSymbol": "SNDK",
  "name": "샌디스크 $1,395 콜",
  "fullName": "샌디스크 $1,395 콜 (5.15)",
  "completeName": "샌디스크 $1,395 콜 (26.5.15)",
  "underlyingSymbol": "SNDK",
  "underlyingGuid": "NAS0250224006",
  "underlyingName": "샌디스크",
  "maturityDate": "2026-05-15",
  "maturityDateTime": "2026-05-16T03:50:00.000+09:00",
  "corporateActionDateTime": null,
  "corporateAction": null,
  "optionLiquidation": {
    "liquidationDateTime": "2026-05-16T03:50:00.000+09:00",
    "tradingEndBufferDateTime": "2026-05-16T05:00:00.000+09:00",
    "displayLiquidationDateTime": "2시간 46분 후 거래 종료",
    "displayCorporateActionName": null
  },
  "underlyingSecuritiesType": "STOCK",
  "putCall": "CALL",
  "strikePrice": 1395.0,
  "baseDate": "2026-05-15",
  "basePrice": 31.8,
  "last": 31.8,
  "bid": 30.1,
  "ask": 33.9,
  "mid": 32.0,
  "contractUnit": 100.0,
  "openInterest": 82,
  "halted": false,
  "incorporated": true,
  "tradingSuspended": false,
  "buySuspended": false,
  "sellSuspended": false,
  "status": "LISTED",
  "overtime": false
}
```

This single response covers everything Image #4 shows for a single row:
- Strike, Put/Call, expiry — `strikePrice`, `putCall`, `maturityDate`
- Last / bid / ask / mid — `last`, `bid`, `ask`, `mid`
- OI — `openInterest`
- Contract multiplier — `contractUnit: 100`
- "2시간 56분 후 거래 종료" countdown — `optionLiquidation.displayLiquidationDateTime`
- Halted / suspended state — flags
- Status `LISTED` / `EXPIRED` / etc.

The parent `result` also carries `optionPennyPilotPriceSupported: true` (penny increments for liquid contracts) and the typical name/logo/market fields.

## Image #4 → Coverage Map

| Image #4 element | Endpoint | Notes |
| --- | --- | --- |
| Header underlying price (SNDK $1,393.96) | `/api/v3/stock-prices/details?productCodes=NAS0250224006` | use the underlying's productCode, not OPT_ |
| 1일 범위 / 52주 범위 / 거래대금 / 체결강도 / 시가총액 순위 (top bar) | `quote get NAS0250224006` | underlying quote |
| **Expiry ladder (5/15, 5/22, 5/29, 6/5, 6/12, 6/18, 6/26, 7/17, 8/21)** | **❌ NOT FOUND** | needs browser capture (see Gaps below) |
| 콜/풋/전체 toggle | client-side filter | derive from `optionInstrument.putCall` |
| 행사가 30개 (Strike chain — call+put per strike) | **❌ NOT FOUND** | needs browser capture |
| Per-row OI / 거래량 / 등락률 / 옵션 가격 | `/api/v3/stock-prices/details` per OPT_ + `optionInstrument.openInterest` from stock-info | bulk works: `productCodes=OPT_A,OPT_B,…` |
| 행사가 column | derive from OPT_ productCode (digits between `C`/`P` and `_<listDate>`, ÷ 1000) | |
| ITM/OTM marker line at SNDK $1,393.98 | client-side computed: ITM if strike < underlyingLast (calls) or strike > underlyingLast (puts) | |
| 옵션 가격 +10.06% etc. | from OPT_'s `/details` `(close-base)/base` (existing logic in `applyPriceDetails`) | |
| Per-option mini chart (15분/일/주/월/년) | `/api/v1/c-chart/us-o/{OPT_…}/min:15` etc. | works with our `chart get` if --tf maps to min/day/week/month/year |
| 호가 panel ("호가를 보려면 옵션계좌가 필요해요") | `/api/v3/stock-prices/{OPT_…}/quotes` | works without auth for read-only |
| 옵션 주문 panel (구매/판매/대기 + 지정가/시장가) | `wts-cert-api/api/v2/trading/order/{OPT_…}/prerequisite` etc. (untested) | requires opening an option account first; out of scope for read-only PR6 |
| `옵션 계좌 만들기` button | UI flow; likely a prerequisite call we haven't captured | out of scope |

## Critical Gap — Chain Enumeration

The chain UI requires two enumeration calls Toss must make somewhere:

1. **List of expiry dates** for an underlying (5월 15일, 5월 22일, …)
2. **List of strike prices** for `(underlying, expiry, call|put)` (e.g. all SNDK calls expiring 2026-05-15)

Without these, the user must already know the exact OCC productCode to look up an option. With these, we can enumerate the full chain matching Image #4.

Probed (all 404):

```
/api/v1/option-infos/expiries?underlyingGuid={code}
/api/v1/option-infos/maturity-dates?underlyingGuid={code}
/api/v1/option-infos/strikes?underlyingGuid={code}&maturityDate=…
/api/v1/option-infos/chain?underlyingGuid={code}
/api/v1/option-chain/{code}/expiries
/api/v1/option-instruments/expiries?…
/api/v1/dashboard/wts/overview/option-chain?…
/api/v1/usa-market/options/{…}
/api/v1/usa-options/{…}
/api/v1/stock-derivatives/option/{…}
/api/v1/options/{…} (wts-info-api and wts-cert-api)
/api/v1/option/{…}
/api/v1/option-board?underlyingGuid=…
/api/v1/derivatives/options?…
+ POST variants on the same paths
```

Some `wts-api.tossinvest.com/api/v1/option-infos/...` returned `401` (existence unclear — auth required first). The actual path is hidden inside a Next.js page chunk that the deployed JS no longer matches the page-shell we fetched (404 on `/_next/.../option-*.js`).

**Action required:** open browser DevTools Network tab on `https://www.tossinvest.com/stocks/NAS0250224006/option`, capture all `XHR/Fetch` calls during expiry-ladder render and strike-chain render, and add the captured response shapes to this file. Once the path is known, implementation is mechanical (same pattern as our existing `Signals` family).

## Workaround Until Chain Found

`/api/v1/option-infos/default-chart-option?underlyingGuid={underlying}` returns the **nearest expiry's at-the-money option's productCode**. From that:

1. Parse the OPT_ to extract `(rootSymbol, yymmdd, C/P, strike, listDate)`.
2. Probe a synthetic range of strikes by varying the `strike*1000` portion (e.g. for SNDK at $1395 try strikes from $1300 to $1500 in $5 steps) against `/api/v3/stock-prices/details?productCodes=…` in bulk. Toss returns valid contracts only.
3. For other expiries, the `listDate` segment is unknown without an explicit endpoint — but for next-Friday standard expiries the `listDate` is often the Tuesday before. Brute-forcing this is fragile.

This is not a substitute for the real enumeration endpoint. Recommend doing the browser capture first.

## CLI Mapping (planned for PR6)

| Command | Endpoint(s) | Status |
| --- | --- | --- |
| `tossctl options info <OPT_…>` | `/api/v2/stock-infos/{OPT_…}` | implementable now |
| `tossctl options quote <OPT_…> [--follow]` | `/api/v3/stock-prices/details` (reuse `quote get`) | reuse existing |
| `tossctl options book <OPT_…> [--follow]` | `/api/v3/stock-prices/{OPT_…}/quotes` (reuse `quotes book`) | reuse existing |
| `tossctl options ticks <OPT_…> [--follow]` | `/api/v2/stock-prices/{OPT_…}/ticks` (reuse `quotes ticks`) | reuse existing |
| `tossctl options chart <OPT_…> --tf {1m\|…}` | `/api/v1/c-chart/us-o/{OPT_…}/min:{N}` | **chart product-prefix logic needs `us-o`** for OPT_ codes |
| `tossctl options nearest-atm <underlying>` | `/api/v1/option-infos/default-chart-option?underlyingGuid={…}` | new helper; returns the OPT_ productCode |
| `tossctl options chain <underlying> [--expiry YYYY-MM-DD] [--type call\|put]` | **TBD** — needs captured chain endpoint | blocked on browser capture |

The first 6 are implementable today. The last is the gap.
