package demo

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

// concurrentLedger records requests at the transport boundary. It independently
// checks that a scenario never has two outstanding writes and that independent
// scenarios cannot operate on the same account namespace.
type concurrentLedger struct {
	fakeLedger
	mu                 sync.Mutex
	active             int
	maxActive          int
	completed          int
	acknowledged       int
	starts             []time.Time
	references         map[string]bool
	lanes              map[string]bool
	accounts           map[string]string
	duplicateReference bool
	overlappingLane    bool
	sharedAccount      bool
	send               func(context.Context, operations.V2CreateTransactionRequest, int) error
}

func (f *concurrentLedger) CreateTransaction(ctx context.Context, request operations.V2CreateTransactionRequest, _ ...operations.Option) (*operations.V2CreateTransactionResponse, error) {
	reference := *request.IdempotencyKey
	lane := reference[:strings.LastIndex(reference, "-")]
	f.mu.Lock()
	if f.references == nil {
		f.references = make(map[string]bool)
		f.lanes = make(map[string]bool)
		f.accounts = make(map[string]string)
	}
	if f.references[reference] {
		f.duplicateReference = true
	}
	f.references[reference] = true
	if f.lanes[lane] {
		f.overlappingLane = true
	}
	f.lanes[lane] = true
	for _, name := range []string{"bank_id", "client_id", "platform_name"} {
		if value, ok := request.V2PostTransaction.Script.Vars[name]; ok {
			account := name + "=" + value
			if owner, exists := f.accounts[account]; exists && owner != lane {
				f.sharedAccount = true
			}
			f.accounts[account] = lane
		}
	}
	ordinal := len(f.starts)
	f.starts = append(f.starts, time.Now())
	f.active++
	f.maxActive = max(f.maxActive, f.active)
	f.mu.Unlock()
	err := f.send(ctx, request, ordinal)
	f.mu.Lock()
	f.active--
	f.completed++
	if err == nil {
		f.acknowledged++
	}
	delete(f.lanes, lane)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &operations.V2CreateTransactionResponse{StatusCode: 200}, nil
}

func TestParallelStreamReachesGlobalRateDespiteRequestLatency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, _ int) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return nil
			}
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
		require.NoError(t, c.runStream(ctx, client, 100, 8, &bytes.Buffer{}))
		client.mu.Lock()
		defer client.mu.Unlock()
		require.GreaterOrEqual(t, client.acknowledged, 180, "the old sequential runner could only acknowledge about 40 transactions")
		require.LessOrEqual(t, client.acknowledged, 200)
		require.LessOrEqual(t, len(client.starts), 201, "--rate is global, not multiplied by the number of scenarios")
		require.Greater(t, client.maxActive, 1)
		require.LessOrEqual(t, client.maxActive, 8)
		require.Zero(t, client.active)
		require.Equal(t, len(client.starts), client.completed)
		require.EqualValues(t, client.acknowledged, c.store.Transactions)
		require.Zero(t, c.store.Errors)
		require.False(t, client.duplicateReference)
		require.False(t, client.overlappingLane, "dependent transactions must wait for their previous acknowledgement")
		require.False(t, client.sharedAccount, "parallel scenarios must not share account balances")
		require.GreaterOrEqual(t, client.starts[100].Sub(client.starts[0]), 990*time.Millisecond)
	})
}

func TestParallelStreamFailureCancelsAndDrainsOutstandingWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failure := errors.New("connection lost after commit")
		client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, ordinal int) error {
			switch ordinal {
			case 0:
				time.Sleep(20 * time.Millisecond)
				return nil
			case 1:
				time.Sleep(50 * time.Millisecond)
				return failure
			default:
				<-ctx.Done()
				time.Sleep(20 * time.Millisecond)
				return ctx.Err()
			}
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
		var progress bytes.Buffer
		err := c.runStream(t.Context(), client, 100, 4, &progress)
		require.ErrorIs(t, err, failure)
		client.mu.Lock()
		defer client.mu.Unlock()
		require.Zero(t, client.active, "return only after all outstanding writes have finished")
		require.Equal(t, len(client.starts), client.completed)
		require.Equal(t, 1, client.acknowledged)
		require.EqualValues(t, client.acknowledged, c.store.Transactions)
		require.EqualValues(t, 1, c.store.Errors, "cancellation of sibling requests must not hide the initiating failure or inflate errors")
		require.Greater(t, client.maxActive, 1)
		require.LessOrEqual(t, client.maxActive, 4)
		require.False(t, client.overlappingLane)
		require.False(t, client.duplicateReference)
		require.Contains(t, progress.String(), "unknown")
	})
}

