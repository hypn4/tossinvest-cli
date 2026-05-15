# Toss Securities RPC Catalog

Verified from public web traffic and public page navigation on 2026-03-11.

This file is the source of truth for endpoint discovery. It should grow before the Go client grows.

## Status Legend

- `public`: works without login
- `guest`: works before authenticated account state, but may depend on browser bootstrap
- `auth`: requires a logged-in web session
- `blocked`: excluded from CLI scope
- `unknown`: not captured yet

## Hostnames

| Hostname | Role | Notes |
| --- | --- | --- |
| `wts-api.tossinvest.com` | core web runtime and session bootstrap | likely holds login and user-setting paths |
| `wts-info-api.tossinvest.com` | market and UI data | strong candidate for read-only quote and stock detail data |
| `wts-cert-api.tossinvest.com` | certified or sensitive read paths | comments, indicators, some overview widgets |
| `cdn-api.tossinvest.com` | refresh and static coordination | low direct CLI value so far |
| `tuba-static.tossinvest.com` | static variables | not a CLI target |
| `log.tossinvest.com` | telemetry | blocked from CLI scope |

## Bootstrap and Runtime

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v3/init?tabId=...` | browser tab bootstrap | `.result` is boolean `true` in public capture | none | useful for reproducing minimal browser session behavior |
| `public` | `GET` | `wts-api.tossinvest.com` | `/api/v1/time` | server time | object under `.result` | none | likely helpful for request signing or freshness checks later |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v1/user-setting` | current user or guest settings | object under `.result` | none | seen without login |
| `public` | `GET` | `wts-api.tossinvest.com` | `/api/v2/system/trading-hours/integrated` | trading-hours metadata | object under `.result` | future metadata | useful for quote context |
| `blocked` | `POST` | `log.tossinvest.com` | `/api/v1/perf-log/bulk` | telemetry | not relevant | none | never call from CLI |
| `blocked` | `POST` | `log.tossinvest.com` | `/api/v2/log/bulk` | telemetry | not relevant | none | never call from CLI |

## Login and Session Discovery

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `guest` | `GET` | `www.tossinvest.com` | `/signin?redirectUrl=%2Faccount` | login page | HTML form with phone and QR flows | `auth login` entry | visiting `/account` without auth redirects here |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/cert-init` | login flow bootstrap | request body still undocumented | `auth login` helper only | observed both before and after login redirect |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/qr` | start QR-based login | request body still undocumented | `auth login` helper only | observed in successful QR flow |
| `guest` | `GET` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss/status` | poll QR login state | object under `.result` | `auth login` helper only | repeated polling until approval |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v2/login/wts/toss` | finalize Toss login | request body still undocumented | `auth login` helper only | observed after status polling |
| `guest` | `POST` | `wts-api.tossinvest.com` | `/api/v3/login/ticket` | obtain post-login ticket | request body still undocumented | `auth login` helper only | likely bridges login flow into WTS session |
| `auth` | `mixed` | browser cookies and storage | session persistence state | authenticated session reuse | cookies plus local/session storage | `auth status`, `auth login` | state-save capture showed both cookies and storage keys matter |

