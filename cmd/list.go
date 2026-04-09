package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/task"
)

// collectedTaskOutput is the JSON representation of a task when listing across
// all collections.  It includes the source collection name, a fully-qualified
// task ID, and qualified dependency IDs so that output is unambiguous even when
// IDs collide across collections.
type collectedTaskOutput struct {
	Collection   string      `json:"collection"`
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Status       task.Status `json:"status"`
	Dependencies []string    `json:"dependencies,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	DocHash      string      `json:"doc_hash"`
}

var listStatus string
var listAllCollections bool
var listTag string
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
			return listAllCollectionsCmd(statusFilter, listJSON, listTag)
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
		fmt.Fprintln(w, "ID\tSTATUS\tTITLE\tTAGS\tDEPS")
		for _, t := range tasks {
			if statusFilter != nil && t.Status != *statusFilter {
				continue
			}
			if listTag != "" && !t.HasTag(listTag) {
				continue
			}
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
func listAllCollectionsCmd(statusFilter *task.Status, jsonOutput bool, tagFilter string) error {
	ms, err := openMultiStore()
	if err != nil {
		return err
	}
	collected, err := ms.LoadAll()
	if err != nil {
		return err
	}

	if jsonOutput {
		var filtered []collectedTaskOutput
		for _, ct := range collected {
			if statusFilter != nil && ct.Status != *statusFilter {
				continue
			}
			filtered = append(filtered, collectedTaskOutput{
				Collection:   ct.Collection,
				ID:           ct.QualifiedID(),
				Title:        ct.Title,
				Status:       ct.Status,
				Dependencies: qualifyDeps(ct.Dependencies, ct.Collection),
				CreatedAt:    ct.CreatedAt,
				DocHash:      ct.DocHash,
			})
		}
		if filtered == nil {
			filtered = []collectedTaskOutput{}
		}
		return printJSON(filtered)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COLLECTION\tID\tSTATUS\tTITLE\tTAGS\tDEPS")
	for _, ct := range collected {
		if statusFilter != nil && ct.Status != *statusFilter {
			continue
		}
		if tagFilter != "" && !ct.HasTag(tagFilter) {
			continue
		}
		collLabel := ct.Collection
		if collLabel == "" {
			collLabel = primaryCollectionLabel
		}
		tags := "-"
		if len(ct.Tags) > 0 {
			tags = strings.Join(ct.Tags, ", ")
		}
		deps := "-"
		if len(ct.Dependencies) > 0 {
			qualified := qualifyDeps(ct.Dependencies, ct.Collection)
			deps = strings.Join(qualified, ", ")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", collLabel, ct.ID, ct.Status, ct.Title, tags, deps)
	}
	_ = w.Flush()
	return nil
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (todo, in-progress, done, blocked)")
	listCmd.Flags().BoolVarP(&listAllCollections, "all-collections", "a", false, "Include tasks from all configured collections")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output tasks as JSON")
}
