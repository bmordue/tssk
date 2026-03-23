package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

// depsCmd is the parent command for dependency management sub-commands.
var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Manage and inspect task dependencies",
}

var depsAddCmd = &cobra.Command{
	Use:   "add <task-id> <dep-id>",
	Short: "Add a dependency to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.New(projectRoot())
		if err := s.AddDep(args[0], args[1]); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", args[0])
			}
			return err
		}
		fmt.Printf("Task %s now depends on %s\n", args[0], args[1])
		return nil
	},
}

var depsRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <dep-id>",
	Short: "Remove a dependency from a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.New(projectRoot())
		if err := s.RemoveDep(args[0], args[1]); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", args[0])
			}
			return err
		}
		fmt.Printf("Removed dependency %s from task %s\n", args[1], args[0])
		return nil
	},
}

var depsCheckCmd = &cobra.Command{
	Use:   "check <task-id>",
	Short: "Check the dependency status of a task",
	Long:  `Show which dependencies of a task are not yet done.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		s := store.New(projectRoot())

		t, err := s.Get(id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("task %s not found", id)
			}
			return err
		}

		if len(t.Dependencies) == 0 {
			fmt.Printf("Task %s has no dependencies.\n", id)
			return nil
		}

		allTasks, err := s.LoadAll()
		if err != nil {
			return err
		}

		taskByID := make(map[string]*task.Task, len(allTasks))
		for _, at := range allTasks {
			taskByID[at.ID] = at
		}

		var blocked []string
		for _, depID := range t.Dependencies {
			dep, ok := taskByID[depID]
			if !ok {
				fmt.Fprintf(os.Stderr, "warning: dependency %s not found\n", depID)
				blocked = append(blocked, depID+" (missing)")
				continue
			}
			if dep.Status != task.StatusDone {
				blocked = append(blocked, fmt.Sprintf("%s (%s) – %s", dep.ID, dep.Status, dep.Title))
			}
		}

		if len(blocked) == 0 {
			fmt.Printf("Task %s: all dependencies are done. ✓\n", id)
		} else {
			fmt.Printf("Task %s is blocked by:\n  %s\n", id, strings.Join(blocked, "\n  "))
		}
		return nil
	},
}

func init() {
	depsCmd.AddCommand(depsAddCmd)
	depsCmd.AddCommand(depsRemoveCmd)
	depsCmd.AddCommand(depsCheckCmd)
}
