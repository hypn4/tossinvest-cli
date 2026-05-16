package domain

import (
	"encoding/json"
	"time"
)

type Account struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	Markets     []string `json:"markets,omitempty"`
	Primary     bool     `json:"primary,omitempty"`
}

type AccountSummary struct {
	TotalAssetAmount      float64                         `json:"total_asset_amount"`
	EvaluatedProfitAmount float64                         `json:"evaluated_profit_amount"`
	ProfitRate            float64                         `json:"profit_rate"`
	OrderableAmountKRW    float64                         `json:"orderable_amount_krw"`
	OrderableAmountUSD    float64                         `json:"orderable_amount_usd"`
	WithdrawableKR        map[string]any                  `json:"withdrawable_kr,omitempty"`
	WithdrawableUS        map[string]any                  `json:"withdrawable_us,omitempty"`
	Markets               map[string]AccountMarketSummary `json:"markets,omitempty"`
}

type AccountMarketSummary struct {
	Market                string  `json:"market"`
	PendingBuyOrderAmount float64 `json:"pending_buy_order_amount"`
	EvaluatedAmount       float64 `json:"evaluated_amount"`
	PrincipalAmount       float64 `json:"principal_amount"`
	EvaluatedProfitAmount float64 `json:"evaluated_profit_amount"`
	ProfitRate            float64 `json:"profit_rate"`
	TotalAssetAmount      float64 `json:"total_asset_amount"`
	OrderableAmountKRW    float64 `json:"orderable_amount_krw"`
	OrderableAmountUSD    float64 `json:"orderable_amount_usd"`
}

type Position struct {
	ProductCode     string  `json:"product_code,omitempty"`
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name,omitempty"`
	MarketType      string  `json:"market_type,omitempty"`
	MarketCode      string  `json:"market_code,omitempty"`
	Quantity        float64 `json:"quantity"`
	TradableQuantity   float64 `json:"tradable_quantity,omitempty"`
	UnsettledQuantity  float64 `json:"unsettled_quantity,omitempty"`
	AveragePrice    float64 `json:"average_price,omitempty"`
	CurrentPrice    float64 `json:"current_price,omitempty"`
	CloseWithoutAfter float64 `json:"close_without_after,omitempty"`
	MarketValue     float64 `json:"market_value,omitempty"`
	MarketValueAfterFees float64 `json:"market_value_after_fees,omitempty"`
	UnrealizedPnL   float64 `json:"unrealized_pnl,omitempty"`
	UnrealizedPnLAfterFees float64 `json:"unrealized_pnl_after_fees,omitempty"`
	ProfitRate      float64 `json:"profit_rate,omitempty"`
	ProfitRateAfterFees float64 `json:"profit_rate_after_fees,omitempty"`
	DailyProfitLoss float64 `json:"daily_profit_loss,omitempty"`
	DailyProfitRate float64 `json:"daily_profit_rate,omitempty"`
	EstimatedCommission float64 `json:"estimated_commission,omitempty"`
	CommissionRate     float64 `json:"commission_rate,omitempty"`
	EstimatedTax       float64 `json:"estimated_tax,omitempty"`
	TaxRate            float64 `json:"tax_rate,omitempty"`
	Delisting          bool    `json:"delisting,omitempty"`
	NXTSupported       bool    `json:"nxt_supported,omitempty"`
	NoticeSplitMerge          bool `json:"notice_split_merge,omitempty"`
	NoticeEarningsAnnouncement bool `json:"notice_earnings_announcement,omitempty"`

	AveragePriceUSD    float64 `json:"average_price_usd,omitempty"`
	CurrentPriceUSD    float64 `json:"current_price_usd,omitempty"`
	CloseWithoutAfterUSD float64 `json:"close_without_after_usd,omitempty"`
	MarketValueUSD     float64 `json:"market_value_usd,omitempty"`
	MarketValueAfterFeesUSD float64 `json:"market_value_after_fees_usd,omitempty"`
	UnrealizedPnLUSD   float64 `json:"unrealized_pnl_usd,omitempty"`
	UnrealizedPnLAfterFeesUSD float64 `json:"unrealized_pnl_after_fees_usd,omitempty"`
	ProfitRateUSD      float64 `json:"profit_rate_usd,omitempty"`
	ProfitRateAfterFeesUSD float64 `json:"profit_rate_after_fees_usd,omitempty"`
	DailyProfitLossUSD float64 `json:"daily_profit_loss_usd,omitempty"`
	DailyProfitRateUSD float64 `json:"daily_profit_rate_usd,omitempty"`
	EstimatedCommissionUSD float64 `json:"estimated_commission_usd,omitempty"`
	EstimatedTaxUSD        float64 `json:"estimated_tax_usd,omitempty"`
}

