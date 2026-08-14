package executor

import "sync"

// budget is a suite-wide, thread-safe cumulative-cost tracker used to enforce
// the --max-cost circuit breaker. A nil *budget (no cap configured) is inert:
// all methods are no-ops, so callers never need to branch on it.
//
// Abort semantics: each sample's cost is added to the running total once the
// sample completes (cost is only known then). Once the total exceeds the cap,
// stopped() reports true and callers stop launching new samples. Samples
// already in flight under --parallel are allowed to finish, so actual spend
// may modestly exceed the cap; no sample is interrupted mid-run.
type budget struct {
	cap      float64
	mu       sync.Mutex
	spent    float64
	exceeded bool
}

// newBudget returns a tracker for the given cap, or nil when cap <= 0 (no cap).
func newBudget(cap float64) *budget {
	if cap <= 0 {
		return nil
	}
	return &budget{cap: cap}
}

// add records the cost of a completed sample and flips the exceeded flag once
// the cumulative total crosses the cap.
func (b *budget) add(cost float64) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spent += cost
	if b.spent > b.cap {
		b.exceeded = true
	}
}

// stopped reports whether the cap has been crossed and no new samples should
// be started.
func (b *budget) stopped() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceeded
}

// report returns the cumulative spend and whether the cap was exceeded.
func (b *budget) report() (spent float64, exceeded bool) {
	if b == nil {
		return 0, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spent, b.exceeded
}
