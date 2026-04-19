package executor

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/driangle/skival/internal/result"
)

// progress tracks and displays execution progress.
type progress struct {
	mu        sync.Mutex
	w         io.Writer
	startedAt time.Time
	totalCost float64
}

func newProgress(w io.Writer) *progress {
	if w == nil {
		return nil
	}
	return &progress{w: w, startedAt: time.Now()}
}

func (p *progress) evalStart(evalNum, totalEvals int, evalName string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] eval %d/%d: %s",
		p.elapsed(), evalNum, totalEvals, evalName)
}

func (p *progress) evalError(evalName string, err error) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s: ERROR: %v\n", p.elapsed(), evalName, err)
}

func (p *progress) sampleStart(evalName, treatName string, sample, totalSamples int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > sample %d/%d (cost: $%.4f)",
		p.elapsed(), evalName, treatName, sample, totalSamples, p.totalCost)
}

func (p *progress) workdir(evalName, treatName, dir string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > workdir: %s\n",
		p.elapsed(), evalName, treatName, dir)
}

func (p *progress) sampleDone(costUSD float64, pass *bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalCost += costUSD
	status := "done"
	if pass != nil {
		if *pass {
			status = "PASS"
		} else {
			status = "FAIL"
		}
	}
	fmt.Fprintf(p.w, "\r\033[K[%s] sample done: %s (cost: $%.4f)\n",
		p.elapsed(), status, costUSD)
}

func (p *progress) skippedVariants(evalName string, skipped []result.SkippedVariant) {
	if p == nil || len(skipped) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	names := make([]string, len(skipped))
	for i, s := range skipped {
		names[i] = s.Name
	}
	fmt.Fprintf(p.w, "\r\033[K[%s] Skipping %d remaining variants for eval %q: %s\n",
		p.elapsed(), len(skipped), evalName, fmt.Sprintf("%v", names))
}

func (p *progress) finish() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] done, total cost: $%.4f\n", p.elapsed(), p.totalCost)
}

func (p *progress) elapsed() string {
	d := time.Since(p.startedAt).Truncate(time.Second)
	return fmt.Sprintf("%v", d)
}