type Order struct {
	ID                    string          `json:"id"`
	ResolvedFromID        string          `json:"resolved_from_id,omitempty"`
	Symbol                string          `json:"symbol"`
	Name                  string          `json:"name,omitempty"`
	Market                string          `json:"market,omitempty"`
	Side                  string          `json:"side,omitempty"`
	Status                string          `json:"status,omitempty"`
	Quantity              float64         `json:"quantity,omitempty"`
	FilledQuantity        float64         `json:"filled_quantity,omitempty"`
	Price                 float64         `json:"price,omitempty"`
	AverageExecutionPrice float64         `json:"average_execution_price,omitempty"`
	OrderDate             string          `json:"order_date,omitempty"`
	SubmittedAt           *time.Time      `json:"submitted_at,omitempty"`
	Raw                   json.RawMessage `json:"raw,omitempty"`
}

type WatchlistItem struct {
	Group    string  `json:"group,omitempty"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name,omitempty"`
	Currency string  `json:"currency,omitempty"`
	Base     float64 `json:"base,omitempty"`
	Last     float64 `json:"last,omitempty"`
}

type Transaction struct {
	Type             string          `json:"type"`
	Category         string          `json:"category"`
	Code             string          `json:"code,omitempty"`
	DisplayName      string          `json:"display_name,omitempty"`
	DisplayType      string          `json:"display_type,omitempty"`
	Summary          string          `json:"summary,omitempty"`
	Market           string          `json:"market"`
	Currency         string          `json:"currency"`
	StockCode        string          `json:"stock_code,omitempty"`
	StockName        string          `json:"stock_name,omitempty"`
	Quantity         float64         `json:"quantity,omitempty"`
	Amount           float64         `json:"amount"`
	AdjustedAmount   float64         `json:"adjusted_amount"`
	CommissionAmount float64         `json:"commission_amount,omitempty"`
	TaxAmount        float64         `json:"tax_amount,omitempty"`
	BalanceAmount    float64         `json:"balance_amount,omitempty"`
	Date             string          `json:"date,omitempty"`
	DateTime         string          `json:"datetime,omitempty"`
	OrderDate        string          `json:"order_date,omitempty"`
	SettlementDate   string          `json:"settlement_date,omitempty"`
	TradeType        string          `json:"trade_type,omitempty"`
	ReferenceType    string          `json:"reference_type,omitempty"`
	ReferenceID      string          `json:"reference_id,omitempty"`
	SortKey          string          `json:"sort_key,omitempty"`
	Raw              json.RawMessage `json:"raw,omitempty"`
}

type TransactionPage struct {
	Market   string        `json:"market"`
	Items    []Transaction `json:"items"`
	LastPage bool          `json:"last_page"`
	Next     *PagingParam  `json:"next,omitempty"`
}

type PagingParam struct {
	Number  int    `json:"number,omitempty"`
	Size    int    `json:"size,omitempty"`
	Key     string `json:"key,omitempty"`
	Filters string `json:"filters,omitempty"`
	Type    string `json:"type,omitempty"`
}

type TransactionOverview struct {
	Market                  string                         `json:"market"`
	OrderableKRW            float64                        `json:"orderable_krw"`
	OrderableUSD            float64                        `json:"orderable_usd"`
	Withdrawable            []SettlementBucket             `json:"withdrawable,omitempty"`
	DisplayWithdrawable     []SettlementBucket             `json:"display_withdrawable,omitempty"`
	Deposit                 []SettlementBucket             `json:"deposit,omitempty"`
	EstimateSettlement      []SettlementEstimate           `json:"estimate_settlement,omitempty"`
	WithdrawableBottomSheet []WithdrawableBottomSheetEntry `json:"withdrawable_bottom_sheet,omitempty"`
}

type SettlementBucket struct {
	Date string  `json:"date,omitempty"`
	KRW  float64 `json:"krw,omitempty"`
	USD  float64 `json:"usd,omitempty"`
}

type SettlementEstimate struct {
	Date       string  `json:"date,omitempty"`
	BuyAmount  float64 `json:"buy_amount,omitempty"`
	SellAmount float64 `json:"sell_amount,omitempty"`
}

type WithdrawableBottomSheetEntry struct {
	Title string  `json:"title"`
	KRW   float64 `json:"krw,omitempty"`
	USD   float64 `json:"usd,omitempty"`
}

type Money struct {
	KRW float64 `json:"krw,omitempty"`
	USD float64 `json:"usd,omitempty"`
}

type OrderableSummary struct {
	OrderableKR Money               `json:"orderable_kr"`
	OrderableUS Money               `json:"orderable_us"`
	KR          TransactionOverview `json:"kr_overview"`
	US          TransactionOverview `json:"us_overview"`
	FetchedAt   time.Time           `json:"fetched_at"`
}

