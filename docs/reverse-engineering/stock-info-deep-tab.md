# Stock Info Deep Tab Reverse Engineering

**Source:** Image #3 (SNDK 종목정보 탭, 2026-05-16 캡쳐) + probe `.captures/2026-05-16/sndk-options-stockinfo/`.

The page tab labeled "종목정보" at `https://www.tossinvest.com/stocks/{productCode}/info` aggregates ~13 panels (회사 개요 / 주요 지표 / 매출 구성 / 재무 / 실적 / 동종업계 / 애널리스트 / 시그널 / 뉴스 / 공시 / 가격 범위 / 안정성 / 상위 트레이더 동향). **One endpoint returns the entire payload:**

## Endpoint

```
GET https://wts-info-api.tossinvest.com/api/v1/stock-detail/ui/{productCode}/info
```

- Auth: **public** (no cookies required)
- Method: GET
- Response: `application/json`
- Observed productCodes: `NAS0250224006` (SNDK), `NAS116LTR-E0` (returns `null` — that's the `companyCode` form; **always use the `productCode` form**)
- Returns `{"result":null}` for productCodes that have no info coverage (e.g. some ETFs).

## Response Shape

`result.sections[]` — array of typed sections. Each `section.type` indicates the panel and `section.data` is the panel-specific payload. Order of sections in SNDK capture:

| `type` | Panel (UI label) | Key fields |
| --- | --- | --- |
| `PRICE` | 가격 범위 (1일 / 52주 / 1년 high·low + open/close) | `low`, `high`, `open`, `close`, `volume`, `amount`, `prevDayVolume`, `high52Week`, `low52Week`, `high1y`, `low1y` (+ all `*Krw` companions) |
| `STOCK_INFO_SIGNAL` | 시그널 chips at top | `[]` of `{signalCategory:"뉴스"\|"시세"\|"펀더멘털", signalSubcategory, signalLabel, signalInfo, signalShortInfo, signalId, signalVersion, sourceId, link, dateTime}` — same shape as `/dashboard/wts/overview/ai-signals` items, repeated here for the tab. |
| `NEWS` | 뉴스 panel | `[{type:"NEWS", data:{LATEST:[…], RELEVANT:[…]}}]` where each news item has `{id, title, summary, imageUrls[], source.{name, faviconUrl}, createdAt}`. |
| `ANNOUNCEMENT` | 공시 panel (SEC + 어닝콜) | `[{id, title, summary, form:"8-K"\|"EARNINGS"\|…, reportId, reportItem, useNewApi, companyCode, stockCode, earningCall:{status,landingUrl,iconUrl,text,liveAt,zonedLiveAt,title,secondTitle,reportTitle}\|null, createdAt}]` |
| `INDICATORS` | 주요 지표 (시가총액 / 배당수익률 / PBR / PER / ROE / PSR) | `{values:[{label, value:number, displayValue:string, dollarValue:number\|null, displayDollarValue:string, pointDate}]}` |
| `VALUATION_METRICS` | 동종업계 비교 | `{values:[{title:업종명, medianPer, medianPbr, medianPsr, companyInfos:[{companyName, index, description, per, pbr, psr, isTarget}]}]}` |
| `EARNINGS_AND_CONSENSUS` | 실적 + 컨센서스 (분기/연 + 매출/EPS/영업이익) | `{yearEarningsAndConsensus, quarterEarningsAndConsensus}` each with `revenueKrw, revenueConsensusKrw, revenueConsensusKrwDifferencePercent, revenueUsd, revenueConsensusUsd, operatingIncomeKrw, operatingIncomeConsensusKrw, epsUsd, epsConsensusUsd, epsKrw, epsConsensusKrw, …` — each is `[{pointDate, fiscalEndDate, fiscalYear, fiscalMonth, fiscalQuarter, calendarYear, calendarQuarter, calendarMonth, valueType, value, expectedReleaseDate, isFuture}]` |
| `FINANCES` | 재무 (분기/연 매출/영업이익/순익) | `{listDate, values:[{period:"Q"\|"Y", values:[{date, revenue, revenueKrw, operatingIncome, operatingIncomeKrw, netProfit, netProfitKrw, isConsensus:bool}]}]}` |
| `STABILITY` | 부채비율 / 유동비율 | `{listedAt, hasMore, liabilityByYear, liabilityByQuarter, currentRatioByYear, currentRatioByQuarter}` each `[{calendarYear, calendarMonth, fiscalYear, fiscalMonth, displayDate, value}]` |
| `ANALYST_OPINION` | 애널리스트 분석 | `{type:"BUY"\|"HOLD"\|"SELL", strongSell, sell, hold, buy, strongBuy, targetPrice:{USD,KRW}, description:"애널리스트 22명 중 18명이 구매 의견을 냈어요."}` |
| `TOP_TIER_TREND` | 상위 트레이더 매수/매도 추이 | `{buyVolume, sellVolume, updatedAt, status:"NORMAL"}` |
| `OVERVIEW` | 회사 개요 + 분류 (TICS) | `{logoUrl, name, subTitle, description, link, extraInfo:[], tics:{baseDate, major:[{id, title, description, imageUrl, imageBackgroundColor, imageBackground:{light,dark}}], minor:[]}}` |
| `COMPOSITION_OF_REVENUE` | 매출 구성 | `{title, description:"YYYY.MM 기준 (출처: …)", isETF:bool, items:[{name, ratio, description}]}` |

## Image #3 → Coverage Map

| Image #3 element | Section | Field |
| --- | --- | --- |
| 시가총액 306조 1,466억원 | `INDICATORS` | `values[label=시가총액].value` (also `quote get` `market_cap_krw`) |
| 시가총액 순위 27위 ▼1 | (not in this endpoint) | comes from `/api/v1/stock-infos/header/{code}` `TRADING_AMOUNT.ranking` |
| 1일 범위 / 52주 범위 | `PRICE` | `low`/`high` + `low52Week`/`high52Week` |
| 거래대금 4위 | (not in this endpoint) | `/api/v1/stock-infos/header/{code}` `TRADING_AMOUNT.ranking` |
| 체결강도 105.93% | (not in this endpoint) | `/api/v3/stock-prices/details` `tradingStrength` |
| 회사명 SANDISK CORP | (not in this endpoint, English) | `/api/v2/stock-infos/{code}` `englishName` (`샌디스크` 한글은 `OVERVIEW.name`) |
| 회사 설명 "SD카드, USB 플래시 드라이브 등의 메모리 제품을 판매하는 회사" | `OVERVIEW` | `description` |
| 출처 "연합인포맥스 및 기업 IR자료" | `COMPOSITION_OF_REVENUE` | `description` |
| 시가총액 (306조 1,466억원) | `INDICATORS` | `시가총액` row |
| 실제 기업 가치 (151조 6,071억원) | `/api/v2/stock-infos/{code}/overview` | `enterpriseValueKrw` / `enterpriseValue` (USD) — **CAPTURED 2026-05-16** |
| 기업명 SANDISK CORP | `/api/v2/stock-infos/{code}/overview` | `company.fullEnglishName` (also `englishName` in stock-infos) |
| 대표이사 David V. Goeckeler | `/api/v2/stock-infos/{code}/overview` | `company.ceo` — **CAPTURED 2026-05-16** |
| 회사 설명 ("SD카드, USB 플래시…") | `/api/v2/stock-infos/{code}/overview` | `company.description` |
| 상장일 2025-02-24 | `/api/v2/stock-infos/{code}/overview` | `company.listDate` + root `listDate` |
| 설립연도 / 산업 분류 / 홈페이지 | `/api/v2/stock-infos/{code}/overview` | `company.{establishYear, industry.{code,displayName}, homepageUrl}` |
| 발행주식수 148,089,758주 | `/api/v2/stock-infos/{code}/overview` | `company.sharesOutstanding` |
| 시가총액 (USD + KRW) | `/api/v2/stock-infos/{code}/overview` | root `marketValue` + `marketValueKrw` |
| 출처 ("출처: 연합인포맥스 및 기업 IR자료") | `/api/v2/stock-infos/{code}/overview` | `dataSource` |
| 매출 구성 (좌측 서브탭) | `/api/v1/companies/{companyCode}/sales-compositions` | `compositions[].{business, ratio}` + `fiscalYear`, `endDate`, `dataSource` |
| TICS 분류 (산업) | `/api/v2/companies/{companyCode}/tics` | `majorList[].{id, title, summary, companyCount, rankings[]}` |
| 주요 지표 (시가총액/PER/PBR/PSR/ROE/EPS/BPS/배당) | `/api/v1/stock-detail/ui/wts/{code}/investment-indicators` | `indicatorSections[].{sectionName, data}` — 가치평가 / 수익 / 배당 / 안정성 분리 |
| Valuation (PER/PBR/PSR + 업종 중앙값 + position) | `POST /api/v2/stock-infos/evaluation/{code}` | `{per, pbr, psr, median, position:"HIGH"\|"LOW"\|"NORMAL"}` |
| 동종업계 비교 | `POST /api/v2/stock-infos/evaluation-comparison/{code}` | `selectableFactors[]`, `selectableFactorsList[][]` (per-factor peer list) |
| 재무 (분기/연 매출/영업이익/순익) | `POST /api/v2/stock-infos/revenue-and-net-profit/{code}` + `POST /api/v2/stock-infos/operating-income/{code}` | per-period values |
| 재무제표 전체 | `POST /api/v2/companies/{code}/financial-statements/comprehensive` + `POST /api/v2/companies/{code}/financial-statement-records` | full statement records |
| 안정성 (부채비율 / 유동비율) | `POST /api/v2/stock-infos/stability/{code}` | year + quarter time series |
| 컨센서스 / 추정치 | `GET /api/v2/companies/{code}/financial/estimate/date` + `POST .../estimate/revenue` + `POST .../estimate/eps` + `POST .../estimate/operating-income` | each returns `[{pointDate, fiscalEndDate, value, isFuture, …}]` |
| 컨센서스 요약 (목표가/등락 이력) | `/api/v2/stock-infos/consensus/{code}` | `targetPrice.{mean, high, low, *Krw, currency}` + `pastClosePrices[]` |
| 배당 history (분기/연 + yield ratio) | `/api/v1/stock-infos/dividend/{code}/years` + `/api/v1/stock-infos/dividend/{code}/summary` + `/api/v1/stock-infos/{code}/dividends/yield-ratio/histories` | dividend amounts by period + yield-ratio time series |
| 애널리스트 의견 (BUY/HOLD/SELL counts + target price) | `/api/v1/stock-detail/ui/wts/{code}/analyst-opinion` | `{type, strongBuy, buy, hold, sell, strongSell, targetPrice.{USD,KRW}, description}` |
| 애널리스트 보고서 (목록) | `/api/v1/stock-detail/ui/wts/{code}/analyst-reports` | list of report metadata |
| 시그널 chips + 뉴스 + 공시 + TOP_TIER_TREND 등 나머지 | `/api/v1/stock-detail/ui/{code}/info` | 기존 PR6 endpoint — 13 sections JSON |

## Section orchestration

`GET /api/v1/stock-detail/ui/wts/{code}/section-orders` returns the canonical render order for the deep tab:

```
["OVERVIEW","FINANCE","REVENUE_NET_PROFIT","OPERATING_INCOME","STABILITY","FINANCIAL_STATEMENT","ESTIMATE_DATE","ESTIMATE_REVENUE","ESTIMATE_EPS","DIVIDEND_SUMMARY","DIVIDEND","EVALUATION_COMPARISON","ANALYST_REPORT"]
```

CLI consumers can follow this ordering when assembling LLM context.

## All gaps now closed (2026-05-16 capture)

Previous gap list eliminated:
- ✅ Enterprise Value — `/overview.enterpriseValueKrw`
- ✅ CEO — `/overview.company.ceo`
- ✅ Dividend history — `/dividend/{code}/years` + `/dividend/{code}/summary` + `/yield-ratio/histories`

The 종목정보 tab is now fully reverse-engineered. Image #3 has 100% endpoint coverage.

## CLI Mapping

`/api/v1/stock-detail/ui/{code}/info` will be exposed as `tossctl stock info <symbol>`:

- snapshot mode: returns the entire `sections` array as JSON; table mode renders one summary line per section.
- `--section <name>` flag (optional): filter to a single section (e.g. `--section FINANCES`).
- Same `--output table|json|csv` convention as the other read commands.

Use `quote get` for live price + 체결강도 (those aren't in `/info`).
