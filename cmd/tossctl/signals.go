package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newSignalsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signals",
		Short: "Read Toss AI signals (reasoning, news, events)",
	}

	listCmd := &cobra.Command{
		Use:   "list <symbol> [<symbol>...]",
		Short: "One-line Korean reasoning per symbol",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			codes := make([]string, 0, len(args))
			for _, arg := range args {
				code, err := app.client.ResolveProductCode(cmd.Context(), arg)
				if err != nil {
					return userFacingCommandError(fmt.Errorf("resolving %s: %w", arg, err))
				}
				codes = append(codes, code)
			}
			signals, err := app.client.ListSignals(cmd.Context(), codes)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteSignals(cmd.OutOrStdout(), app.format, signals)
		},
	}

	detailCmd := &cobra.Command{
		Use:   "detail <symbol>",
		Short: "Full reasoning + news + related stocks for a single symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			code, err := app.client.ResolveProductCode(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			detail, err := app.client.GetSignalDetail(cmd.Context(), code)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteSignalDetail(cmd.OutOrStdout(), app.format, detail)
		},
	}

	eventsCmd := &cobra.Command{
		Use:   "events <symbol> [<symbol>...]",
		Short: "Scheduled event signals (earnings, disclosures, …)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			codes := make([]string, 0, len(args))
			for _, arg := range args {
				code, err := app.client.ResolveProductCode(cmd.Context(), arg)
				if err != nil {
					return userFacingCommandError(err)
				}
				codes = append(codes, code)
			}
			events, err := app.client.ListEventSignals(cmd.Context(), codes)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteEventSignals(cmd.OutOrStdout(), app.format, events)
		},
	}

	cmd.AddCommand(listCmd, detailCmd, eventsCmd)
	return cmd
}
