package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/driangle/skival/internal/suite"
)

// validateFilters ensures every --evals and --variants filter value matches a
// known id in the suite. It returns a hard error (before any run happens) when
// a value matches nothing, so a typo'd filter never becomes a silent no-op.
func validateFilters(s *suite.Suite, opts *Options) error {
	validEvals := evalIDSet(s)
	validVariants := variantNameSet(s)

	unmatchedEvals := unmatched(opts.EvalIDs, validEvals)
	unmatchedVariants := unmatched(opts.Variants, validVariants)

	if len(unmatchedEvals) == 0 && len(unmatchedVariants) == 0 {
		return nil
	}

	var parts []string
	if len(unmatchedEvals) > 0 {
		parts = append(parts, filterError("eval", unmatchedEvals, validEvals))
	}
	if len(unmatchedVariants) > 0 {
		parts = append(parts, filterError("variant", unmatchedVariants, validVariants))
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

// filterError builds a message like:
//
//	no eval matches "add-bugs". Valid evals: add-dependency, remove-bug
func filterError(kind string, unmatchedIDs []string, valid map[string]bool) string {
	verb := "matches"
	subject := kind
	if len(unmatchedIDs) > 1 {
		verb = "match"
		subject = kind + "s"
	}
	return fmt.Sprintf("no %s %s %s. Valid %ss: %s",
		subject, verb, quoteList(unmatchedIDs), kind, strings.Join(sortedKeys(valid), ", "))
}

// unmatched returns the filter values not present in valid, preserving order.
func unmatched(filter []string, valid map[string]bool) []string {
	var out []string
	for _, v := range filter {
		if !valid[v] {
			out = append(out, v)
		}
	}
	return out
}

func evalIDSet(s *suite.Suite) map[string]bool {
	set := make(map[string]bool, len(s.Evals))
	for i := range s.Evals {
		set[s.Evals[i].ID] = true
	}
	return set
}

func variantNameSet(s *suite.Suite) map[string]bool {
	set := make(map[string]bool)
	for i := range s.Evals {
		for j := range s.Evals[i].Variants {
			set[s.Evals[i].Variants[j].Name] = true
		}
	}
	return set
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func quoteList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}
