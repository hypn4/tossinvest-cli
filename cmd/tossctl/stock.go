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

	cmd.AddCommand(infoCmd, overviewCmd, indicatorsCmd)
	return cmd
}