## Market Overview

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/trading-info` | dashboard trading-hours cards | `.result.data[]` with `key`, `name`, `marketOpen`, `currentMarketTradingHour` | none | useful reference data, not first-class CLI target |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/exchange-rates` | exchange-rate summary | object under `.result` | none | may support quote context |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/indicator/index?market=kr` | market indicators | `.result.majorIndicatorInfos` | none | public page dependency |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/calendar/economic-events` | calendar snippets | object under `.result` | none | public page dependency |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/dashboard/wts/overview/ranking` | overview ranking widgets | `.result.items[]` | none | body `{"id":"biggest_total_amount","filters":["KRX_MANAGEMENT_STOCK","MARKET_CAP_GREATER_THAN_50M","STOCKS_PRICE_GREATER_THAN_ONE_DOLLAR"],"duration":"realtime","tag":"all"}`. `id` candidates: `biggest_total_amount` (거래대금), and likely `biggest_volume`, `most_rising`, `most_falling` |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/intelligences/all` | dashboard cards | object under `.result.intelligences[]` typed `MAIN`, `GROUP`, `SURVEY`, etc. | none | body `{"productCode":"..."}` optional; returns site-wide intelligence tiles |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/dashboard/wts/overview/signals` | event signals (earnings/disclosure) | `.result.signalsList[]` with `productCode`, `primarySignal.{signalLabel, signalInfo, signalId, datetime}`, `signals[]` | `signals events <syms>` (since v0.5.x) | body `{"productCodes":[...], "filters":[]}`; `signalId` 6000000-range observed for 실적 발표 |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/ai-signals` | **AI signals — batch reasoning** | `.result.signals[]` with `productCode`, `reasoningDescription` (one-line Korean rationale, e.g. `"실적 개선 효과"`, `"차익실현 매도"`) | `signals list <syms>` (since v0.5.x) | body `{"productCodes":[...]}`. Toss's qualitative replacement for classic technical indicators |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/ai-signals/detail?productCode={code}&productType=STOCKS` | **AI signal detail + news + related** | `.result` with `signalDirection` (1/-1), `reasoning.description.data[]` (3 KR bullets), `reasoning.news.data[]` (Benzinga/Reuters etc. with `title`, `agencyName`, `createdAt`), `reasoning.keywords[]`, `relatedReasoning.details[]` (related stocks + relationship) | `signals detail <sym>` (since v0.5.x) | richest single endpoint for LLM context; fully Korean narrative |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/dashboard/wts/overview/indicator` | global index/macro cards | `.result` with major indices, FX, futures | none | ~12KB headline dashboard payload |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/exchange-rates` | exchange-rate summary | object under `.result` | quote KRW conversion fallback | already listed under `public` overview row |

