package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
)

// tagsCmd is the parent command for tag management sub-commands.
var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage and inspect task tags",
}

var tagsAddCmd = &cobra.Command{
	Use:   "add <task-id> <tag...>",
	Short: "Add tags to a task",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		tags := args[1:]

		s, err := openStore()
		if err != nil {
			return err
		}

		if err := s.AddTags(id, tags); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", id)
			}
			return err
		}

		fmt.Printf("Added tags [%s] to task %s\n", strings.Join(tags, ", "), id)
		return nil
	},
}

var tagsRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <tag...>",
	Short: "Remove tags from a task",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		tags := args[1:]

		s, err := openStore()
		if err != nil {
			return err
		}

		if err := s.RemoveTags(id, tags); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", id)
			}
			return err
		}

		fmt.Printf("Removed tags [%s] from task %s\n", strings.Join(tags, ", "), id)
		return nil
	},
}

var tagsListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List tags on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		s, err := openStore()
		if err != nil {
			return err
		}

		t, err := s.Get(id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", id)
			}
			return err
		}

		if len(t.Tags) == 0 {
			fmt.Printf("Task %s has no tags.\n", id)
			return nil
		}

		fmt.Fprintf(os.Stdout, "Task %s tags: %s\n", id, strings.Join(t.Tags, ", "))
		return nil
	},
}

func init() {
	tagsCmd.AddCommand(tagsAddCmd)
	tagsCmd.AddCommand(tagsRemoveCmd)
	tagsCmd.AddCommand(tagsListCmd)
}
