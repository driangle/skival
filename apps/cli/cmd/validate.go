package cmd

import (
	"fmt"

	"github.com/driangle/skival/internal/suite"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <suite.yaml>",
	Short: "Validate a suite file",
	Long:  "Parse and validate a suite YAML file without executing it. Reports any structural errors found.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		out := cmd.OutOrStdout()
		errOut := cmd.ErrOrStderr()

		s, err := suite.Load(path)
		if err != nil {
			fmt.Fprintf(errOut, "FAIL %s\n\n%v\n", path, err)
			return err
		}

		fmt.Fprintf(out, "OK %s\n\n", path)
		fmt.Fprintf(out, "  version:     %d\n", s.Version)
		if s.Description != "" {
			fmt.Fprintf(out, "  description: %s\n", s.Description)
		}
		fmt.Fprintf(out, "  evals:       %d\n", len(s.Evals))

		for _, eval := range s.Evals {
			variations := len(eval.Treatments.Variations)
			treatments := 1 + variations // control + variations

			fmt.Fprintf(out, "\n  eval %q\n", eval.ID)
			if eval.Name != "" {
				fmt.Fprintf(out, "    name:       %s\n", eval.Name)
			}
			if eval.Complexity != "" {
				fmt.Fprintf(out, "    complexity: %s\n", eval.Complexity)
			}
			fmt.Fprintf(out, "    treatments: %d (control: %q", treatments, eval.Treatments.Control.Name)
			for _, v := range eval.Treatments.Variations {
				fmt.Fprintf(out, ", %q", v.Name)
			}
			fmt.Fprintf(out, ")\n")

			verifiers := countVerifiers(eval.Correctness)
			if verifiers > 0 {
				fmt.Fprintf(out, "    verifiers:  %d\n", verifiers)
			}
		}

		fmt.Fprintln(out)
		return nil
	},
}

func countVerifiers(c suite.Correctness) int {
	count := 0
	if c.Execute != nil && *c.Execute {
		count++
	}
	if c.Compiles != "" {
		count++
	}
	if len(c.ExpectedOutput) > 0 {
		count++
	}
	if c.Script != "" {
		count++
	}
	if len(c.State) > 0 {
		count++
	}
	if len(c.Judge) > 0 {
		count++
	}
	return count
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
