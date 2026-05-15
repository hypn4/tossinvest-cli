package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newChartCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chart",
		Short: "Read OHLCV chart candles",
	}

	var (
		timeframe   string
		count       int
		session     string
		investMode  string
		from        string
		noAdjust    bool
	)

	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch chart candles for a symbol",
		Long: `Fetch OHLCV candles from Toss Securities.

Supported --tf values: 1m, 3m, 5m, 10m, 15m, 30m, 1h (= 60m), 1d, 1w, 1mo, 3mo, 1y.
Supported --session values: all (default), main, day, pre, after.

For continuous polling use shell composition:

  watch -n 3 tossctl chart get SOXL --tf 30m --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			chartOpts := client.ChartOptions{
				Timeframe:  timeframe,
				Count:      count,
				Session:    session,
				InvestMode: investMode,
			}
			if noAdjust {
				adjusted := false
				chartOpts.UseAdjusted = &adjusted
			}
			if from != "" {
				parsed, err := time.Parse(time.RFC3339, from)
				if err != nil {
					return fmt.Errorf("--from must be an RFC3339 timestamp with timezone: %w", err)
				}
				chartOpts.From = parsed
			}

			chart, err := app.client.GetChart(cmd.Context(), args[0], chartOpts)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteChart(cmd.OutOrStdout(), app.format, chart)
		},
	}
	getCmd.Flags().StringVar(&timeframe, "tf", "1d", "Timeframe alias: 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y")
	getCmd.Flags().IntVar(&count, "count", 100, "Number of candles (1..)")
	getCmd.Flags().StringVar(&session, "session", "", "Session filter: all/main/day/pre/after (default: server default)")
	getCmd.Flags().StringVar(&investMode, "invest-mode", "", "Invest mode: integrated/regular (default: server default)")
	getCmd.Flags().StringVar(&from, "from", "", "Pagination cursor — RFC3339 timestamp; older-than this point")
	getCmd.Flags().BoolVar(&noAdjust, "no-adjust", false, "Disable split/dividend adjustment (default: enabled)")

	cmd.AddCommand(getCmd)
	return cmd
}
