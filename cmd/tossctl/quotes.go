package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	var (
		ticksCount    int
		ticksFollow   bool
		ticksInterval time.Duration
		ticksSince    float64
	)
	ticksCmd := &cobra.Command{
		Use:   "ticks <symbol>",
		Short: "Show recent trade ticks (체결 틱); --follow for NDJSON stream",
		Long: `Show recent trade ticks for a symbol.

Without --follow this prints a snapshot of the last --count ticks (newest first).

With --follow this becomes a long-running stream: ticks are emitted as
newline-delimited JSON (NDJSON) in chronological order, deduped by
cumulativeVolume. Use Ctrl-C to stop.

Examples:
  tossctl quotes ticks SOXL --count 20
  tossctl quotes ticks SOXL --follow --interval 2s
  tossctl quotes ticks SOXL --follow --since 1731780`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if !ticksFollow {
				ticks, err := app.client.GetTicks(cmd.Context(), args[0], ticksCount)
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteTicks(cmd.OutOrStdout(), app.format, ticks)
			}
			return runTicksFollow(cmd.Context(), app, args[0], ticksCount, ticksInterval, ticksSince, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	ticksCmd.Flags().IntVar(&ticksCount, "count", 50, "Number of ticks per fetch (1..)")
	ticksCmd.Flags().BoolVar(&ticksFollow, "follow", false, "Stream new ticks as NDJSON until Ctrl-C")
	ticksCmd.Flags().DurationVar(&ticksInterval, "interval", 2*time.Second, "Poll interval when --follow is set")
	ticksCmd.Flags().Float64Var(&ticksSince, "since", 0, "Resume from this cumulativeVolume (exclusive)")

	cmd.AddCommand(bookCmd, ticksCmd)
	return cmd
}

func runTicksFollow(ctx context.Context, app *appContext, symbol string, count int, interval time.Duration, since float64, stdout, stderr io.Writer) error {
	stream, err := app.client.StreamTicks(client.StreamTicksOptions{
		Symbol:   symbol,
		Count:    count,
		Interval: interval,
		Since:    since,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "tick poll error: %v\n", err)
		},
	})
	if err != nil {
		return userFacingCommandError(err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Run(sigCtx)
	}()

	for {
		select {
		case tick, ok := <-stream.Ticks():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteTickNDJSON(stdout, tick); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