type CompactExecution struct {
	ProductCode             string    `json:"product_code"`
	TradeType               string    `json:"trade_type"` // "buy" or "sell"
	ExecutionAvgKRWPrice    float64   `json:"execution_avg_krw_price"`
	ExecutionAvgLocalPrice  float64   `json:"execution_avg_local_price"`
	ExecutionTotalKRWAmount float64   `json:"execution_total_krw_amount"`
	ExecutionTotalLocal     float64   `json:"execution_total_local"`
	Quantity                float64   `json:"quantity"`
	BucketStart             time.Time `json:"bucket_start"`
}

type Quote struct {
	ProductCode    string    `json:"product_code,omitempty"`
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name,omitempty"`
	MarketCode     string    `json:"market_code,omitempty"`
	Market         string    `json:"market,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	ReferencePrice float64   `json:"reference_price,omitempty"`
	Last           float64   `json:"last,omitempty"`
	Change         float64   `json:"change,omitempty"`
	ChangeRate     float64   `json:"change_rate,omitempty"`
	Volume         float64   `json:"volume,omitempty"`
	Status         string    `json:"status,omitempty"`
	BadgeCount     int       `json:"badge_count,omitempty"`
	NoticeCount    int       `json:"notice_count,omitempty"`
	FetchedAt      time.Time `json:"fetched_at"`

	Open             float64 `json:"open,omitempty"`
	High             float64 `json:"high,omitempty"`
	Low              float64 `json:"low,omitempty"`
	Value            float64 `json:"value,omitempty"`
	High52W          float64 `json:"high_52w,omitempty"`
	Low52W           float64 `json:"low_52w,omitempty"`
	High1Y           float64 `json:"high_1y,omitempty"`
	Low1Y            float64 `json:"low_1y,omitempty"`
	MarketCap        float64 `json:"market_cap,omitempty"`
	TradingStrength  float64 `json:"trading_strength,omitempty"`
	PreDayVolume     float64 `json:"pre_day_volume,omitempty"`
	UpperLimit       float64 `json:"upper_limit,omitempty"`
	LowerLimit       float64 `json:"lower_limit,omitempty"`
	AfterMarketOpen  float64 `json:"after_market_open,omitempty"`
	AfterMarketHigh  float64 `json:"after_market_high,omitempty"`
	AfterMarketLow   float64 `json:"after_market_low,omitempty"`
	AfterMarketClose float64 `json:"after_market_close,omitempty"`
	LastKRW          float64 `json:"last_krw,omitempty"`
	OpenKRW          float64 `json:"open_krw,omitempty"`
	HighKRW          float64 `json:"high_krw,omitempty"`
	LowKRW           float64 `json:"low_krw,omitempty"`
	ReferencePriceKRW float64 `json:"reference_price_krw,omitempty"`
	ValueKRW         float64 `json:"value_krw,omitempty"`
	MarketCapKRW     float64 `json:"market_cap_krw,omitempty"`
	GrossExpenseRatio float64 `json:"gross_expense_ratio,omitempty"`
	DividendYieldRate float64 `json:"dividend_yield_rate,omitempty"`
	TradingAmountRank int     `json:"trading_amount_rank,omitempty"`
}

type Candle struct {
	DateTime    string  `json:"datetime"`
	SessionType string  `json:"session_type,omitempty"`
	Base        float64 `json:"base,omitempty"`
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      float64 `json:"volume,omitempty"`
	Amount      float64 `json:"amount,omitempty"`
}

type Chart struct {
	ProductCode  string    `json:"product_code"`
	Symbol       string    `json:"symbol,omitempty"`
	Name         string    `json:"name,omitempty"`
	Market       string    `json:"market,omitempty"`
	Currency     string    `json:"currency,omitempty"`
	Unit         string    `json:"unit"`
	Step         int       `json:"step"`
	Session      string    `json:"session,omitempty"`
	InvestMode   string    `json:"invest_mode,omitempty"`
	UseAdjusted  bool      `json:"use_adjusted_rate"`
	ExchangeRate float64   `json:"exchange_rate,omitempty"`
	NextDateTime string    `json:"next_datetime,omitempty"`
	Candles      []Candle  `json:"candles"`
	FetchedAt    time.Time `json:"fetched_at"`
}

type OrderBookLevel struct {
	Price    float64 `json:"price"`
	PriceKRW float64 `json:"price_krw,omitempty"`
	Volume   float64 `json:"volume"`
}

type OrderBook struct {
	ProductCode     string           `json:"product_code"`
	Symbol          string           `json:"symbol,omitempty"`
	Name            string           `json:"name,omitempty"`
	Market          string           `json:"market,omitempty"`
	Currency        string           `json:"currency,omitempty"`
	Last            float64          `json:"last,omitempty"`
	LastKRW         float64          `json:"last_krw,omitempty"`
	Offers          []OrderBookLevel `json:"offers"`
	Bids            []OrderBookLevel `json:"bids"`
	OfferVolumeSum  float64          `json:"offer_volume_sum,omitempty"`
	BidVolumeSum    float64          `json:"bid_volume_sum,omitempty"`
	SinglePrice     bool             `json:"single_price,omitempty"`
	EstimatedPrice  float64          `json:"estimated_price,omitempty"`
	EstimatedVolume float64          `json:"estimated_volume,omitempty"`
	FetchedAt       time.Time        `json:"fetched_at"`
}

type Tick struct {
	Time             string    `json:"time"`
	ProductCode      string    `json:"product_code"`
	Price            float64   `json:"price"`
	PriceKRW         float64   `json:"price_krw,omitempty"`
	Base             float64   `json:"base,omitempty"`
	Volume           float64   `json:"volume"`
	TradeType        string    `json:"trade_type"`
	CumulativeVolume float64   `json:"cumulative_volume"`
	FetchedAt        time.Time `json:"fetched_at,omitempty"`
}

type Signal struct {
	ProductCode          string `json:"product_code"`
	ReasoningDescription string `json:"reasoning_description"`
}

type SignalNews struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	AgencyName string    `json:"agency_name"`
	Title      string    `json:"title"`
	FaviconURL string    `json:"favicon_url,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

type RelatedSignal struct {
	AssetCode   string   `json:"asset_code"`
	AssetName   string   `json:"asset_name"`
	StockCode   string   `json:"stock_code,omitempty"`
	StockSymbol string   `json:"stock_symbol,omitempty"`
	Relation    string   `json:"relation,omitempty"`
	Description []string `json:"description,omitempty"`
}

type SignalDetail struct {
	ProductCode      string          `json:"product_code"`
	AssetName        string          `json:"asset_name,omitempty"`
	SignalID         string          `json:"signal_id,omitempty"`
	SignalDirection  int             `json:"signal_direction"` // 1 bullish, -1 bearish
	CreatedAt        time.Time       `json:"created_at,omitempty"`
	Description      string          `json:"description"`
	DescriptionItems []string        `json:"description_items,omitempty"`
	ProfitLossRate   float64         `json:"profit_loss_rate,omitempty"`
	News             []SignalNews    `json:"news,omitempty"`
	Keywords         []string        `json:"keywords,omitempty"`
	Related          []RelatedSignal `json:"related,omitempty"`
	FetchedAt        time.Time       `json:"fetched_at"`
}

type EventSignal struct {
	ProductCode string    `json:"product_code"`
	SignalLabel string    `json:"signal_label"` // e.g. "소식"
	SignalInfo  string    `json:"signal_info"`  // e.g. "실적이 발표됐어요. ..."
	SignalID    int64     `json:"signal_id"`
	DateTime    time.Time `json:"datetime"`
}

// StockInfoDetail is the full 종목정보 deep-tab payload. Sections are
// untyped on purpose: there are 13+ section types and the schemas vary
// independently (FINANCES, EARNINGS_AND_CONSENSUS, etc.). Consumers
// inspect Sections[i].Type and unmarshal Data themselves.
//
// Endpoint: GET /api/v1/stock-detail/ui/{productCode}/info
type StockInfoDetail struct {
	ProductCode string             `json:"product_code"`
	Sections    []StockInfoSection `json:"sections"`
	FetchedAt   time.Time          `json:"fetched_at"`
}

type StockInfoSection struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// OptionInstrument captures the option-specific metadata returned by
// /api/v2/stock-infos/{OPT_…} under result.optionInstrument.
type OptionInstrument struct {
	ProductCode         string    `json:"product_code"`
	MarketCode          string    `json:"market_code,omitempty"`
	RootSymbol          string    `json:"root_symbol,omitempty"`
	Name                string    `json:"name,omitempty"`
	FullName            string    `json:"full_name,omitempty"`
	CompleteName        string    `json:"complete_name,omitempty"`
	UnderlyingSymbol    string    `json:"underlying_symbol,omitempty"`
	UnderlyingGuid      string    `json:"underlying_guid,omitempty"`
	UnderlyingName      string    `json:"underlying_name,omitempty"`
	MaturityDate        string    `json:"maturity_date,omitempty"`
	MaturityDateTime    string    `json:"maturity_date_time,omitempty"`
	PutCall             string    `json:"put_call,omitempty"`
	StrikePrice         float64   `json:"strike_price,omitempty"`
	BasePrice           float64   `json:"base_price,omitempty"`
	Last                float64   `json:"last,omitempty"`
	Bid                 float64   `json:"bid,omitempty"`
	Ask                 float64   `json:"ask,omitempty"`
	Mid                 float64   `json:"mid,omitempty"`
	ContractUnit        float64   `json:"contract_unit,omitempty"`
	OpenInterest        int       `json:"open_interest,omitempty"`
	Halted              bool      `json:"halted,omitempty"`
	TradingSuspended    bool      `json:"trading_suspended,omitempty"`
	BuySuspended        bool      `json:"buy_suspended,omitempty"`
	SellSuspended       bool      `json:"sell_suspended,omitempty"`
	Status              string    `json:"status,omitempty"`
	Overtime            bool      `json:"overtime,omitempty"`
	LiquidationDisplay  string    `json:"liquidation_display,omitempty"`
	LiquidationDateTime string    `json:"liquidation_date_time,omitempty"`
	PennyPilot          bool      `json:"penny_pilot,omitempty"`
	FetchedAt           time.Time `json:"fetched_at"`
}

// OptionExpiry is one expiry-ladder entry returned by
// /api/v1/option-maturity-date/get-all.
type OptionExpiry struct {
	MaturityDate                string `json:"maturity_date"`
	MaturityDateTime            string `json:"maturity_date_time,omitempty"`
	LiquidationDateTime         string `json:"liquidation_date_time,omitempty"`
	DisplayLiquidationDateTime  string `json:"display_liquidation_date_time,omitempty"`
	CorporateActionDateTime     string `json:"corporate_action_date_time,omitempty"`
	CorporateActionName         string `json:"corporate_action_name,omitempty"`
	DisplayCorporateActionName  string `json:"display_corporate_action_name,omitempty"`
}

// OptionChainRow is one strike row returned by /api/v1/option-both-chain/get-all.
type OptionChainRow struct {
	StrikePrice      float64      `json:"strike_price"`
	CallGuid         string       `json:"call_guid,omitempty"`
	PutGuid          string       `json:"put_guid,omitempty"`
	CallOpenInterest int          `json:"call_open_interest,omitempty"`
	PutOpenInterest  int          `json:"put_open_interest,omitempty"`
	CallPrice        *OptionPrice `json:"call_price,omitempty"`
	PutPrice         *OptionPrice `json:"put_price,omitempty"`
}

// OptionPrice is one price row returned by /api/v2/stock-prices (bulk).
type OptionPrice struct {
	Code             string  `json:"code"`
	Base             float64 `json:"base,omitempty"`
	Close            float64 `json:"close,omitempty"`
	ChangeType       string  `json:"change_type,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Volume           float64 `json:"volume,omitempty"`
	BaseKrw          float64 `json:"base_krw,omitempty"`
	CloseKrw         float64 `json:"close_krw,omitempty"`
	BaseKrwDecimal   float64 `json:"base_krw_decimal,omitempty"`
	CloseKrwDecimal  float64 `json:"close_krw_decimal,omitempty"`
}

// CompanyOverview is the payload from /api/v2/stock-infos/{code}/overview.
// Surfaces the top-card of the 종목정보 deep tab: CEO, EV, market value,
// company description, industry classification, listing details.
type CompanyOverview struct {
	ProductCode        string         `json:"product_code"`
	Type               string         `json:"type,omitempty"`
	MarketCode         string         `json:"market_code,omitempty"`
	Market             string         `json:"market,omitempty"`
	ListDate           string         `json:"list_date,omitempty"`
	MarketValue        float64        `json:"market_value,omitempty"`
	MarketValueKrw     float64        `json:"market_value_krw,omitempty"`
	EnterpriseValue    float64        `json:"enterprise_value,omitempty"`
	EnterpriseValueKrw float64        `json:"enterprise_value_krw,omitempty"`
	DataSource         string         `json:"data_source,omitempty"`
	Company            CompanyProfile `json:"company"`
	FetchedAt          time.Time      `json:"fetched_at"`
}

type CompanyProfile struct {
	Code              string  `json:"code,omitempty"`
	Name              string  `json:"name,omitempty"`
	EnglishName       string  `json:"english_name,omitempty"`
	FullEnglishName   string  `json:"full_english_name,omitempty"`
	IndustryCode      string  `json:"industry_code,omitempty"`
	IndustryName      string  `json:"industry_name,omitempty"`
	Description       string  `json:"description,omitempty"`
	EstablishYear     int     `json:"establish_year,omitempty"`
	ListDate          string  `json:"list_date,omitempty"`
	CEO               string  `json:"ceo,omitempty"`
	HomepageURL       string  `json:"homepage_url,omitempty"`
	LogoImageURL      string  `json:"logo_image_url,omitempty"`
	SharesOutstanding int64   `json:"shares_outstanding,omitempty"`
	MarketValue       float64 `json:"market_value,omitempty"`
	MarketValueKrw    float64 `json:"market_value_krw,omitempty"`
	Currency          string  `json:"currency,omitempty"`
}

// StockIndicators is the aggregated valuation/earnings/dividend/stability
// snapshot returned by /api/v1/stock-detail/ui/wts/{code}/investment-indicators.
// Each section is the raw map of keys returned by Toss for that block; callers
// route by section name.
type StockIndicators struct {
	ProductCode string                     `json:"product_code"`
	Sections    map[string]IndicatorFields `json:"sections"` // key: 가치평가|수익|배당|안정성
	FetchedAt   time.Time                  `json:"fetched_at"`
}

// IndicatorFields holds the raw per-section payload as decoded JSON.
type IndicatorFields map[string]any

// StockValuation aggregates the per-stock valuation snapshot (evaluation) and
// peer comparison matrix (evaluation-comparison).
type StockValuation struct {
	ProductCode string          `json:"product_code"`
	PER         float64         `json:"per"`
	PBR         float64         `json:"pbr"`
	PSR         float64         `json:"psr"`
	Median      float64         `json:"median"`        // industry median for SelectedFactor
	Position    string          `json:"position"`      // HIGH|LOW|NORMAL
	Factor      string          `json:"factor"`        // PER (default), PBR, PSR, …
	Industry    string          `json:"industry"`      // selectedTics displayName
	Peers       []PeerValuation `json:"peers"`
	FetchedAt   time.Time       `json:"fetched_at"`
}

// PeerValuation is one row in the peer comparison table. `Value` is the most
// recent graph point for the selected factor.
type PeerValuation struct {
	ProductCode string  `json:"product_code"`
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Period      string  `json:"period"`
	IsSelf      bool    `json:"is_self"`
}

// SalesComposition is the revenue breakdown by business segment returned by
// /api/v1/companies/{companyCode}/sales-compositions.
type SalesComposition struct {
	ProductCode string                 `json:"product_code"`
	CompanyCode string                 `json:"company_code"`
	FiscalYear  int                    `json:"fiscal_year"`
	EndDate     string                 `json:"end_date"`
	Items       []SalesCompositionItem `json:"items"`
	DataSource  string                 `json:"data_source"`
	FetchedAt   time.Time              `json:"fetched_at"`
}

type SalesCompositionItem struct {
	Business string  `json:"business"`
	Product  string  `json:"product,omitempty"`
	Ratio    float64 `json:"ratio"`
}

type TICSIndustry struct {
	ProductCode string      `json:"product_code"`
	CompanyCode string      `json:"company_code"`
	BaseDate    string      `json:"base_date"`
	Major       []TICSEntry `json:"major"`
	Minor       []TICSEntry `json:"minor"`
	FetchedAt   time.Time   `json:"fetched_at"`
}

type TICSEntry struct {
	ID             int           `json:"id"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	CompanyCount   int           `json:"company_count"`
	Representative bool          `json:"representative"`
	Rankings       []TICSRanking `json:"rankings"`
}

type TICSRanking struct {
	BaseDate     string  `json:"base_date"`
	FiscalPeriod string  `json:"fiscal_period"`
	TypeName     string  `json:"type_name"`     // 시가총액|매출|영업이익률
	Ranking      int     `json:"ranking"`
	CompanyCount int     `json:"company_count"`
	DisplayValue string  `json:"display_value"`
	Value        float64 `json:"value"`
}

type AnalystSnapshot struct {
	ProductCode string          `json:"product_code"`
	Opinion     AnalystOpinion  `json:"opinion"`
	Consensus   ConsensusTarget `json:"consensus"`
	Reports     []AnalystReport `json:"reports"`
	FetchedAt   time.Time       `json:"fetched_at"`
}

type AnalystOpinion struct {
	Type        string  `json:"type"`        // BUY|HOLD|SELL
	StrongBuy   int     `json:"strong_buy"`
	Buy         int     `json:"buy"`
	Hold        int     `json:"hold"`
	Sell        int     `json:"sell"`
	StrongSell  int     `json:"strong_sell"`
	TargetUSD   float64 `json:"target_usd"`
	TargetKRW   float64 `json:"target_krw"`
	Description string  `json:"description"`
}

type ConsensusTarget struct {
	Mean       float64              `json:"mean"`
	High       float64              `json:"high"`
	Low        float64              `json:"low"`
	MeanKRW    float64              `json:"mean_krw"`
	HighKRW    float64              `json:"high_krw"`
	LowKRW     float64              `json:"low_krw"`
	Currency   string               `json:"currency"`
	PointDate  string               `json:"point_date"`
	PastCloses []ConsensusPastClose `json:"past_closes"`
}

type ConsensusPastClose struct {
	Date     string  `json:"date"`
	Price    float64 `json:"price"`
	PriceKRW float64 `json:"price_krw"`
}

type AnalystReport struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Date   string `json:"date"`
	URL    string `json:"url,omitempty"`
}

