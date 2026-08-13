package report

import (
	"math"
	"testing"
)

func approxEq(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestWilsonInterval_ZeroTotal(t *testing.T) {
	lo, hi := wilsonInterval(0, 0, wilsonZ95)
	if lo != 0 || hi != 1 {
		t.Errorf("total==0 should give the widest interval (0,1), got (%f,%f)", lo, hi)
	}
}

func TestWilsonInterval_Symmetric(t *testing.T) {
	lo, hi := wilsonInterval(5, 10, wilsonZ95)
	if !(lo < 0.5 && hi > 0.5) {
		t.Errorf("expected lo < 0.5 < hi, got (%f,%f)", lo, hi)
	}
	// Known Wilson 95% interval for 5/10 ≈ [0.237, 0.763].
	if !approxEq(lo, 0.237, 0.02) {
		t.Errorf("lo = %f, want ≈0.237", lo)
	}
	if !approxEq(hi, 0.763, 0.02) {
		t.Errorf("hi = %f, want ≈0.763", hi)
	}
	// The interval is symmetric about the observed proportion.
	if !approxEq(0.5-lo, hi-0.5, 1e-9) {
		t.Errorf("expected symmetry about 0.5, got lo=%f hi=%f", lo, hi)
	}
}

func TestWilsonInterval_AllPass(t *testing.T) {
	lo, hi := wilsonInterval(3, 3, wilsonZ95)
	if hi != 1.0 {
		t.Errorf("all-pass upper bound should clamp to 1.0, got %f", hi)
	}
	if lo >= 1.0 {
		t.Errorf("all-pass lower bound should be < 1.0, got %f", lo)
	}
	if lo <= 0 {
		t.Errorf("all-pass lower bound should be positive, got %f", lo)
	}
}

func TestWilsonInterval_NonePass(t *testing.T) {
	lo, hi := wilsonInterval(0, 4, wilsonZ95)
	if lo != 0.0 {
		t.Errorf("none-pass lower bound should clamp to 0.0, got %f", lo)
	}
	if hi <= 0 || hi >= 1 {
		t.Errorf("none-pass upper bound should be in (0,1), got %f", hi)
	}
}

func TestWilsonInterval_BoundsOrdered(t *testing.T) {
	for _, tc := range []struct{ passed, total int }{{1, 9}, {7, 9}, {50, 100}} {
		lo, hi := wilsonInterval(tc.passed, tc.total, wilsonZ95)
		p := float64(tc.passed) / float64(tc.total)
		if !(lo <= p && p <= hi) {
			t.Errorf("passed=%d total=%d: expected lo<=p<=hi, got lo=%f p=%f hi=%f",
				tc.passed, tc.total, lo, p, hi)
		}
	}
}

func TestIntervalsOverlap(t *testing.T) {
	cases := []struct {
		name               string
		loA, hiA, loB, hiB float64
		want               bool
	}{
		{"clearly overlapping", 0.2, 0.8, 0.5, 0.9, true},
		{"disjoint", 0.0, 0.3, 0.6, 0.9, false},
		{"disjoint reversed order", 0.6, 0.9, 0.0, 0.3, false},
		{"touching endpoints", 0.2, 0.5, 0.5, 0.8, true},
		{"nested", 0.1, 0.9, 0.4, 0.5, true},
	}
	for _, tc := range cases {
		if got := intervalsOverlap(tc.loA, tc.hiA, tc.loB, tc.hiB); got != tc.want {
			t.Errorf("%s: intervalsOverlap = %v, want %v", tc.name, got, tc.want)
		}
	}
}
