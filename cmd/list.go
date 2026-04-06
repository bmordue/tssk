package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/task"
)

var listStatus string
var listAllCollections bool

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

		if listAllCollections {
			return listAllCollectionsCmd(statusFilter)
		}

		st, err := openStore()
		if err != nil {
			return err
		}
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

// listAllCollectionsCmd handles --all-collections: loads tasks from every
// configured collection and prints them with a COLLECTION column.
func listAllCollectionsCmd(statusFilter *task.Status) error {
	ms, err := openMultiStore()
	if err != nil {
		return err
	}
	collected, err := ms.LoadAll()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COLLECTION\tID\tSTATUS\tTITLE\tDEPS")
	for _, ct := range collected {
		if statusFilter != nil && ct.Status != *statusFilter {
			continue
		}
		collLabel := ct.Collection
		if collLabel == "" {
			collLabel = "(primary)"
		}
		deps := "-"
		if len(ct.Dependencies) > 0 {
			deps = strings.Join(ct.Dependencies, ", ")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", collLabel, ct.ID, ct.Status, ct.Title, deps)
	}
	_ = w.Flush()
	return nil
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (todo, in-progress, done, blocked)")
	listCmd.Flags().BoolVarP(&listAllCollections, "all-collections", "a", false, "Include tasks from all configured collections")
}
