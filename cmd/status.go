package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

var statusCmd = &cobra.Command{
	Use:   "status <task-id> <new-status>",
	Short: "Update the status of a task",
	Long: `Update the status of a task.

Valid status values: todo, in-progress, done, blocked`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		newStatus := task.Status(args[1])

		if !newStatus.IsValid() {
			return fmt.Errorf("unknown status %q; valid values: todo, in-progress, done, blocked", args[1])
		}

		s, err := openStore()
		if err != nil {
			return err
		}
		t, err := s.UpdateStatus(id, newStatus)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", id)
			}
			return err
		}
		fmt.Printf("Updated %s status to %s\n", t.ID, t.Status)
		return nil
	},
}
