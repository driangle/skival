package suite

import "testing"

func TestExpandMatrix_TwoDimensions(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "runner",
				Values: []MatrixDimensionValue{
					{Label: "claude-code", Runner: "claude-code"},
					{Label: "ollama", Runner: "ollama"},
				},
			},
			{
				Name: "model",
				Values: []MatrixDimensionValue{
					{Label: "opus", Model: "claude-opus-4-6"},
					{Label: "sonnet", Model: "claude-sonnet-4-6"},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(variants))
	}

	// Verify deterministic naming and field assignment
	expected := []struct {
		name   string
		runner string
		model  string
	}{
		{"claude-code_opus", "claude-code", "claude-opus-4-6"},
		{"claude-code_sonnet", "claude-code", "claude-sonnet-4-6"},
		{"ollama_opus", "ollama", "claude-opus-4-6"},
		{"ollama_sonnet", "ollama", "claude-sonnet-4-6"},
	}

	for i, exp := range expected {
		if variants[i].Name != exp.name {
			t.Errorf("variant[%d].Name = %q, want %q", i, variants[i].Name, exp.name)
		}
		if variants[i].Runner != exp.runner {
			t.Errorf("variant[%d].Runner = %q, want %q", i, variants[i].Runner, exp.runner)
		}
		if variants[i].Model != exp.model {
			t.Errorf("variant[%d].Model = %q, want %q", i, variants[i].Model, exp.model)
		}
	}
}

func TestExpandMatrix_SingleDimension(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "model",
				Values: []MatrixDimensionValue{
					{Label: "opus", Model: "claude-opus-4-6"},
					{Label: "sonnet", Model: "claude-sonnet-4-6"},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	if variants[0].Name != "opus" {
		t.Errorf("expected name %q, got %q", "opus", variants[0].Name)
	}
	if variants[1].Name != "sonnet" {
		t.Errorf("expected name %q, got %q", "sonnet", variants[1].Name)
	}
}

func TestExpandMatrix_DimensionValues(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "runner",
				Values: []MatrixDimensionValue{
					{Label: "claude-code", Runner: "claude-code"},
				},
			},
			{
				Name: "model",
				Values: []MatrixDimensionValue{
					{Label: "opus", Model: "claude-opus-4-6"},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}

	if variants[0].DimensionValues["runner"] != "claude-code" {
		t.Errorf("expected dimension runner=%q, got %q", "claude-code", variants[0].DimensionValues["runner"])
	}
	if variants[0].DimensionValues["model"] != "opus" {
		t.Errorf("expected dimension model=%q, got %q", "opus", variants[0].DimensionValues["model"])
	}
}

func TestExpandMatrix_SkillAndEnv(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "skill",
				Values: []MatrixDimensionValue{
					{Label: "baseline"},
					{Label: "with-tools", Skill: "./skills/tools.md", Env: map[string]string{"TOOLS": "true"}},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	if variants[0].Skill != "" {
		t.Errorf("baseline should have empty skill, got %q", variants[0].Skill)
	}
	if variants[1].Skill != "./skills/tools.md" {
		t.Errorf("with-tools skill = %q, want %q", variants[1].Skill, "./skills/tools.md")
	}
	if variants[1].Env["TOOLS"] != "true" {
		t.Errorf("with-tools env TOOLS = %q, want %q", variants[1].Env["TOOLS"], "true")
	}
}

func TestExpandMatrix_RunnerConfig(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "config",
				Values: []MatrixDimensionValue{
					{Label: "default"},
					{Label: "custom", RunnerConfig: map[string]any{"max_turns": 20}},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if variants[0].RunnerConfig != nil {
		t.Errorf("default should have nil runner_config, got %v", variants[0].RunnerConfig)
	}
	if variants[1].RunnerConfig["max_turns"] != 20 {
		t.Errorf("custom runner_config max_turns = %v, want 20", variants[1].RunnerConfig["max_turns"])
	}
}

func TestExpandMatrix_ThreeDimensions(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{Name: "a", Values: []MatrixDimensionValue{{Label: "a1"}, {Label: "a2"}}},
			{Name: "b", Values: []MatrixDimensionValue{{Label: "b1"}, {Label: "b2"}}},
			{Name: "c", Values: []MatrixDimensionValue{{Label: "c1"}, {Label: "c2"}}},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 8 {
		t.Fatalf("expected 2*2*2=8 variants, got %d", len(variants))
	}

	// First variant should be a1_b1_c1
	if variants[0].Name != "a1_b1_c1" {
		t.Errorf("first variant name = %q, want %q", variants[0].Name, "a1_b1_c1")
	}
	// Last variant should be a2_b2_c2
	if variants[7].Name != "a2_b2_c2" {
		t.Errorf("last variant name = %q, want %q", variants[7].Name, "a2_b2_c2")
	}
}

func TestExpandMatrices_SetsVariants(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{{
			ID:     "e1",
			Prompt: "test",
			Model:  "claude-sonnet-4-6",
			Matrix: &Matrix{
				Dimensions: []MatrixDimension{
					{
						Name: "runner",
						Values: []MatrixDimensionValue{
							{Label: "claude-code", Runner: "claude-code"},
							{Label: "ollama", Runner: "ollama"},
						},
					},
				},
			},
		}},
	}

	expandMatrices(s)

	if s.Evals[0].Matrix != nil {
		t.Error("matrix should be nil after expansion")
	}
	if len(s.Evals[0].Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(s.Evals[0].Variants))
	}
	if s.Evals[0].Variants[0].Name != "claude-code" {
		t.Errorf("variant[0] name = %q, want %q", s.Evals[0].Variants[0].Name, "claude-code")
	}
	if s.Evals[0].Variants[1].Name != "ollama" {
		t.Errorf("variant[1] name = %q, want %q", s.Evals[0].Variants[1].Name, "ollama")
	}
}

func TestExpandMatrices_SkipsEvalsWithoutMatrix(t *testing.T) {
	s := &Suite{
		Version: 1,
		Evals: []Eval{{
			ID:       "e1",
			Prompt:   "test",
			Model:    "claude-sonnet-4-6",
			Variants: []Variant{{Name: "ctrl"}},
		}},
	}

	expandMatrices(s)

	if s.Evals[0].Variants[0].Name != "ctrl" {
		t.Error("variants should be unchanged for non-matrix evals")
	}
}

func TestExpandMatrix_PromptAndConfigDir(t *testing.T) {
	m := &Matrix{
		Dimensions: []MatrixDimension{
			{
				Name: "config",
				Values: []MatrixDimensionValue{
					{Label: "default", Prompt: "do A"},
					{Label: "custom", Prompt: "do B", ConfigDir: "/tmp/custom-config"},
				},
			},
		},
	}

	variants := expandMatrix(m)

	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	if variants[0].Prompt != "do A" {
		t.Errorf("default prompt = %q, want %q", variants[0].Prompt, "do A")
	}
	if variants[0].ConfigDir != "" {
		t.Errorf("default config_dir should be empty, got %q", variants[0].ConfigDir)
	}
	if variants[1].Prompt != "do B" {
		t.Errorf("custom prompt = %q, want %q", variants[1].Prompt, "do B")
	}
	if variants[1].ConfigDir != "/tmp/custom-config" {
		t.Errorf("custom config_dir = %q, want %q", variants[1].ConfigDir, "/tmp/custom-config")
	}
}

func TestCartesianProduct_Empty(t *testing.T) {
	result := cartesianProduct(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}