## Quote and Symbol Detail

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/{code}` | symbol metadata | `.result` with `symbol`, `name`, `market`, `currency`, `isinCode`, `status`, `leverageFactor`, `riskLevel`, `purchasePrerequisite`, `derivativeEtf`, `optionSupported`, `daytimePriceSupported`, `nxtSupported`, `sharesOutstanding`, `listDate` | `quote get` | best starting point for product metadata |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/code-or-symbol/{codeOrSymbol}` | resolve symbol → GUID | same shape as `/api/v2/stock-infos/{code}` | symbol resolver | accepts either Toss GUID (`A005930`, `US20100311002`) or local symbol (`005930`); useful for CLI flag parsing |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos/header/{code}` | stock page header card | `.result.sections[]` typed `ETF` (`grossExpenseRatio`, `dividendYieldRatio`), `PRICE` (KRW+USD `close/high/low/high52w/low52w`), `TRADING_AMOUNT` (`ranking`, `rankingDelta`), `TRADING_STRENGTH` (`tradingStrength`) | `quote get` (since v0.5.x) | covers ETF cost/yield + 52w + trading-volume ranking + buy/sell ratio in one call |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/{code}/common` | stock detail UI metadata | `.result` with `symbol`, `name`, `badges`, `notices`, `memoCount` | `quote get` | enriched quote view |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos/{code}/wts-badges` | UI badges | `.result` badge list | `quote get` (badges) | shown alongside `stock-detail/ui/.../common` |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/stock-infos/{code}/red-flags` | risk flags | flag list | `quote get` (warnings) | authenticated; surfaces pre-trade warnings |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/product/stock-prices?meta=true&productCodes=...` | bulk price lookup | `.result[]` with `productCode`, `base`, `close`, `currency`, `exchange`, `volume`, KRW equivalents | `quote get`, watchlist | strong candidate for quote batch retrieval |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v3/stock-prices?meta=true&productCodes=...` | lightweight price + trading session times | `.result[].{tradingEnd, nextTradingStart, afterMarketClose*}` | watchlist refresh | lighter than `/details`, includes after-market close |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v3/stock-prices/details?productCodes=...` | **full per-product price** | `.result[]` with `open/high/low/close/volume/value/base`, `high52w/low52w`, `high1y/low1y`, `marketCap`, `tradingStrength`, `preDayVolume`, `afterMarket{Open,High,Low,Close}`, all KRW equivalents + `*KrwDecimal`; KR adds `upperLimit`/`lowerLimit`/`nxtSinglePrice`/`exchange` | `quote get` (since v0.5.x) | preferred single-shot quote endpoint |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v3/stock-prices/{code}/quotes` | **orderbook (호가)** | `.result.{offerPrices[], offerVolumes[], bidPrices[], bidVolumes[], offerVolume, bidVolume}` + KR adds `midPrices[]`, `midOfferVolumes[]`, `midBidVolumes[]`, `singlePrice`, `estimatedPrice`, `estimatedVolume` | `quotes book <sym>` (since v0.5.x) | **KR returns 10 levels, US returns top-of-book only**; depth param ignored. See [`order-page-deep-dive.md`](order-page-deep-dive.md#c-호가창-orderbook--시장별-깊이-차이) |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-prices/{code}/quotes` | orderbook (legacy Korean keys + timestamp) | `.result.{sellPrices[], sellQuantities[], buyPrices[], buyQuantities[], dt, isSinglePrice, sumOfSellQuantities, sumOfBuyQuantities, estimatedPrice, estimatedVolume}` | (alternative to v3) | different field names than v3; includes `dt` server timestamp |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-prices/{code}/quotes` | orderbook v1 (same shape as v2) | same as v2 | — | redundant with v2; prefer v3 unless `dt` is needed |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-prices/{code}/ticks?count=N` | **trade tick history (snapshot)** | `.result[]` newest-first, each `{time:"HH:MM:SS", code, price, priceKrw, base, baseKrw, volume, tradeType:"BUY"\|"SELL", cumulativeVolume}` | `quotes ticks <sym> [--follow]` (since v0.5.x) | no date field — dedupe by `cumulativeVolume` (monotonic per session); `--follow` streams oldest-first NDJSON |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/c-chart/{product}/{code}/{stepUnit}` | **chart candles** | `.result.{code, nextDateTime, exchangeRate, candles[]}`; candles `{dt, base, open, high, low, close, volume, amount}` (+ `sessionType` on intraday) | `chart get --tf 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y` (since v0.5.x) | supported stepUnits: `min:1/3/5/10/15/30/60`, `day:1`, `week:1`, `month:1/3`, `year:1`. Query params: `count`, `from`, `session=all\|main\|day\|pre\|after`, `investMode=integrated\|regular`, `useAdjustedRate`. `product` = `us-s`/`kr-s` (and per-exchange variants). See [`order-page-deep-dive.md`](order-page-deep-dive.md#a-차트-시간단위--전체-도메인) |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/common/stocks/mini-chart` | batch intraday mini-chart | body `{"codes":[...]}`; `.result.miniCharts[].candles[]` with `startDate`/`endDate`/`open`/`close`/`low`/`high`/`base`/`sessionType` + per-chart `tradingStart`, `tradingEnd`, `timezone`, `baseRange`, `baseStep` | `chart mini` | different field names than `/c-chart`; used by dashboard tile rows |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/analysis/productCode/{code}` | trading analysis panel | `.result` (`null` for ETFs without coverage) | none | informational; not first-class |
| `public` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v4/comments?subjectType=STOCK&subjectId=...` | community comments | object under `.result` | none | exclude from first release due to identity and moderation concerns |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/{code}/info` | **full 종목정보 deep tab** | `.result.sections[]` typed `OVERVIEW`/`INDICATORS`/`NEWS`/`ANNOUNCEMENT`/`FINANCES`/`EARNINGS_AND_CONSENSUS`/`ANALYST_OPINION`/`VALUATION_METRICS`/`COMPOSITION_OF_REVENUE`/`STABILITY`/`TOP_TIER_TREND`/`STOCK_INFO_SIGNAL`/`PRICE` | `stock info <sym>` (since v0.6.x) | returns `{"result":null}` for products without coverage. Sections are independent panels; consumers route by `section.type`. See [`stock-info-deep-tab.md`](stock-info-deep-tab.md) |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/option-infos/default-chart-option?underlyingGuid={code}` | nearest-expiry ATM option | `.result` string OCC-style `OPT_<root><YYMMDD><C|P><strikeMillis8>_<listDate>` | `options nearest-atm <sym>` (since v0.6.x) | gateway to the option product family |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/{OPT_…}` | single-option metadata (strike/expiry/bid/ask/mid/OI/contract unit/halt) | `.result.optionInstrument.{rootSymbol, underlyingSymbol, underlyingGuid, maturityDate, putCall, strikePrice, basePrice, last, bid, ask, mid, contractUnit, openInterest, halted, tradingSuspended, status, optionLiquidation.{liquidationDateTime, displayLiquidationDateTime}}` + parent `optionPennyPilotPriceSupported` | `options info <OPT_…>` (since v0.6.x) | same shape as stock `stock-infos` for the wrapper fields; option block is the value-add. See [`options.md`](options.md) |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/option-maturity-date/get-all?underlyingGuid={code}` | **option expiry ladder** | `.result.items[]` with `maturityDate`, `maturityDateTime`, `liquidationDateTime`, `displayLiquidationDateTime` ("24분 후 거래 종료"), `corporateActionDateTime`, `corporateActionName`, `displayCorporateActionName` | `options expiries <sym>` (since v0.7.x) | weeklies + monthlies + LEAPS; SNDK returned 19 expiries from this-week through 2028-12 |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/option-both-chain/get-all?underlyingGuid={code}&maturityDate=YYYY-MM-DD` | **strike chain (call + put per strike) for a single expiry** | `.result[]` flat array of `{strikePrice, callGuid:"OPT_…", putGuid:"OPT_…", callOpenInterest, putOpenInterest}` | `options chain <sym> --expiry YYYY-MM-DD` (since v0.7.x) | SNDK 2026-05-15 returned 285 strike rows. Prices NOT included — fetch separately via bulk endpoint |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-prices?codes=OPT_A,OPT_B,…` | **bulk option/stock prices (light)** | `.result.prices[].{code, base, close, changeType, currency, volume, baseKrw, closeKrw, *KrwDecimal}` | `options prices <codes>` / `options chain --with-prices` (since v0.7.x) | comma-separated `codes` (URL-encoded); chain UI batches ~58 codes/call. Lighter than `/api/v3/stock-prices/details` |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/code-or-symbol/{key}` | symbol↔productCode resolver | `.result` = full stock-info object (same as `/api/v2/stock-infos/{code}`) | (resolver helper) | strictly better than `/api/v2/search/stocks` for known codes — single round-trip, no fuzzy ranking, handles `NAS*`/`AMX*`/`NYS*`/`OPT_…` |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-prices/mainsession?codes={codes}` | main-session price (regular hours only, excludes AH) | `.result.prices[].{code, base, close, changeType, currency, tradingEnd, nextTradingStart, volume, baseKrw, closeKrw, *KrwDecimal, metaData.{afterMarketClose, afterMarketCloseKrw, …}}` | (potential future helper) | useful when caller wants the regular-session close separated from after-market drift |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/{code}/overview` | **company overview** (CEO, EV, market value, industry, description, listDate, sharesOutstanding) | `.result.{type, market.{code,displayName}, company.{code, name, englishName, fullName, fullEnglishName, industry.{code,displayName}, tics[], description, establishYear, listDate, ceo, homepageUrl, logoImageUrl, sharesOutstanding, marketValue, marketValueKrw, currency}, marketValueKrw, marketValue, enterpriseValueKrw, enterpriseValue, dataSource, listDate}` | (planned) `stock overview <sym>` | **CAPTURED 2026-05-16** — single endpoint for everything Image #3's top-card showed: CEO + 실제 기업 가치 (EV) + 시가총액 + 회사 설명 + 산업 분류 + 발행주식수 |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/wts/{code}/section-orders` | deep-tab section render order | `.result.orders[]` of `"OVERVIEW","FINANCE","REVENUE_NET_PROFIT","OPERATING_INCOME","STABILITY","FINANCIAL_STATEMENT","ESTIMATE_DATE","ESTIMATE_REVENUE","ESTIMATE_EPS","DIVIDEND_SUMMARY","DIVIDEND","EVALUATION_COMPARISON","ANALYST_REPORT"` | (orchestrator) | canonical ordering for LLM-friendly section concatenation |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/wts/{code}/investment-indicators` | key valuation/income/dividend/stability indicators | `.result.indicatorSections[].{sectionName:"가치평가"\|"수익"\|"배당"\|"안정성", data}` with per-section sub-fields (displayPer/Pbr/Psr, eps, bps, roe, dividendFrequency, dividendYieldRatio, annualCash, …) | (planned) `stock indicators <sym>` | parallel to `/info` `INDICATORS` section but with extra fields |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/companies/{companyCode}/sales-compositions` | **revenue composition by business** | `.result.{code, fiscalYear, endDate, compositions[].{business, ratio, product}, dataSource}` | (planned) | uses `companyCode` (e.g. `NAS116LTR-E0`) from `/overview` — not the productCode |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{companyCode}/tics` | TICS industry taxonomy + peer ranking within industry | `.result.{baseDate, majorList[].{id, title, description, summary, companyCount, rankings[]}}` | (planned) | rankings includes peer time series; for "동종 산업 1위/2위" labels |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/evaluation/{code}` | per-stock valuation snapshot vs. industry median | body `{}`; `.result.{per, pbr, psr, median, position:"HIGH"\|"LOW"\|"NORMAL"}` | (planned) | empty body required; companion to the wider `evaluation-comparison` endpoint |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/evaluation-comparison/{code}` | peer comparison matrix for PER/PBR/PSR | body `{}`; `.result.{selectedFactor, selectableFactors[], selectableFactorsList[][]}` with per-factor peer arrays | (planned) | richer than `/info` `VALUATION_METRICS` — switchable factor |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/revenue-and-net-profit/{code}` | quarterly + annual revenue and net-profit time series | body `{}`; per-period values | (planned) `stock financials` | granular than `/info` `FINANCES` |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/operating-income/{code}` | quarterly + annual operating income series | body `{}`; per-period values | (planned) `stock financials` | parallel to revenue-and-net-profit |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/stability/{code}` | 부채비율 / 유동비율 time series (year + quarter) | body `{}`; `liabilityByYear/Quarter[]`, `currentRatioByYear/Quarter[]` (same shape as `/info` `STABILITY`) | (planned) | year+quarter breakouts |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial-statements/comprehensive` | full comprehensive financial statements | body `{}`; multi-period statements | (planned) | drill-down beyond `/info` summary |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial-statement-records` | raw line-item financial-statement records | body `{}`; line items | (planned) | most granular tier of financials |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial/estimate/date` | latest consensus pointDate | `.result` date metadata | (planned) | gateway for the three estimate POSTs below |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial/estimate/revenue` | analyst revenue estimates time series | body `{}`; `[{pointDate, fiscalEndDate, value, isFuture, …}]` | (planned) | `valueType` 31=Y, 32/42=Q variants |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial/estimate/eps` | analyst EPS estimates time series | body `{}` | (planned) | shape mirrors revenue estimates |
| `public` | `POST` | `wts-info-api.tossinvest.com` | `/api/v2/companies/{code}/financial/estimate/operating-income` | analyst operating-income estimates | body `{}` | (planned) | shape mirrors revenue estimates |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/stock-infos/consensus/{code}` | consensus target price + past close history | `.result.{targetPrice.{mean, high, low, *Krw, currency}, pointDate, pastClosePrices[].{price, priceKrw, date}}` | (planned) | drives the "현재가 vs 목표가" overlay |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos/dividend/{code}/years` | per-year dividend records | `.result` per-year array | (planned) `stock dividends` | per-year payout records |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos/dividend/{code}/summary` | dividend summary (yield, last payout, frequency) | `.result` aggregated summary | (planned) `stock dividends` | summary card driver |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos/{code}/dividends/yield-ratio/histories` | dividend yield ratio time series | `.result` time series | (planned) `stock dividends` | history chart driver |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/wts/{code}/analyst-opinion` | analyst BUY/HOLD/SELL counts + target price | `.result.{type:"BUY"\|..., strongBuy, buy, hold, sell, strongSell, targetPrice.{USD,KRW}, description}` | (planned) `stock analyst` | richer header card than `/info.ANALYST_OPINION` |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-detail/ui/wts/{code}/analyst-reports` | analyst report metadata list | `.result` list of report items | (planned) | report titles + dates + sources |

## Rankings and Watch Surface

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/rankings/realtime/stock?size=10` | realtime ranking list | object under `.result` | none | useful for future discovery commands |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v1/stock-infos?codes=...` | bulk metadata lookup | object under `.result` | future watchlist | useful companion to bulk price lookup |
| `public` | `GET` | `wts-info-api.tossinvest.com` | `/api/v2/screener/screen/search/modal` | screener modal data | object under `.result` | none | outside first release scope |

## Account, Portfolio, Orders, Watchlist

These are approved CLI targets. Initial authenticated discovery happened on 2026-03-11 from the `/account` page after QR login.

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/account/list` | account list and primary account key | `.result.accountList`, `.result.primaryKey` | `account list` | high-value first endpoint; sanitize account identifiers |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/my-assets/summaries/markets/all/overview` | total assets and profit summary | `.result.accountNo`, `totalAssetAmount`, `evaluatedProfitAmount`, `profitRate`, `overviewByMarket` | `account summary`, `portfolio allocation` | account number appears in response |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/my-assets/summaries/markets/kr/withdrawable-amount` | KRW withdrawable amounts | `.result.amount0..amount3`, `.result.date0..date3` | `account summary` | public account summary dependency |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/my-assets/summaries/markets/us/withdrawable-amount` | USD withdrawable amounts | `.result.amount0..amount3`, `.result.date0..date3` | `account summary` | public account summary dependency |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/trading/orders/histories/all/pending` | pending order history | `.result` list | `orders list` | initial capture returned an empty list |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/common/cached-orderable-amount` | orderable buying power | `.result.orderableAmountKr`, `.result.orderableAmountUs` | `orderable` (v0.5.x) | useful for summary view |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/asset/sections/all` | account dashboard sections | body `{"types":["MIDDLE"]}` (and others) | dashboard middle banner | filter required since 2026-05-13 (#29) |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v2/dashboard/asset/sections/all` | **full portfolio sections v2** | body `{"types":["SORTED_OVERVIEW"]}` returns `.result.sections[].data` with aggregate `principalAmount/evaluatedAmount/profitLossAmount/dailyProfitLossAmount/profitLossRate/dailyProfitLossRate/...AfterFees` (KRW+USD) + `products[]` per `marketType` (`US_OPTION`/`US_STOCK`/`KR_STOCK`/`US_BOND`) + `items[]` per holding (`stockCode`, `stockIsin`, `stockSymbol`, `quantity`, `tradableQuantity`, `unsettledQuantity`, `currentPrice`, `basePrice`, `closeWithoutAfter`, `baseWithoutAfter`, `purchasePrice`, `purchaseAmount`, `evaluatedAmount{,AfterFees}`, `profitLossAmount{,AfterFees}`, `dailyProfitLossAmount`, `commission`, `commissionRate`, `buyCommission`, `sellCommission`, `tax`, `taxRate`, `delisting`, `nxtSupported`, `notice.{splitMerge,earningsAnnouncement}`, `domesticExchange`, `marketCode`, `shareHoldingsType`); plus `hiddenStock`, `usePolling`, `pollIntervalMillis:3000` | `portfolio positions` (extended in v0.5.x), `watchlist list` | **2026-05-13: empty `{}` body returns empty sections. Must pass `types`.** `SORTED_OVERVIEW` is the canonical holdings call. See [`order-page-deep-dive.md`](order-page-deep-dive.md#d-보유-주식-holdings--portfolio) |
| `auth` | `POST` | `wts-cert-api.tossinvest.com` | `/api/v1/profit/overview` | profit overview widget | body contract unknown | `portfolio allocation` | body still needs capture |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v1/account/detail` | account detail | `.result.{no, status, openDate, lastTradeDate, accountName}` | `account detail` | includes account holder name; sanitize before logging |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/dashboard/wts/overview/margin` | margin / receivable | `.result.{kr, us, total}.{message, receivable, landingUrl, invoiceType}` | `account summary` | both nullable when no margin obligations |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v1/new-watchlists?includePrice=true&lazyLoad=false` | watchlists with prices | `.result.watchlists[].items[]` with `code`, `name`, `symbol`, `prices.{base, close, baseKrw, closeKrw, currency}`, `logoImageUrl`, `hasMemo`, `ordering`, `createdAt`, plus folders like `RECENT_WATCH` | `watchlist list` | first confirmed watchlist read endpoint |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/trading/orders/histories/compact/executed?productCode={code}&timeUnit={thirty_minute\|...}&excludeSavings=false` | own executions bucketed | `.result.body[]` with `productCode`, `tradeType`, `executionAvgKrwPrice`, `executionTotalKrwPrice`, `executionAvgLocalPrice`, `executionTotalLocalAmount`, `quantity`, `executedTimeBucket` | `my fills --tf 30m` | for chart-overlay of own trades |

Watchlist-specific endpoints are still not isolated. The `/account` page did not clearly expose a standalone watchlist read path in the first authenticated capture.

## Transactions Ledger

Captured via `/my-assets` navigation on 2026-04-19. Covers trades, cash flow, dividends, and stock in/out per market.

| Status | Method | Host | Path | Purpose | Observed shape | CLI mapping | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v3/my-assets/transactions/markets/{market}` | paginated transaction ledger | `.result.body[]` with `type`, `transactionType.{code,displayName}`, `stockCode`, `stockName`, `quantity`, `amount`, `adjustedAmount`, `commissionAmount`, `totalTaxAmount`, `balanceAmount`, `date`, `dateTime`, `settlementDate`, `referenceType`, `referenceId`, `compositeKey` | `transactions list` | `market` = `kr` or `us`. Query params: `size`, `filters` (0=all, 1=trades, 2=cash/dividend, 3=stock in-out, 6=alt cash; 4/5/7 return 500), `range.from`, `range.to`. `size` is honored; `range.from` and `number` are silently ignored — Toss returns up to `size` entries within the tail of `range.to`. Items are grouped by `type` ASC (1 = trade records, 2 = cash-flow records), then DESC by `dateTime`/`date` inside each group. US `type=1` trades populate only `settlementDate` (T+2); client range-filter falls back to `compositeKey.orderDate` to match execution day. Client pages older data by re-issuing with `range.to` set to the earliest date seen, dedupes by SortKey (derived from `compositeKey`), and filters items to the caller's `[from, to]` window. Max range = 200 days (client-side guard). |
| `auth` | `GET` | `wts-api.tossinvest.com` | `/api/v3/my-assets/transactions/markets/{market}/overview` | cash overview per market | `.result` with `orderableAmount`, `withdrawableAmount.amount0..3`, `depositAmount.amount0..3`, `estimateSettlementAmount.day1..2`, `withdrawableAmountBottomSheet` | `orderable` / `transactions overview` | `depositAmount` buckets represent upcoming settlement credits; `estimateSettlementAmount` shows buy/sell amounts clearing on each upcoming settlement date. |
| `auth` | `GET` | `wts-cert-api.tossinvest.com` | `/api/v3/trading/orders/histories/compact/executed` | own-fill executions bucketed by time | `.result.body[]` with `productCode`, `tradeType`, `executionAvgKrwPrice`, `executionAvgLocalPrice`, `executionTotalKrwAmount`, `executionTotalLocalAmount`, `quantity`, `executedTimeBucket` | `my fills <sym>` (v0.5.x) | Query params: `productCode`, `timeUnit` (verified `thirty_minute`; other Toss-supported timeUnits forwarded as-is), `excludeSavings`. Useful for chart-overlay personal fill markers. |

## Chart stepUnit Vocabulary

Verified 2026-05-15 by enumerating 90 `unit × step` combinations against `/api/v1/c-chart/us-s/{code}/{unit}:{step}`. Only 12 are accepted; every other combination returns `400 {"error":{"statusCode":400,"code":"400"}}`.

| Supported stepUnit | Notes |
| --- | --- |
| `min:1`, `min:3`, `min:5`, `min:10`, `min:15`, `min:30`, `min:60` | 분봉. `min:60` is the only accepted hour-equivalent — `hour:*`, `minute:*` are rejected |
| `day:1` | 일봉 |
| `week:1` | 주봉 |
| `month:1`, `month:3` | 월봉 + 분기봉 |
| `year:1` | 연봉 |

Filter parameters:

- `session` — `all` (default), `main`, `day`, `pre`, `after`. Any other value returns 400. Each value filters the returned candles to that `sessionType` only.
- `investMode` — `integrated` (default) or `regular`. For US ETFs the two responses are identical; KR equities with NXT support may diverge.
- `useAdjustedRate` — `true`/`false` (split/dividend adjustment).
- `count`, `from` — page size and (ISO 8601 with TZ) cursor; the response's `.result.nextDateTime` is the cursor for the next older page.

See [`order-page-deep-dive.md`](order-page-deep-dive.md#a-차트-시간단위--전체-도메인) for the full enumeration.

## Realtime Strategy

Toss web does **not** stream prices over WebSocket or SSE. Realtime is `(SSE thin notification) + (REST poll)`:

| Surface | Recommended cadence | Endpoint |
| --- | --- | --- |
| Last/OHLC/체결강도/marketCap | 3-5s during regular session, 10-15s pre/after | `GET /api/v3/stock-prices/details?productCodes={code}` |
| Orderbook (호가) | 1-2s | `GET /api/v3/stock-prices/{code}/quotes` |
| Recent ticks | 2-3s | `GET /api/v2/stock-prices/{code}/ticks?count={N}` — dedupe by `cumulativeVolume` |
| Intraday candle | 1× per minute, plus final at bucket close | `GET /api/v1/c-chart/{product}/{code}/min:{N}?count=2` |
| Push triggers | continuous | `GET https://sse-message.tossinvest.com/api/v1/wts-notification` (see [`push-events.md`](push-events.md)) |

Polling these directly produces a complete realtime view without needing the SSE channel; the SSE channel is mostly used to invalidate **own portfolio/orders** state, not market data.

**CLI mappings (added in PR5):**
- `Last/OHLC/체결강도/marketCap` → `tossctl quote get --follow` (default 3s)
- `Orderbook` → `tossctl quotes book --follow` (default 1s)
- `Recent ticks` → `tossctl quotes ticks --follow` (default 2s)
- `Intraday candle` → `tossctl chart get --tf <1m|5m|15m|...> --follow` (default 60s)
- `Push triggers` → `tossctl push listen`

## Indicators / Signals

Toss does **not** expose server-computed technical indicators (MA / RSI / MACD / Bollinger). The pro chart computes everything client-side from `/c-chart` OHLCV. User-side chart layout/state is persisted at `/api/v1/properties/member/{prochart-setting,multi-prochart-setting,mts-prochart-setting}`.

In place of classic indicators, Toss exposes a richer **AI signal** surface (Korean-first, news-grounded):

| Endpoint | Use |
| --- | --- |
| `POST /api/v1/dashboard/wts/overview/ai-signals` body `{"productCodes":[...]}` | one-line `reasoningDescription` per code |
| `GET /api/v1/dashboard/wts/overview/ai-signals/detail?productCode={code}&productType=STOCKS` | full 3-bullet narrative + news headlines + related stocks |
| `POST /api/v2/dashboard/wts/overview/signals` body `{"productCodes":[...]}` | scheduled event signals (earnings, disclosure) keyed by `signalId` |

The `ai-signals/detail` payload is the highest-density single call for LLM context.

## Read-Only Policy Notes

The Go client should only admit endpoints that are:

- observed in this catalog
- explicitly classified as read-only
- mapped to an approved CLI command

The following classes stay blocked:

- any order placement endpoint
- any order modification or cancelation endpoint
- any watchlist mutation endpoint unless scope changes
- telemetry endpoints
- comment posting or social actions

## Next Catalog Work

1. Capture authenticated account flows with a clean browser session.
2. Record additional `id` values for `dashboard/wts/overview/ranking` (only `biggest_total_amount` captured so far).
3. Promote quote-related endpoints into typed Go client methods (`/api/v3/stock-prices/details`, `/api/v3/stock-prices/{code}/quotes`, `/api/v2/stock-prices/{code}/ticks`).
4. Promote `/api/v1/c-chart/{product}/{code}/{stepUnit}` to a typed Go client with the full 12-value stepUnit enum and 5-value session enum.
5. Promote `/api/v2/dashboard/asset/sections/all {"types":["SORTED_OVERVIEW"]}` to a typed portfolio client.
6. Add stable fixture names for every supported endpoint family.
7. Map the `signalId` numeric ranges from `/dashboard/wts/overview/signals` to categories (실적 발표 = 6000000-range observed; disclose/dividend/insider categories still unknown).
