package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for the tssk CLI.
var rootCmd = &cobra.Command{
	Use:   "tssk",
	Short: "tssk – manage repository tasks from the command line",
	Long: `tssk is a command line tool for managing repository tasks.

Task metadata is stored in tasks.jsonl at the project root.
Full task detail text is kept in content-addressed markdown files under docs/.`,
}

// Execute adds all child commands to the root command and runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(depsCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(serveCmd)
}
