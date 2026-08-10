package suite

import "strings"

// expandMatrices replaces each eval's Matrix (if set) with generated Variants
// from the cartesian product of all matrix dimensions.
func expandMatrices(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		if e.Matrix == nil || len(e.Matrix.Dimensions) == 0 {
			continue
		}

		variants := expandMatrix(e.Matrix)
		if len(variants) == 0 {
			continue
		}

		e.Variants = variants
		e.Matrix = nil
	}
}

// expandMatrix computes the cartesian product of all dimensions and returns
// a flat list of variants. The first variant becomes the control.
func expandMatrix(m *Matrix) []Variant {
	combos := cartesianProduct(m.Dimensions)
	variants := make([]Variant, 0, len(combos))
	for _, combo := range combos {
		variants = append(variants, variantFromCombo(combo))
	}
	return variants
}

// variantFromCombo builds a single variant from one combination of dimension
// values, merging each value's fields into the variant.
func variantFromCombo(combo []dimensionEntry) Variant {
	t := Variant{
		DimensionValues: make(map[string]string, len(combo)),
	}

	var nameParts []string
	for _, entry := range combo {
		nameParts = append(nameParts, entry.value.Label)
		t.DimensionValues[entry.dimName] = entry.value.Label
		applyDimensionValue(&t, entry.value)
	}

	t.Name = strings.Join(nameParts, "_")
	return t
}

// applyDimensionValue merges a dimension value's non-empty fields into t.
func applyDimensionValue(t *Variant, v MatrixDimensionValue) {
	if v.Prompt != "" {
		t.Prompt = v.Prompt
	}
	if v.ConfigDir != "" {
		t.ConfigDir = v.ConfigDir
	}
	if v.Model != "" {
		t.Model = v.Model
	}
	if v.Runner != "" {
		t.Runner = v.Runner
	}
	if v.Skill != "" {
		t.Skill = v.Skill
	}
	if len(v.Skills) > 0 {
		t.Skills = append(t.Skills, v.Skills...)
	}
	if v.RunnerConfig != nil {
		t.RunnerConfig = mergeMaps(t.RunnerConfig, v.RunnerConfig)
	}
	if v.Env != nil {
		if t.Env == nil {
			t.Env = make(map[string]string)
		}
		for k, val := range v.Env {
			t.Env[k] = val
		}
	}
}

// dimensionEntry pairs a dimension name with one of its values.
type dimensionEntry struct {
	dimName string
	value   MatrixDimensionValue
}

// cartesianProduct returns all combinations of dimension values.
func cartesianProduct(dims []MatrixDimension) [][]dimensionEntry {
	if len(dims) == 0 {
		return nil
	}

	result := [][]dimensionEntry{{}}

	for _, dim := range dims {
		if len(dim.Values) == 0 {
			continue
		}
		var next [][]dimensionEntry
		for _, combo := range result {
			for _, val := range dim.Values {
				entry := dimensionEntry{dimName: dim.Name, value: val}
				newCombo := make([]dimensionEntry, len(combo)+1)
				copy(newCombo, combo)
				newCombo[len(combo)] = entry
				next = append(next, newCombo)
			}
		}
		result = next
	}

	return result
}