type StockFinancials struct {
	ProductCode     string                `json:"product_code"`
	Stability       StabilityRatios       `json:"stability"`
	Revenue         RevenueSeries         `json:"revenue"`
	OperatingIncome OperatingIncomeSeries `json:"operating_income"`
	FetchedAt       time.Time             `json:"fetched_at"`
}

type StabilityRatios struct {
	LiabilityRatio        float64 `json:"liability_ratio"`
	CurrentRatio          float64 `json:"current_ratio"`
	InterestCoverageRatio float64 `json:"interest_coverage_ratio"`
	IndustryMedian        float64 `json:"industry_median"`
	Position              string  `json:"position"` // HIGH|LOW|NORMAL
}

type RevenueSeries struct {
	CompanyName         string         `json:"company_name"`
	RecentFiscalYear    int            `json:"recent_fiscal_year"`
	RecentFiscalQuarter int            `json:"recent_fiscal_quarter"`
	RecentNetProfit     float64        `json:"recent_net_profit"`
	RecentNetProfitKrw  float64        `json:"recent_net_profit_krw"`
	FluctuationRate     float64        `json:"fluctuation_rate"`
	Position            string         `json:"position"`
	Graph               []RevenuePoint `json:"graph"`
}

type RevenuePoint struct {
	Period         string  `json:"period"`
	Revenue        float64 `json:"revenue"`
	RevenueKrw     float64 `json:"revenue_krw"`
	NetProfit      float64 `json:"net_profit"`
	NetProfitKrw   float64 `json:"net_profit_krw"`
	NetProfitRatio float64 `json:"net_profit_ratio"`
}

