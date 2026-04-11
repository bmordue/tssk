package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

var editTitle string
var editDetail string
var editPriority string

var editCmd = &cobra.Command{
	Use:   "edit <task-id>",
	Short: "Edit a task's title and/or detail",
	Long: `Edit a task's title and/or detail.

Use --title to update the task title.
Use --detail to update the task detail text.
Use --priority to update the task priority.
Both flags can be used together to update multiple fields.

Note: Changing the title will update the content-addressed hash, which may
result in a new detail file being created.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if editTitle == "" && editDetail == "" && editPriority == "" {
			return fmt.Errorf("at least one of --title, --detail, or --priority must be provided")
		}

		id := args[0]

		s, err := openStore()
		if err != nil {
			return err
		}

		if editPriority != "" {
			priority := task.Priority(editPriority)
			if !priority.IsValid() {
				return fmt.Errorf("unknown priority %q; valid values: low, medium, high, critical", editPriority)
			}
			t, err := s.UpdatePriority(id, priority)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("task %s not found", id)
				}
				return fmt.Errorf("updating priority: %w", err)
			}
			fmt.Printf("Updated %s priority to %q\n", t.ID, t.Priority)
		}

		if editTitle != "" {
			t, err := s.UpdateTitle(id, editTitle)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("task %s not found", id)
				}
				return fmt.Errorf("updating title: %w", err)
			}
			fmt.Printf("Updated %s title to %q\n", t.ID, t.Title)
		}

		if editDetail != "" {
			// Get the task (may have been updated by UpdateTitle)
			t, err := s.Get(id)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("task %s not found", id)
				}
				return fmt.Errorf("getting task: %w", err)
			}

			_, err = s.UpdateDetail(id, editDetail)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("task %s not found", id)
				}
				return fmt.Errorf("updating detail: %w", err)
			}
			fmt.Printf("Updated %s detail\n", t.ID)
		}

		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "Update task title")
	editCmd.Flags().StringVar(&editDetail, "detail", "", "Update task detail text")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Update task priority (low, medium, high, critical)")
}
