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

func newChartCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chart",
		Short: "Read OHLCV chart candles",
	}

	var (
		timeframe  string
		count      int
		session    string
		investMode string
		from       string
		noAdjust   bool
		follow     bool
		interval   time.Duration
	)

	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch chart candles for a symbol; --follow for NDJSON stream",
		Long: `Fetch OHLCV candles from Toss Securities.

Supported --tf values: 1m, 3m, 5m, 10m, 15m, 30m, 1h (= 60m), 1d, 1w, 1mo, 3mo, 1y.
Supported --session values: all (default), main, day, pre, after.

Without --follow this prints the requested candles once.

With --follow this becomes a long-running stream: every --interval (default 60s)
the latest two buckets are polled, and any new or updated candle is emitted as
NDJSON. Dedup key is (datetime, close, volume) — intra-bucket updates and new
buckets both emit. Use Ctrl-C to stop.

Toss exposes no WebSocket/SSE for prices; --follow is REST polling. During
closed market hours the dedup keys stop changing so no lines are emitted; this
is intentional and not a broken stream.

Examples:
  tossctl chart get SOXL --tf 1m --count 60
  tossctl chart get SOXL --tf 1m --follow --interval 60s
  tossctl chart get A005930 --tf 5m --follow`,
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

			if !follow {
				chart, err := app.client.GetChart(cmd.Context(), args[0], chartOpts)
				if err != nil {
					return userFacingCommandError(err)
				}
				return output.WriteChart(cmd.OutOrStdout(), app.format, chart)
			}
			return runChartFollow(cmd.Context(), app, args[0], timeframe, count, session, interval, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	getCmd.Flags().StringVar(&timeframe, "tf", "1d", "Timeframe alias: 1m/3m/5m/10m/15m/30m/1h/1d/1w/1mo/3mo/1y")
	getCmd.Flags().IntVar(&count, "count", 100, "Number of candles (1..); when --follow is set, defaults to 2 if unspecified or >5")
	getCmd.Flags().StringVar(&session, "session", "", "Session filter: all/main/day/pre/after (default: server default)")
	getCmd.Flags().StringVar(&investMode, "invest-mode", "", "Invest mode: integrated/regular (default: server default)")
	getCmd.Flags().StringVar(&from, "from", "", "Pagination cursor — RFC3339 timestamp; older-than this point")
	getCmd.Flags().BoolVar(&noAdjust, "no-adjust", false, "Disable split/dividend adjustment (default: enabled)")
	getCmd.Flags().BoolVar(&follow, "follow", false, "Stream candle updates as NDJSON until Ctrl-C")
	getCmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "Poll interval when --follow is set")

	cmd.AddCommand(getCmd)
	return cmd
}

func runChartFollow(ctx context.Context, app *appContext, symbol, timeframe string, count int, session string, interval time.Duration, stdout, stderr io.Writer) error {
	pollCount := count
	if pollCount <= 0 || pollCount > 5 {
		// Tight default — the stream's purpose is "what just changed", not
		// "give me 100 buckets every minute". Cap at 5 even if user passed more.
		pollCount = 2
	}
	stream, err := app.client.StreamChart(client.StreamChartOptions{
		Symbol:    symbol,
		Timeframe: timeframe,
		Count:     pollCount,
		Interval:  interval,
		Session:   session,
		OnError: func(err error) {
			fmt.Fprintf(stderr, "chart poll error: %v\n", err)
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
		case candle, ok := <-stream.Ticks():
			if !ok {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return userFacingCommandError(err)
			}
			if err := output.WriteCandleNDJSON(stdout, candle); err != nil {
				stream.Close()
				return err
			}
		}
	}
}
