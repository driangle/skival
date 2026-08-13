package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/driangle/skival/internal/suite"
)

// fixtureSuite builds a small suite with two evals, each with a couple of
// variants, for filter-validation tests.
func fixtureSuite() *suite.Suite {
	return &suite.Suite{
		Evals: []suite.Eval{
			{ID: "add-dependency", Variants: []suite.Variant{{Name: "control"}, {Name: "opus"}}},
			{ID: "remove-bug", Variants: []suite.Variant{{Name: "control"}, {Name: "sonnet"}}},
		},
	}
}

func TestValidateFilters_UnmatchedEval(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{EvalIDs: []string{"add-bugs"}})
	if err == nil {
		t.Fatal("expected error for unmatched eval id")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"add-bugs"`) {
		t.Errorf("error should mention the bad value, got %q", msg)
	}
	if !strings.Contains(msg, "add-dependency") || !strings.Contains(msg, "remove-bug") {
		t.Errorf("error should list valid eval ids, got %q", msg)
	}
}

func TestValidateFilters_UnmatchedVariant(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{Variants: []string{"haiku"}})
	if err == nil {
		t.Fatal("expected error for unmatched variant name")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"haiku"`) {
		t.Errorf("error should mention the bad value, got %q", msg)
	}
	if !strings.Contains(msg, "control") || !strings.Contains(msg, "opus") || !strings.Contains(msg, "sonnet") {
		t.Errorf("error should list valid variant names, got %q", msg)
	}
}

func TestValidateFilters_PartiallyValidStillErrors(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{EvalIDs: []string{"remove-bug", "typo"}})
	if err == nil {
		t.Fatal("expected error when one eval id is a typo")
	}
	if !strings.Contains(err.Error(), `"typo"`) {
		t.Errorf("error should mention the unmatched value, got %q", err.Error())
	}
}

func TestValidateFilters_BothUnmatched(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{
		EvalIDs:  []string{"nope"},
		Variants: []string{"gpt"},
	})
	if err == nil {
		t.Fatal("expected error when both filters are unmatched")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no eval") || !strings.Contains(msg, "no variant") {
		t.Errorf("error should mention both evals and variants, got %q", msg)
	}
}

func TestValidateFilters_MultipleUnmatchedPluralized(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{EvalIDs: []string{"a", "b"}})
	if err == nil {
		t.Fatal("expected error for two unmatched eval ids")
	}
	if !strings.Contains(err.Error(), "no evals match") {
		t.Errorf("expected pluralized message, got %q", err.Error())
	}
}

func TestValidateFilters_AllValid(t *testing.T) {
	err := validateFilters(fixtureSuite(), &Options{
		EvalIDs:  []string{"add-dependency", "remove-bug"},
		Variants: []string{"control", "opus"},
	})
	if err != nil {
		t.Errorf("expected no error for all-valid filters, got %v", err)
	}
}

func TestValidateFilters_EmptyFiltersRunAll(t *testing.T) {
	if err := validateFilters(fixtureSuite(), &Options{}); err != nil {
		t.Errorf("expected no error for absent filters, got %v", err)
	}
}

func TestExecute_UnmatchedFilterErrorsBeforeRun(t *testing.T) {
	_, err := Execute(context.Background(), fixtureSuite(), nil, &Options{EvalIDs: []string{"add-bugs"}})
	if err == nil {
		t.Fatal("expected Execute to return an error for a typo'd eval filter")
	}
	if !strings.Contains(err.Error(), `"add-bugs"`) {
		t.Errorf("expected error to mention the bad value, got %q", err.Error())
	}
}
