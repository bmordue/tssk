package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/task"
)

var (
	addTitle    string
	addDetail   string
	addDeps     []string
	addTags     []string
	addPriority string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task",
	Long:  `Add a new task with a title, optional detail text, optional dependencies, optional tags, and optional priority.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(addTitle) == "" {
			return fmt.Errorf("--title is required")
		}

		priority := task.Priority(addPriority)
		if addPriority != "" && !priority.IsValid() {
			return fmt.Errorf("unknown priority %q; valid values: low, medium, high, critical", addPriority)
		}
		if addPriority == "" {
			priority = task.PriorityNone
		}

		s, err := openStore()
		if err != nil {
			return err
		}
		t, err := s.Add(addTitle, addDetail, addDeps, addTags, priority)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Added task %s: %s\n", t.ID, t.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addTitle, "title", "t", "", "Task title (required)")
	addCmd.Flags().StringVarP(&addDetail, "detail", "d", "", "Task detail text (written to a markdown file)")
	addCmd.Flags().StringSliceVarP(&addDeps, "deps", "D", nil, "Comma-separated list of dependency task IDs")
	addCmd.Flags().StringSliceVarP(&addTags, "tags", "T", nil, "Comma-separated list of tags")
	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "", "Task priority (low, medium, high, critical)")
}
