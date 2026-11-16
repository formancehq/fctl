package demo

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type streamResult struct {
	scenario scenario
	request  operations.V2CreateTransactionRequest
	commit   func()
	err      error
}

func (c *controller) runStream(ctx context.Context, client ledgerClient, rate float64, concurrency int, progress io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	start := time.Now()
	defer func() {
		c.store.ElapsedSeconds = time.Since(start).Seconds()
		if c.store.ElapsedSeconds > 0 {
			c.store.Rate = float64(c.store.Transactions) / c.store.ElapsedSeconds
		}
		fmt.Fprintf(progress, "Stopped: ledger=%s transactions=%d errors=%d interrupted=%d rate=%.2f/s (data retained)\n", c.store.Ledger, c.store.Transactions, c.store.Errors, c.store.Interrupted, c.store.Rate)
	}()
	ready := make([]scenario, concurrency)
	for i := range ready {
		namespace := ""
		if concurrency > 1 {
			namespace = fmt.Sprintf("worker_%03d", i)
		}
		ready[i] = newRecipeScenario(c.recipeName, namespace)
	}

	results := make(chan streamResult, concurrency)
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	heartbeat := time.NewTicker(time.Second)
	defer heartbeat.Stop()
	// Allow at most one millisecond of tokens. This accommodates timer resolution
	// at high rates without replaying a backlog after a slow request or stall.
	capacity := max(1, rate/1000)
	tokens := float64(1)
	lastRefill := start
	lastProgress := start
	lastCount := uint64(0)
	lastTemplate := ""
	done := ctx.Done()
	inflight := 0
	stopping := false
	var firstErr error
	for {
		if ctx.Err() != nil {
			stopping = true
			cancel()
			done = nil
		}
		if stopping && inflight == 0 {
			if c.store.Interrupted > 0 {
				fmt.Fprintf(progress, "%d requests interrupted; their commit status is unknown. Inspect the retained ledger by demo.run=%s.\n", c.store.Interrupted, c.store.RunID)
			}
			return firstErr
		}
		var tick <-chan time.Time
		if !stopping {
			now := time.Now()
			tokens = min(capacity, tokens+now.Sub(lastRefill).Seconds()*rate)
			lastRefill = now
			for tokens >= 1 && len(ready) > 0 && ctx.Err() == nil {
				scenario := ready[0]
				ready = ready[1:]
				request, commit := scenario.next(c.store.Ledger, c.store.RunID)
				inflight++
				tokens--
				go func(result streamResult) {
					response, err := client.CreateTransaction(ctx, result.request, operations.WithOperationTimeout(30*time.Second))
					if err == nil && response.StatusCode != 200 {
						err = fmt.Errorf("unexpected HTTP %d", response.StatusCode)
					}
					result.err = err
					// The coordinator drains every submitted request, including after errors
					// and cancellation. Acknowledged writes remain included in the summary.
					results <- result
				}(streamResult{scenario: scenario, request: request, commit: commit})
			}
			if len(ready) > 0 {
				delay := max(time.Nanosecond, time.Duration(max(0, (1-tokens)/rate)*float64(time.Second)))
				timer.Reset(delay)
				tick = timer.C
			} else {
				timer.Stop()
			}
		} else {
			timer.Stop()
		}
		select {
		case result := <-results:
			inflight--
			if result.err != nil {
				if ctx.Err() != nil {
					c.store.Interrupted++
				} else {
					c.store.Errors++
					firstErr = fmt.Errorf("demo ledger %s stopped at %s (data retained; commit status may be unknown): %w", c.store.Ledger, *result.request.IdempotencyKey, result.err)
					stopping = true
					cancel()
					done = nil
				}
			} else {
				result.commit()
				c.store.Transactions++
				lastTemplate = result.request.V2PostTransaction.V2Metadata["demo.template"]
				ready = append(ready, result.scenario)
			}
		case <-tick:
		case <-heartbeat.C:
			now := time.Now()
			observed := float64(c.store.Transactions-lastCount) / now.Sub(lastProgress).Seconds()
			fmt.Fprintf(progress, "ledger=%s transactions=%d errors=%d rate=%.2f/s target=%g/s concurrency=%d inflight=%d template=%s\n", c.store.Ledger, c.store.Transactions, c.store.Errors, observed, rate, concurrency, inflight, lastTemplate)
			lastProgress = now
			lastCount = c.store.Transactions
		case <-done:
			stopping = true
			cancel()
			done = nil
		}
	}
}