type OperatingIncomeSeries struct {
	CompanyName              string                 `json:"company_name"`
	RecentFiscalYear         int                    `json:"recent_fiscal_year"`
	RecentFiscalQuarter      int                    `json:"recent_fiscal_quarter"`
	RecentOperatingIncome    float64                `json:"recent_operating_income"`
	RecentOperatingIncomeKrw float64                `json:"recent_operating_income_krw"`
	FluctuationRate          float64                `json:"fluctuation_rate"`
	Position                 string                 `json:"position"`
	Graph                    []OperatingIncomePoint `json:"graph"`
}

type OperatingIncomePoint struct {
	Period               string  `json:"period"`
	OperatingIncome      float64 `json:"operating_income"`
	OperatingIncomeKrw   float64 `json:"operating_income_krw"`
	OperatingIncomeRatio float64 `json:"operating_income_ratio"`
}

type StockDividends struct {
	ProductCode string               `json:"product_code"`
	Summary     DividendYieldCard    `json:"summary"`
	RecentYears DividendYearsPayouts `json:"recent_years"`
	FullHistory []DividendPayout     `json:"full_history"`
	FetchedAt   time.Time            `json:"fetched_at"`
}

// DividendYieldCard is the TTM summary card from
// /api/v1/stock-infos/{code}/dividends/yield-ratio/histories.
type DividendYieldCard struct {
	DividendCount         int      `json:"dividend_count"`
	DividendMonths        []int    `json:"dividend_months"`
	DividendCash          float64  `json:"dividend_cash"`
	DividendCashKrw       *float64 `json:"dividend_cash_krw"`
	DividendYieldRatio    float64  `json:"dividend_yield_ratio"`
	TTMDividendYieldRatio float64  `json:"ttm_dividend_yield_ratio"`
	TTMDividendMonths     []string `json:"ttm_dividend_months"`
	TTMDps                float64  `json:"ttm_dps"`
	TTMDpsKrw             *float64 `json:"ttm_dps_krw"`
	TTMDividendTotalCount int      `json:"ttm_dividend_total_count"`
	DividendGrowthRatio   *float64 `json:"dividend_growth_ratio"`
	Currency              string   `json:"currency"`
}

