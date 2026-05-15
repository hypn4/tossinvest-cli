package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
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

	expiriesCmd := &cobra.Command{
		Use:   "expiries <underlying>",
		Short: "List the option expiry ladder for an underlying",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			exps, err := app.client.ListOptionExpiries(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOptionExpiries(cmd.OutOrStdout(), app.format, exps)
		},
	}

	var (
		chainExpiry     string
		chainType       string
		chainWithPrices bool
	)
	chainCmd := &cobra.Command{
		Use:   "chain <underlying>",
		Short: "Show the strike chain (call + put per strike) for an expiry",
		Long: `Show the strike chain for an underlying's expiry.

If --expiry is omitted, the nearest expiry from the expiry ladder is used.

Use --with-prices to join in bulk option prices (one extra round-trip; ~58
codes per call). --type call|put filters the rows client-side.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			expiry := chainExpiry
			if expiry == "" {
				exps, err := app.client.ListOptionExpiries(cmd.Context(), args[0])
				if err != nil {
					return userFacingCommandError(err)
				}
				if len(exps) == 0 {
					return fmt.Errorf("no expiries available for %s", args[0])
				}
				expiry = exps[0].MaturityDate
			}
			rows, err := app.client.GetOptionChain(cmd.Context(), args[0], expiry)
			if err != nil {
				return userFacingCommandError(err)
			}
			if chainWithPrices {
				codes := make([]string, 0, len(rows)*2)
				for _, r := range rows {
					if r.CallGuid != "" {
						codes = append(codes, r.CallGuid)
					}
					if r.PutGuid != "" {
						codes = append(codes, r.PutGuid)
					}
				}
				prices, err := app.client.GetOptionPrices(cmd.Context(), codes)
				if err != nil {
					return userFacingCommandError(err)
				}
				priceByCode := make(map[string]*domain.OptionPrice, len(prices))
				for i := range prices {
					priceByCode[prices[i].Code] = &prices[i]
				}
				for i := range rows {
					if p, ok := priceByCode[rows[i].CallGuid]; ok {
						rows[i].CallPrice = p
					}
					if p, ok := priceByCode[rows[i].PutGuid]; ok {
						rows[i].PutPrice = p
					}
				}
			}
			switch strings.ToLower(chainType) {
			case "", "both":
			case "call":
				for i := range rows {
					rows[i].PutGuid = ""
					rows[i].PutPrice = nil
					rows[i].PutOpenInterest = 0
				}
			case "put":
				for i := range rows {
					rows[i].CallGuid = ""
					rows[i].CallPrice = nil
					rows[i].CallOpenInterest = 0
				}
			default:
				return fmt.Errorf("--type must be call/put/both (got %q)", chainType)
			}
			return output.WriteOptionChain(cmd.OutOrStdout(), app.format, rows)
		},
	}
	chainCmd.Flags().StringVar(&chainExpiry, "expiry", "", "Expiry date YYYY-MM-DD (default: nearest)")
	chainCmd.Flags().StringVar(&chainType, "type", "both", "Filter: call / put / both")
	chainCmd.Flags().BoolVar(&chainWithPrices, "with-prices", false, "Join bulk prices into the chain rows (extra round-trip)")

	pricesCmd := &cobra.Command{
		Use:   "prices <code> [<code>...]",
		Short: "Bulk option/stock prices (lighter than quote get)",
		Long: `Fetch a flat price list for one or many productCodes (typically OPT_ codes).

Example:
  tossctl options prices OPT_SNDK260515C01395000_20260506 OPT_SNDK260515P01395000_20260506`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			prices, err := app.client.GetOptionPrices(cmd.Context(), args)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOptionPrices(cmd.OutOrStdout(), app.format, prices)
		},
	}

	cmd.AddCommand(infoCmd, atmCmd, expiriesCmd, chainCmd, pricesCmd)
	return cmd
}
