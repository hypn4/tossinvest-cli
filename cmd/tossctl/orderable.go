package main

import (
	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newOrderableCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "orderable",
		Short: "Show orderable + withdrawable summary across KR and US markets",
		Long: `Show the combined orderable cash + withdrawable amounts per
settlement date for both KR and US markets.

The output bundles three Toss endpoints:
  - /api/v1/dashboard/common/cached-orderable-amount
  - /api/v3/my-assets/transactions/markets/kr/overview
  - /api/v3/my-assets/transactions/markets/us/overview

Use --output json to get a single structured payload for LLM/agent pipelines.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			summary, err := app.client.GetOrderableSummary(cmd.Context())
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOrderable(cmd.OutOrStdout(), app.format, summary)
		},
	}
}