// DividendYearsPayouts is the recent-range payout list from
// /api/v1/stock-infos/dividend/{code}/years.
type DividendYearsPayouts struct {
	StartDate    string          `json:"start_date"`
	RangeLabel   string          `json:"range_label"`
	Payouts      []DividendPayout `json:"payouts"`
	TotalCash    float64         `json:"total_cash"`
	TotalCashKrw float64         `json:"total_cash_krw"`
}

// DividendPayout is one historical payout record (used by both `years` and
// `summary` endpoints).
type DividendPayout struct {
	ExDate        string  `json:"ex_date"`
	PaymentDate   string  `json:"payment_date"`
	Currency      string  `json:"currency"`
	Cash          float64 `json:"cash"`
	CashKrw       float64 `json:"cash_krw"`
	YieldRatio    float64 `json:"yield_ratio"`
	TTMYieldRatio float64 `json:"ttm_yield_ratio"`
}

type StockEstimates struct {
	ProductCode     string                        `json:"product_code"`
	Headline        EstimateHeadline              `json:"headline"`
	Revenue         EstimateRevenueSeries         `json:"revenue"`
	EPS             EstimateEpsSeries             `json:"eps"`
	OperatingIncome EstimateOperatingIncomeSeries `json:"operating_income"`
	FetchedAt       time.Time                     `json:"fetched_at"`
}

