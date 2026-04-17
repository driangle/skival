package suite

import "strings"

// expandMatrices replaces each eval's Matrix (if set) with generated Treatments
// from the cartesian product of all matrix dimensions.
func expandMatrices(s *Suite) {
	for i := range s.Evals {
		e := &s.Evals[i]
		if e.Matrix == nil || len(e.Matrix.Dimensions) == 0 {
			continue
		}

		treatments := expandMatrix(e.Matrix)
		if len(treatments) == 0 {
			continue
		}

		e.Treatments = Treatments{
			Control:    treatments[0],
			Variations: treatments[1:],
		}
		e.Matrix = nil
	}
}

// expandMatrix computes the cartesian product of all dimensions and returns
// a flat list of treatments. The first treatment becomes the control.
func expandMatrix(m *Matrix) []Treatment {
	combos := cartesianProduct(m.Dimensions)
	treatments := make([]Treatment, 0, len(combos))

	for _, combo := range combos {
		t := Treatment{
			DimensionValues: make(map[string]string, len(combo)),
		}

		var nameParts []string
		for _, entry := range combo {
			nameParts = append(nameParts, entry.value.Label)
			t.DimensionValues[entry.dimName] = entry.value.Label

			if entry.value.Prompt != "" {
				t.Prompt = entry.value.Prompt
			}
			if entry.value.ConfigDir != "" {
				t.ConfigDir = entry.value.ConfigDir
			}
			if entry.value.Model != "" {
				t.Model = entry.value.Model
			}
			if entry.value.Runner != "" {
				t.Runner = entry.value.Runner
			}
			if entry.value.Skill != "" {
				t.Skill = entry.value.Skill
			}
			if len(entry.value.Skills) > 0 {
				t.Skills = append(t.Skills, entry.value.Skills...)
			}
			if entry.value.RunnerConfig != nil {
				t.RunnerConfig = mergeMaps(t.RunnerConfig, entry.value.RunnerConfig)
			}
			if entry.value.Env != nil {
				if t.Env == nil {
					t.Env = make(map[string]string)
				}
				for k, v := range entry.value.Env {
					t.Env[k] = v
				}
			}
		}

		t.Name = strings.Join(nameParts, "_")
		treatments = append(treatments, t)
	}

	return treatments
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
