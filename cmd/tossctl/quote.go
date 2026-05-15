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
	"github.com/junghoonkye/tossinvest-cli/internal/domain"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
)

func newQuoteCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote",
		Short: "Read quote data",
	}

	var (
		follow   bool
		interval time.Duration
	)
	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch the latest quote for a symbol; --follow for NDJSON stream",
		Long: `Show the latest quote (price, OHLC, 52w/1y range, marketCap, trading
strength, ETF expense ratio, dividend yield, ranking).

Without --follow this prints a snapshot.

With --follow this becomes a long-running stream: every --interval (default 3s)
the quote is re-fetched and a snapshot is emitted as NDJSON whenever last price
or volume changes. Use Ctrl-C to stop.

Toss exposes no WebSocket/SSE for prices; --follow is REST polling. During
closed market hours last and volume do not change so no lines are emitted;
this is intentional.

Examples:
  tossctl quote get SOXL
  tossctl quote get SOXL --follow --interval 3s
  tossctl quote get A005930 --follow`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if !follow {
				quote, err := app.client.GetQuote(cmd.Context(), args[0])
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteQuote(cmd.OutOrStdout(), app.format, quote)
			}
			return runQuoteFollow(cmd.Context(), app, args[0], interval, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	getCmd.Flags().BoolVar(&follow, "follow", false, "Stream quote updates as NDJSON until Ctrl-C")
	getCmd.Flags().DurationVar(&interval, "interval", 3*time.Second, "Poll interval when --follow is set")

	batchCmd := &cobra.Command{
		Use:   "batch <symbol> [symbol...]",
		Short: "Fetch quotes for multiple symbols at once",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			var quotes []domain.Quote
			for _, symbol := range args {
				quote, err := app.client.GetQuote(cmd.Context(), symbol)
				if err != nil {
					return err
				}
				quotes = append(quotes, quote)
			}

			return output.WriteQuotes(cmd.OutOrStdout(), app.format, quotes)
		},
	}

	cmd.AddCommand(getCmd, batchCmd)

	return cmd
}

func runQuoteFollow(ctx context.Context, app *appContext, symbol string, interval time.Duration, stdout, stderr io.Writer) error {
	stream, err := app.client.StreamQuote(client.StreamQuoteOptions{
		Symbol:   symbol,
		Interval: interval,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "quote poll error: %v\n", err)
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
		case q, ok := <-stream.Quotes():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteQuoteNDJSON(stdout, q); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
