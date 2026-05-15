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
	AveragePrice    float64 `json:"average_price,omitempty"`
	CurrentPrice    float64 `json:"current_price,omitempty"`
	MarketValue     float64 `json:"market_value,omitempty"`
	UnrealizedPnL   float64 `json:"unrealized_pnl,omitempty"`
	ProfitRate      float64 `json:"profit_rate,omitempty"`
	DailyProfitLoss float64 `json:"daily_profit_loss,omitempty"`
	DailyProfitRate float64 `json:"daily_profit_rate,omitempty"`

	AveragePriceUSD    float64 `json:"average_price_usd,omitempty"`
	CurrentPriceUSD    float64 `json:"current_price_usd,omitempty"`
	MarketValueUSD     float64 `json:"market_value_usd,omitempty"`
	UnrealizedPnLUSD   float64 `json:"unrealized_pnl_usd,omitempty"`
	ProfitRateUSD      float64 `json:"profit_rate_usd,omitempty"`
	DailyProfitLossUSD float64 `json:"daily_profit_loss_usd,omitempty"`
	DailyProfitRateUSD float64 `json:"daily_profit_rate_usd,omitempty"`
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
