package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newMyCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "my",
		Short: "Personal data (own fills, recent activity)",
	}

	var timeUnit string
	fillsCmd := &cobra.Command{
		Use:   "fills <symbol>",
		Short: "Show own executions bucketed by time (chart overlay)",
		Long: `List the executions of the current account for a given symbol,
bucketed by --tf granularity (default thirty_minute).

This wraps /api/v3/trading/orders/histories/compact/executed.

Examples:
  tossctl my fills SOXL                       # 30-minute buckets
  tossctl my fills SOXL --tf one_hour         # 1-hour buckets (if supported)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			productCode, err := app.client.ResolveProductCode(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			fills, err := app.client.ListCompactExecutions(cmd.Context(), productCode, timeUnit)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteFills(cmd.OutOrStdout(), app.format, fills)
		},
	}
	fillsCmd.Flags().StringVar(&timeUnit, "tf", "thirty_minute", "Bucket size: thirty_minute (default) or other Toss-supported timeUnit string")

	cmd.AddCommand(fillsCmd)
	return cmd
}
