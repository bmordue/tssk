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
var listJSON bool

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
			return listAllCollectionsCmd(statusFilter, listJSON)
		}

		st, err := openStore()
		if err != nil {
			return err
		}
		tasks, err := st.LoadAll()
		if err != nil {
			return err
		}

		if listJSON {
			var filtered []task.Task
			for _, t := range tasks {
				if statusFilter != nil && t.Status != *statusFilter {
					continue
				}
				filtered = append(filtered, *t)
			}
			if filtered == nil {
				filtered = []task.Task{}
			}
			return printJSON(filtered)
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

// primaryCollectionLabel is the display label used for the primary (unnamed)
// collection when listing tasks across all collections.
const primaryCollectionLabel = "(primary)"

// qualifyDeps returns the dependency list qualified with the given collection
// name.  Dependencies that already contain ":" (already qualified) are left
// unchanged; bare IDs are prefixed with "{collection}:".  An empty collection
// means the primary (unnamed) store – those are left unqualified.
func qualifyDeps(deps []string, collection string) []string {
	if len(deps) == 0 || collection == "" {
		return deps
	}
	out := make([]string, len(deps))
	for i, d := range deps {
		if strings.Contains(d, ":") {
			out[i] = d
		} else {
			out[i] = collection + ":" + d
		}
	}
	return out
}

// listAllCollectionsCmd handles --all-collections: loads tasks from every
// configured collection and prints them with a COLLECTION column.
func listAllCollectionsCmd(statusFilter *task.Status, jsonOutput bool) error {
	ms, err := openMultiStore()
	if err != nil {
		return err
	}
	collected, err := ms.LoadAll()
	if err != nil {
		return err
	}

	if jsonOutput {
		var filtered []task.Task
		for _, ct := range collected {
			if statusFilter != nil && ct.Status != *statusFilter {
				continue
			}
			filtered = append(filtered, *ct.Task)
		}
		if filtered == nil {
			filtered = []task.Task{}
		}
		return printJSON(filtered)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COLLECTION\tID\tSTATUS\tTITLE\tDEPS")
	for _, ct := range collected {
		if statusFilter != nil && ct.Status != *statusFilter {
			continue
		}
		collLabel := ct.Collection
		if collLabel == "" {
			collLabel = primaryCollectionLabel
		}
		deps := "-"
		if len(ct.Dependencies) > 0 {
			qualified := qualifyDeps(ct.Dependencies, ct.Collection)
			deps = strings.Join(qualified, ", ")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", collLabel, ct.ID, ct.Status, ct.Title, deps)
	}
	_ = w.Flush()
	return nil
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (todo, in-progress, done, blocked)")
	listCmd.Flags().BoolVarP(&listAllCollections, "all-collections", "a", false, "Include tasks from all configured collections")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output tasks as JSON")
}