// EstimateHeadline is the consensus-summary card from GET .../financial/estimate/date.
type EstimateHeadline struct {
	AnnounceAt            *string  `json:"announce_at"`
	RevenueEst            *float64 `json:"revenue_est"`
	RevenueEstKrw         *float64 `json:"revenue_est_krw"`
	EPSEst                *float64 `json:"eps_est"`
	EPSEstKrw             *float64 `json:"eps_est_krw"`
	OperatingIncomeEst    *float64 `json:"operating_income_est"`
	OperatingIncomeEstKrw *float64 `json:"operating_income_est_krw"`
}

type EstimateRevenueSeries struct {
	RevenueEst      *float64               `json:"revenue_est"`
	RevenueEstKrw   *float64               `json:"revenue_est_krw"`
	FluctuationRate float64                `json:"fluctuation_rate"`
	Fluctuation     float64                `json:"fluctuation"`
	FluctuationKrw  float64                `json:"fluctuation_krw"`
	Position        *string                `json:"position"`
	Graph           []EstimateRevenuePoint `json:"graph"`
}

type EstimateRevenuePoint struct {
	Period        string   `json:"period"`
	Revenue       *float64 `json:"revenue"`
	RevenueEst    *float64 `json:"revenue_est"`
	RevenueKrw    *float64 `json:"revenue_krw"`
	RevenueEstKrw *float64 `json:"revenue_est_krw"`
	Surprise      *float64 `json:"surprise"`
}

