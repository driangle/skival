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
			fmt.Fprintf(out, "\n  eval %q\n", eval.ID)
			if eval.Name != "" {
				fmt.Fprintf(out, "    name:       %s\n", eval.Name)
			}
			fmt.Fprintf(out, "    variants:   %d (", len(eval.Variants))
			for i, v := range eval.Variants {
				if i > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "%q", v.Name)
			}
			fmt.Fprintf(out, ")\n")

			if len(eval.Verify) > 0 {
				fmt.Fprintf(out, "    verifiers:  %d\n", len(eval.Verify))
			}
		}

		fmt.Fprintln(out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
