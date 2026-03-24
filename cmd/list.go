package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

var listStatus string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long:  `List all tasks, optionally filtered by status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var statusFilter *task.Status
		if listStatus != "" {
			s := task.Status(listStatus)
			if !s.IsValid() {
				return fmt.Errorf("unknown status %q; valid values: todo, in-progress, done, blocked", listStatus)
			}
			statusFilter = &s
		}

		st := store.New(projectRoot())
		tasks, err := st.LoadAll()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tTITLE\tDEPS")
		for _, t := range tasks {
			if statusFilter != nil && t.Status != *statusFilter {
				continue
			}
			deps := "-"
			if len(t.Dependencies) > 0 {
				deps = strings.Join(t.Dependencies, ", ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Status, t.Title, deps)
		}
		_ = w.Flush()
		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (todo, in-progress, done, blocked)")
}
