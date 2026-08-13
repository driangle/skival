package report

import "math"

// wilsonZ95 is the standard-normal quantile for a 95% two-sided confidence
// interval (z ≈ 1.96).
const wilsonZ95 = 1.96

// wilsonInterval returns the Wilson score interval for a binomial proportion
// (passed out of total successes), clamped to [0,1]. The Wilson interval stays
// well-behaved at small n and at the boundaries (0% / 100%), unlike the normal
// approximation. With total == 0 the proportion is undefined, so it returns the
// widest honest interval, (0, 1).
func wilsonInterval(passed, total int, z float64) (lo, hi float64) {
	if total <= 0 {
		return 0, 1
	}

	n := float64(total)
	pHat := float64(passed) / n
	z2 := z * z

	denom := 1 + z2/n
	center := (pHat + z2/(2*n)) / denom
	half := (z / denom) * math.Sqrt(pHat*(1-pHat)/n+z2/(4*n*n))

	return clamp01(center - half), clamp01(center + half)
}

// clamp01 bounds x to the [0,1] interval.
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// intervalsOverlap reports whether the closed intervals [loA,hiA] and [loB,hiB]
// share any point. Touching endpoints count as overlapping.
func intervalsOverlap(loA, hiA, loB, hiB float64) bool {
	return loA <= hiB && loB <= hiA
}
