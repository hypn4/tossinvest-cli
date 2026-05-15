package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newStockCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stock",
		Short: "Stock info (deep tab)",
	}

	infoCmd := &cobra.Command{
		Use:   "info <symbol>",
		Short: "Fetch the 종목정보 deep-tab payload (OVERVIEW / FINANCES / ANALYST / 매출 구성 / 동종업계 / ...)",
		Long: `Fetch the deep stock-info payload behind the Toss 종목정보 tab.

Returns ~13 sections (OVERVIEW, INDICATORS, NEWS, ANNOUNCEMENT, FINANCES,
EARNINGS_AND_CONSENSUS, ANALYST_OPINION, COMPOSITION_OF_REVENUE,
VALUATION_METRICS, STABILITY, TOP_TIER_TREND, STOCK_INFO_SIGNAL, PRICE).

Table mode prints a one-line summary per section. Use --output json for the
full structured payload (consumer-friendly for LLM pipelines).

Examples:
  tossctl stock info SNDK
  tossctl stock info NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			detail, err := app.client.GetStockInfoDetail(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteStockInfoDetail(cmd.OutOrStdout(), app.format, detail)
		},
	}

	overviewCmd := &cobra.Command{
		Use:   "overview <symbol>",
		Short: "Show the company overview card (CEO, EV, industry, description, listing)",
		Long: `Fetch the company overview from /api/v2/stock-infos/{code}/overview.

Returns the top-card of the 종목정보 deep tab: company name (KR + EN),
CEO, industry, description, establish year, list date, shares outstanding,
market value (USD + KRW), enterprise value (USD + KRW), homepage URL,
data source attribution.

Examples:
  tossctl stock overview SNDK
  tossctl stock overview NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			ov, err := app.client.GetCompanyOverview(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteCompanyOverview(cmd.OutOrStdout(), app.format, ov)
		},
	}

	indicatorsCmd := &cobra.Command{
		Use:   "indicators <symbol>",
		Short: "Show investment indicators (가치평가/수익/배당/안정성)",
		Long: `Fetch the investment indicators payload from
/api/v1/stock-detail/ui/wts/{code}/investment-indicators.

Returns four sectioned blocks: 가치평가 (PER/PBR/PSR), 수익 (EPS/BPS/ROE),
배당 (frequency, yield, annual cash), 안정성 (debt/current ratios).

Examples:
  tossctl stock indicators SNDK
  tossctl stock indicators NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			ind, err := app.client.GetStockIndicators(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteStockIndicators(cmd.OutOrStdout(), app.format, ind)
		},
	}

	valuationCmd := &cobra.Command{
		Use:   "valuation <symbol>",
		Short: "Per-stock PER/PBR/PSR vs industry median + peer table",
		Long: `Fetch the valuation snapshot and peer comparison from
/api/v2/stock-infos/evaluation/{code} and
/api/v2/stock-infos/evaluation-comparison/{code} (both POST {}).

Output shows PER/PBR/PSR plus the industry median and HIGH/LOW/NORMAL
position label, followed by the peer table (5 stocks) for the
selected factor.

Examples:
  tossctl stock valuation SNDK
  tossctl stock valuation NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			val, err := app.client.GetStockValuation(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteStockValuation(cmd.OutOrStdout(), app.format, val)
		},
	}

	revenueCmd := &cobra.Command{
		Use:   "revenue <symbol>",
		Short: "Revenue composition by business segment (latest fiscal period)",
		Long: `Fetch revenue composition from
/api/v1/companies/{companyCode}/sales-compositions.

The companyCode (e.g. NAS116LTR-E0) is resolved internally via the
company-overview call.

Examples:
  tossctl stock revenue SNDK
  tossctl stock revenue NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			sc, err := app.client.GetSalesComposition(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteSalesComposition(cmd.OutOrStdout(), app.format, sc)
		},
	}

	peersCmd := &cobra.Command{
		Use:   "peers <symbol>",
		Short: "TICS industry classification + peer rankings within industry",
		Long: `Fetch the TICS industry taxonomy and per-metric peer rankings
from /api/v2/companies/{companyCode}/tics.

Each industry block lists the company's rank within that industry for
시가총액 / 매출 / 영업이익률 (most recent fiscal period).

Table mode shows only major-industry entries; use --output json to also
see minorList.

Examples:
  tossctl stock peers SNDK
  tossctl stock peers NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			ind, err := app.client.GetTICSIndustry(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteTICSIndustry(cmd.OutOrStdout(), app.format, ind)
		},
	}

	analystCmd := &cobra.Command{
		Use:   "analyst <symbol>",
		Short: "Analyst BUY/HOLD/SELL counts + consensus target + reports",
		Long: `Fetch the analyst opinion + consensus target price + report list.

Stitches three endpoints:
  - /api/v1/stock-detail/ui/wts/{code}/analyst-opinion
  - /api/v2/stock-infos/consensus/{code}
  - /api/v1/stock-detail/ui/wts/{code}/analyst-reports

Examples:
  tossctl stock analyst SNDK
  tossctl stock analyst NAS0250224006 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			snap, err := app.client.GetAnalystSnapshot(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteAnalystSnapshot(cmd.OutOrStdout(), app.format, snap)
		},
	}

	cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd, valuationCmd, revenueCmd, peersCmd, analystCmd)
	return cmd
}
