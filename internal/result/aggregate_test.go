package result

import (
	"math"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestComputeAggregate_EmptyRuns(t *testing.T) {
	if got := ComputeAggregate(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
	if got := ComputeAggregate([]RunResult{}); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestComputeAggregate_SingleRun(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 0.50, DurationMs: 2000},
	}
	agg := ComputeAggregate(runs)
	if agg == nil {
		t.Fatal("expected non-nil aggregate")
	}
	if agg.MedianCostUSD != 0.50 {
		t.Errorf("median cost = %f, want 0.50", agg.MedianCostUSD)
	}
	if agg.MinCostUSD != 0.50 || agg.MaxCostUSD != 0.50 {
		t.Errorf("min/max cost = %f/%f, want 0.50/0.50", agg.MinCostUSD, agg.MaxCostUSD)
	}
	if agg.MedianDurationMs != 2000 {
		t.Errorf("median duration = %d, want 2000", agg.MedianDurationMs)
	}
	if agg.CostCV != nil {
		t.Errorf("cost CV should be nil for single run, got %f", *agg.CostCV)
	}
	if agg.DurationCV != nil {
		t.Errorf("duration CV should be nil for single run, got %f", *agg.DurationCV)
	}
}

func TestComputeAggregate_TwoRuns(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 1.0, DurationMs: 1000},
		{CostUSD: 3.0, DurationMs: 3000},
	}
	agg := ComputeAggregate(runs)

	if agg.MedianCostUSD != 2.0 {
		t.Errorf("median cost = %f, want 2.0", agg.MedianCostUSD)
	}
	if agg.MedianDurationMs != 2000 {
		t.Errorf("median duration = %d, want 2000", agg.MedianDurationMs)
	}
	if agg.MinCostUSD != 1.0 || agg.MaxCostUSD != 3.0 {
		t.Errorf("min/max cost = %f/%f, want 1.0/3.0", agg.MinCostUSD, agg.MaxCostUSD)
	}
	if agg.CostCV != nil {
		t.Errorf("cost CV should be nil for 2 runs")
	}
}

func TestComputeAggregate_ThreeRunsOdd(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 3.0, DurationMs: 300},
		{CostUSD: 1.0, DurationMs: 100},
		{CostUSD: 2.0, DurationMs: 200},
	}
	agg := ComputeAggregate(runs)

	if agg.MedianCostUSD != 2.0 {
		t.Errorf("median cost = %f, want 2.0", agg.MedianCostUSD)
	}
	if agg.MedianDurationMs != 200 {
		t.Errorf("median duration = %d, want 200", agg.MedianDurationMs)
	}
	if agg.CostCV == nil {
		t.Fatal("cost CV should be non-nil for 3 runs")
	}
	// mean=2, stddev=sqrt(2/3)≈0.8165, CV≈0.4082
	if math.Abs(*agg.CostCV-0.4082) > 0.001 {
		t.Errorf("cost CV = %f, want ~0.4082", *agg.CostCV)
	}
}

func TestComputeAggregate_FourRunsEven(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 4.0, DurationMs: 400},
		{CostUSD: 1.0, DurationMs: 100},
		{CostUSD: 3.0, DurationMs: 300},
		{CostUSD: 2.0, DurationMs: 200},
	}
	agg := ComputeAggregate(runs)

	// Sorted: 1, 2, 3, 4 → median = (2+3)/2 = 2.5
	if agg.MedianCostUSD != 2.5 {
		t.Errorf("median cost = %f, want 2.5", agg.MedianCostUSD)
	}
	if agg.MedianDurationMs != 250 {
		t.Errorf("median duration = %d, want 250", agg.MedianDurationMs)
	}
}

func TestConservativePass_AllPass(t *testing.T) {
	runs := []RunResult{
		{Pass: boolPtr(true)},
		{Pass: boolPtr(true)},
		{Pass: boolPtr(true)},
	}
	agg := ComputeAggregate(runs)
	if agg.Pass == nil || !*agg.Pass {
		t.Error("expected pass=true when all runs pass")
	}
}

func TestConservativePass_OneFail(t *testing.T) {
	runs := []RunResult{
		{Pass: boolPtr(true)},
		{Pass: boolPtr(false)},
		{Pass: boolPtr(true)},
	}
	agg := ComputeAggregate(runs)
	if agg.Pass == nil || *agg.Pass {
		t.Error("expected pass=false when any run fails")
	}
}

func TestConservativePass_OneNil(t *testing.T) {
	runs := []RunResult{
		{Pass: boolPtr(true)},
		{Pass: nil},
		{Pass: boolPtr(true)},
	}
	agg := ComputeAggregate(runs)
	if agg.Pass != nil {
		t.Errorf("expected pass=nil when any run is unverified, got %v", *agg.Pass)
	}
}

func TestConservativePass_FailAndNil(t *testing.T) {
	runs := []RunResult{
		{Pass: boolPtr(false)},
		{Pass: nil},
		{Pass: boolPtr(true)},
	}
	agg := ComputeAggregate(runs)
	if agg.Pass == nil || *agg.Pass {
		t.Error("expected pass=false when fail present (fail > nil)")
	}
}

func TestConservativePass_AllNil(t *testing.T) {
	runs := []RunResult{
		{Pass: nil},
		{Pass: nil},
	}
	agg := ComputeAggregate(runs)
	if agg.Pass != nil {
		t.Errorf("expected pass=nil when all unverified, got %v", *agg.Pass)
	}
}

func TestCV_AllSameValues(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 5.0, DurationMs: 100},
		{CostUSD: 5.0, DurationMs: 100},
		{CostUSD: 5.0, DurationMs: 100},
	}
	agg := ComputeAggregate(runs)
	if agg.CostCV == nil {
		t.Fatal("cost CV should be non-nil for 3 runs")
	}
	if *agg.CostCV != 0 {
		t.Errorf("cost CV = %f, want 0 for identical values", *agg.CostCV)
	}
}

func TestCV_AllZeroMean(t *testing.T) {
	runs := []RunResult{
		{CostUSD: 0, DurationMs: 0},
		{CostUSD: 0, DurationMs: 0},
		{CostUSD: 0, DurationMs: 0},
	}
	agg := ComputeAggregate(runs)
	if agg.CostCV != nil {
		t.Errorf("cost CV should be nil when mean is zero, got %f", *agg.CostCV)
	}
}
