package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newQuotesCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quotes",
		Short: "Read orderbook and tick data",
	}

	bookCmd := &cobra.Command{
		Use:   "book <symbol>",
		Short: "Show the latest orderbook (호가창)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			book, err := app.client.GetOrderBook(cmd.Context(), args[0])
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteOrderBook(cmd.OutOrStdout(), app.format, book)
		},
	}

	cmd.AddCommand(bookCmd)
	return cmd
}

// Suppress unused imports until Task 12 lands.
var (
	_ = context.Background
	_ = errors.New
	_ = fmt.Sprintf
	_ = os.Stdin
	_ = signal.Notify
	_ = syscall.SIGINT
	_ = time.Second
)

var (
	_ = client.StreamTicksOptions{}
)
