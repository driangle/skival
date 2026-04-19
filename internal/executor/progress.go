package executor

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/driangle/skival/internal/result"
	"github.com/driangle/skival/internal/verifier"
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

func (p *progress) evalStart(evalNum, totalEvals int, evalLabel string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] eval %d/%d: %s\n",
		p.elapsed(), evalNum, totalEvals, evalLabel)
}

func (p *progress) evalError(evalLabel string, err error) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s: ERROR: %v\n", p.elapsed(), evalLabel, err)
}

func (p *progress) sampleStart(evalLabel, variantName string, sample, totalSamples int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > sample %d/%d (cost: $%.4f)",
		p.elapsed(), evalLabel, variantName, sample, totalSamples, p.totalCost)
}

func (p *progress) workdir(evalLabel, variantName, dir string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > workdir: %s\n",
		p.elapsed(), evalLabel, variantName, dir)
}

func (p *progress) sessionID(evalLabel, variantName, sessionID string) {
	if p == nil || sessionID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > session: %s\n",
		p.elapsed(), evalLabel, variantName, sessionID)
}

func (p *progress) verifyResults(evalLabel, variantName string, steps []verifier.StepResult) {
	if p == nil || len(steps) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, step := range steps {
		status := "PASS"
		if !step.Result.Pass {
			status = "FAIL"
		}
		line := fmt.Sprintf("\r\033[K[%s] %s > %s > verify %s: %s",
			p.elapsed(), evalLabel, variantName, step.Name, status)
		if !step.Result.Pass && step.Result.Reason != "" {
			line += fmt.Sprintf(" (%s)", step.Result.Reason)
		}
		fmt.Fprintln(p.w, line)
	}
}

func (p *progress) sampleDone(evalLabel, variantName string, sample int, costUSD float64, pass *bool) {
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
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > sample %d: %s (cost: $%.4f)\n",
		p.elapsed(), evalLabel, variantName, sample, status, costUSD)
}

func (p *progress) skippedVariants(evalLabel string, skipped []result.SkippedVariant) {
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
		p.elapsed(), len(skipped), evalLabel, fmt.Sprintf("%v", names))
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