func TestParallelStreamCancellationWaitsForOutstandingRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, _ int) error {
			<-ctx.Done()
			// A transport may need time to unwind after cancellation. The runner must
			// wait instead of returning while requests still use the client or output.
			time.Sleep(20 * time.Millisecond)
			return ctx.Err()
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
		start := time.Now()
		require.NoError(t, c.runStream(ctx, client, 100, 4, &bytes.Buffer{}))
		require.GreaterOrEqual(t, time.Since(start), 70*time.Millisecond)
		client.mu.Lock()
		defer client.mu.Unlock()
		require.Zero(t, client.active)
		require.Equal(t, len(client.starts), client.completed)
		require.Equal(t, 4, client.maxActive)
		require.EqualValues(t, 4, c.store.Interrupted)
		require.Zero(t, c.store.Transactions)
		require.Zero(t, c.store.Errors)
		require.False(t, client.overlappingLane)
	})
}

func TestParallelStreamFractionalRatesDoNotStall(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rate     float64
		duration time.Duration
		want     int
	}{
		{name: "two and a half", rate: 2.5, duration: 2 * time.Second, want: 5},
		{name: "half", rate: 0.5, duration: 5 * time.Second, want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), tc.duration)
				defer cancel()
				client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, _ int) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Millisecond):
						return nil
					}
				}}
				c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
				require.NoError(t, c.runStream(ctx, client, tc.rate, 4, &bytes.Buffer{}))
				client.mu.Lock()
				defer client.mu.Unlock()
				require.GreaterOrEqual(t, len(client.starts), tc.want)
				require.LessOrEqual(t, len(client.starts), tc.want+1, "a submission may race the exact deadline, but no backlog may follow")
				require.Equal(t, tc.want, client.acknowledged)
				require.EqualValues(t, tc.want, c.store.Transactions)
				require.Zero(t, client.active)
				require.Zero(t, c.store.Errors)
			})
		})
	}
}

func TestParallelStreamTenThousandRateRemainsGloballyBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, _ int) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return nil
			}
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
		require.NoError(t, c.runStream(ctx, client, 10000, 512, &bytes.Buffer{}))
		client.mu.Lock()
		defer client.mu.Unlock()
		require.GreaterOrEqual(t, client.acknowledged, 400)
		require.LessOrEqual(t, client.acknowledged, 1000)
		require.LessOrEqual(t, len(client.starts), 1001, "all workers share the same 10000/s budget")
		require.Greater(t, client.maxActive, 400)
		require.LessOrEqual(t, client.maxActive, 512)
		require.Zero(t, client.active)
		require.Equal(t, len(client.starts), client.completed)
		require.EqualValues(t, client.acknowledged, c.store.Transactions)
		require.False(t, client.duplicateReference)
		require.False(t, client.overlappingLane)
		require.False(t, client.sharedAccount)
		// The high-rate token bucket may retain at most one millisecond of tokens.
		// At the same virtual instant, a burst therefore cannot exceed ten writes.
		simultaneous := make(map[time.Time]int)
		for _, start := range client.starts {
			simultaneous[start]++
			require.LessOrEqual(t, simultaneous[start], 10)
		}
	})
}

func TestParallelStreamCountsLateAcknowledgementAfterCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, _ int) error {
			<-ctx.Done()
			// The server committed before cancellation, and the transport still returns
			// its successful response while the runner drains outstanding requests.
			time.Sleep(20 * time.Millisecond)
			return nil
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-parallel", "run")}
		start := time.Now()
		require.NoError(t, c.runStream(ctx, client, 100, 1, &bytes.Buffer{}))
		require.Equal(t, 70*time.Millisecond, time.Since(start))
		client.mu.Lock()
		defer client.mu.Unlock()
		require.Len(t, client.starts, 1, "a late success must not restart submissions")
		require.Equal(t, 1, client.completed)
		require.Zero(t, client.active)
		require.EqualValues(t, 1, c.store.Transactions)
		require.Zero(t, c.store.Errors)
	})
}
