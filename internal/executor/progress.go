package executor

import (
	"fmt"
	"io"
	"time"
)

// progress tracks and displays execution progress.
type progress struct {
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
	fmt.Fprintf(p.w, "\r\033[K[%s] eval %d/%d: %s",
		p.elapsed(), evalNum, totalEvals, evalName)
}

func (p *progress) evalError(evalName string, err error) {
	if p == nil {
		return
	}
	fmt.Fprintf(p.w, "\r\033[K[%s] %s: ERROR: %v\n", p.elapsed(), evalName, err)
}

func (p *progress) sampleStart(evalName, treatName string, sample, totalSamples int) {
	if p == nil {
		return
	}
	fmt.Fprintf(p.w, "\r\033[K[%s] %s > %s > sample %d/%d (cost: $%.4f)",
		p.elapsed(), evalName, treatName, sample, totalSamples, p.totalCost)
}

func (p *progress) sampleDone(costUSD float64, pass *bool) {
	if p == nil {
		return
	}
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

func (p *progress) finish() {
	if p == nil {
		return
	}
	fmt.Fprintf(p.w, "\r\033[K[%s] done, total cost: $%.4f\n", p.elapsed(), p.totalCost)
}

func (p *progress) elapsed() string {
	d := time.Since(p.startedAt).Truncate(time.Second)
	return fmt.Sprintf("%v", d)
}
