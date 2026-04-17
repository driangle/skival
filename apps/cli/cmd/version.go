package cmd

import "github.com/spf13/cobra"

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the skival version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