type EstimateEpsSeries struct {
	EPSEst          *float64           `json:"eps_est"`
	EPSEstKrw       *float64           `json:"eps_est_krw"`
	FluctuationRate float64            `json:"fluctuation_rate"`
	Fluctuation     float64            `json:"fluctuation"`
	FluctuationKrw  float64            `json:"fluctuation_krw"`
	Position        *string            `json:"position"`
	Graph           []EstimateEpsPoint `json:"graph"`
}

type EstimateEpsPoint struct {
	Period    string   `json:"period"`
	EPS       *float64 `json:"eps"`
	EPSEst    *float64 `json:"eps_est"`
	EPSKrw    *float64 `json:"eps_krw"`
	EPSEstKrw *float64 `json:"eps_est_krw"`
	Surprise  *float64 `json:"surprise"`
}

type EstimateOperatingIncomeSeries struct {
	OperatingIncomeEst    *float64                       `json:"operating_income_est"`
	OperatingIncomeEstKrw *float64                       `json:"operating_income_est_krw"`
	FluctuationRate       float64                        `json:"fluctuation_rate"`
	Fluctuation           float64                        `json:"fluctuation"`
	FluctuationKrw        float64                        `json:"fluctuation_krw"`
	Position              *string                        `json:"position"`
	Graph                 []EstimateOperatingIncomePoint `json:"graph"`
}

type EstimateOperatingIncomePoint struct {
	Period                string   `json:"period"`
	OperatingIncome       *float64 `json:"operating_income"`
	OperatingIncomeEst    *float64 `json:"operating_income_est"`
	OperatingIncomeKrw    *float64 `json:"operating_income_krw"`
	OperatingIncomeEstKrw *float64 `json:"operating_income_est_krw"`
	Surprise              *float64 `json:"surprise"`
}

// StockStatements is the pivoted view of a financial-statement-records call.
// Periods are ordered oldest-first; line items follow the parent/child order
// returned by the API.
type StockStatements struct {
	ProductCode string            `json:"product_code"`
	Factor      StatementFactor   `json:"factor"`     // BAL|INC|CAS
	Period      string            `json:"period"`     // Q|Y
	IsKr        bool              `json:"is_kr"`
	Periods     []StatementPeriod `json:"periods"`
	FetchedAt   time.Time         `json:"fetched_at"`
}

type StatementFactor struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type StatementPeriod struct {
	Period string              `json:"period"`
	Items  []StatementLineItem `json:"items"`
}

type StatementLineItem struct {
	Item       string   `json:"item"`                    // RTLR, SREV, etc
	ParentItem string   `json:"parent_item,omitempty"`
	NameKor    string   `json:"name_kor"`
	NameEng    string   `json:"name_eng,omitempty"`
	Unit       string   `json:"unit,omitempty"` // USD, KRW, etc
	Value      *float64 `json:"value"`          // null when not reported
}

type StockRatios struct {
	ProductCode string          `json:"product_code"`
	Factor      RatioFactor     `json:"factor"`      // DEBT_RATIO|CURRENT_RATIO|INTEREST_COVERAGE_RATIO
	Period      string          `json:"period"`      // Q|Y
	RangeLabel  string          `json:"range_label"` // 1년|3년|5년|전체
	Items       []RatioLineItem `json:"items"`
	FetchedAt   time.Time       `json:"fetched_at"`
}

type RatioFactor struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}

type RatioLineItem struct {
	Code   string       `json:"code"`  // e.g. TOTAL_SHAREHOLDERS_EQUITY
	Unit   string       `json:"unit"`  // AMOUNT|PERCENT
	Name   string       `json:"name"`  // 총자본
	Values []RatioValue `json:"values"`
}

type RatioValue struct {
	Period   string  `json:"period"`
	Value    float64 `json:"value"`
	ValueKrw float64 `json:"value_krw,omitempty"`
}

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
