package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newOptionsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "options",
		Short: "US option contracts (read-only)",
		Long: `Read-only access to US option contracts.

Toss treats each option as a first-class product with an OCC-style productCode:

  OPT_<rootSymbol><YYMMDD><C|P><strikeMillis8>_<listDateYYYYMMDD>

Example: SNDK 2026-05-15 $1,395 Call = OPT_SNDK260515C01395000_20260506.

Once you have an OPT_ productCode, the existing tossctl commands all work:
  tossctl quote   get   OPT_...
  tossctl quotes  book  OPT_...
  tossctl quotes  ticks OPT_...   [--follow]
  tossctl chart   get   OPT_... --tf 15m   (auto-routes to us-o chart family)

Use 'options nearest-atm <symbol>' to discover the OPT_ code for an
underlying's nearest-expiry at-the-money option. The full strike+expiry
chain enumeration endpoint has not yet been reverse-engineered.`,
	}

	infoCmd := &cobra.Command{
		Use:   "info <OPT_…>",
		Short: "Show option-specific metadata (strike, expiry, OI, bid/ask, halt status)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			inst, err := app.client.GetOptionInstrument(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOptionInstrument(cmd.OutOrStdout(), app.format, inst)
		},
	}

	atmCmd := &cobra.Command{
		Use:   "nearest-atm <underlying>",
		Short: "Return the OPT_ productCode for the underlying's nearest-expiry ATM option",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			code, err := app.client.GetNearestATMOption(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), code)
			return err
		},
	}

	cmd.AddCommand(infoCmd, atmCmd)
	return cmd
}
