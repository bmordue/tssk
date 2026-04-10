package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/task"
)

var readyJSON bool

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List tasks that are ready to start",
	Long: `List all tasks with status 'todo' that do not depend on any task 
with status 'todo' or 'in-progress'.

A task is considered "ready" when it has no blocking dependencies, meaning
all its dependencies (if any) are either 'done' or 'blocked'.

Note: Tasks with 'blocked' dependencies are considered ready, as 'blocked'
indicates an external blocker rather than active work in progress.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := openStore()
		if err != nil {
			return err
		}

		tasks, err := st.LoadAll()
		if err != nil {
			return err
		}

		// Build a map for quick task lookup.
		taskByID := make(map[string]*task.Task, len(tasks))
		for _, t := range tasks {
			taskByID[t.ID] = t
		}

		// Collect ready tasks and track warnings.
		var readyTasks []*task.Task
		var warnings []string
		for _, t := range tasks {
			if t.Status != task.StatusTodo {
				continue
			}

			// Check if any dependency has status 'todo' or 'in-progress'.
			hasBlockingDep := false
			for _, depID := range t.Dependencies {
				dep, ok := taskByID[depID]
				if !ok {
					// Missing dependency - treat as blocking and warn.
					hasBlockingDep = true
					warnings = append(warnings, fmt.Sprintf("warning: task %s depends on non-existent task %s", t.ID, depID))
					break
				}
				if dep.Status == task.StatusTodo || dep.Status == task.StatusInProgress {
					hasBlockingDep = true
					break
				}
			}

			if !hasBlockingDep {
				readyTasks = append(readyTasks, t)
			}
		}

		// Emit warnings to stderr.
		for _, w := range warnings {
			fmt.Fprintln(cmd.ErrOrStderr(), w)
		}

		if readyJSON {
			if readyTasks == nil {
				readyTasks = []*task.Task{}
			}
			return printJSON(readyTasks)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tTITLE\tTAGS\tDEPS")
		for _, t := range readyTasks {
			tags := "-"
			if len(t.Tags) > 0 {
				tags = strings.Join(t.Tags, ", ")
			}
			deps := "-"
			if len(t.Dependencies) > 0 {
				deps = strings.Join(t.Dependencies, ", ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, t.Status, t.Title, tags, deps)
		}
		_ = w.Flush()
		return nil
	},
}

func init() {
	readyCmd.Flags().BoolVar(&readyJSON, "json", false, "Output tasks as JSON")
}
