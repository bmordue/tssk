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

var showAllCollections bool
var showJSON bool

// showTaskOutput is the JSON structure for tssk show --json
type showTaskOutput struct {
	task.Task
	Detail string `json:"detail,omitempty"`
}

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show full details of a task",
	Long: `Show the metadata and full markdown detail text for a task.

When --all-collections is set, task-id can be qualified as "{collection}:{id}"
to show tasks from any configured collection. Unqualified IDs resolve against
the primary store.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if showAllCollections {
			return showFromMultiStore(id)
		}

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

		if showJSON {
			output := showTaskOutput{
				Task: *t,
			}
			detail, readErr := s.ReadDetail(t)
			if readErr != nil {
				// Detail file missing is non-fatal; surface a warning.
				fmt.Fprintf(os.Stderr, "warning: could not read detail file: %v\n", readErr)
			} else if detail != "" {
				output.Detail = detail
			}
			return printJSON(output)
		}

		printTask(s, t)
		return nil
	},
}

// showFromMultiStore resolves a possibly-qualified task ID across all
// configured collections and displays its details.
func showFromMultiStore(qualifiedID string) error {
	ms, err := openMultiStore()
	if err != nil {
		return err
	}

	ct, err := ms.Get(qualifiedID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("task %s not found", qualifiedID)
		}
		return err
	}

	cfg, err := store.ConfigFromFileAndEnv(projectRoot())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// ct.Collection is "" for an unnamed primary store, or cfg.Name for a
	// named primary store.  In both cases we use the primary store directly.
	if ct.Collection == "" || ct.Collection == cfg.Name {
		s, err := openStore()
		if err != nil {
			return err
		}
		if showJSON {
			return printTaskJSON(s, ct.Task)
		}
		printTask(s, ct.Task)
		return nil
	}

	// Named secondary collection - open its store directly.
	for _, cc := range cfg.Collections {
		if cc.Name == ct.Collection {
			collStore, err := store.CollectionStoreFromConfig(cc)
			if err != nil {
				return fmt.Errorf("opening collection store: %w", err)
			}
			if showJSON {
				return printTaskJSON(collStore, ct.Task)
			}
			printTask(collStore, ct.Task)
			return nil
		}
	}
	return fmt.Errorf("collection %q not found in configuration", ct.Collection)
}

// printTaskJSON emits a task as JSON, including its detail content.
func printTaskJSON(s *store.Store, t *task.Task) error {
	output := showTaskOutput{
		Task: *t,
	}
	detail, readErr := s.ReadDetail(t)
	if readErr != nil {
		// Detail file missing is non-fatal; surface a warning.
		fmt.Fprintf(os.Stderr, "warning: could not read detail file: %v\n", readErr)
	} else if detail != "" {
		output.Detail = detail
	}
	return printJSON(output)
}

// printTask displays a task's metadata and detail content.
func printTask(s *store.Store, t *task.Task) {
	fmt.Fprintf(os.Stdout, "ID:         %s\n", t.ID)
	fmt.Fprintf(os.Stdout, "Title:      %s\n", t.Title)
	fmt.Fprintf(os.Stdout, "Status:     %s\n", t.Status)
	fmt.Fprintf(os.Stdout, "Created:    %s\n", t.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	if len(t.Dependencies) > 0 {
		fmt.Fprintf(os.Stdout, "Depends on: %s\n", strings.Join(t.Dependencies, ", "))
	}
	fmt.Fprintf(os.Stdout, "Doc hash:   %s\n", t.DocHash)

	detail, err := s.ReadDetail(t)
	if err != nil {
		// Detail file missing is non-fatal; surface a warning.
		fmt.Fprintf(os.Stderr, "warning: could not read detail file: %v\n", err)
	} else if detail != "" {
		fmt.Fprintf(os.Stdout, "\n%s\n", detail)
	}
}

func init() {
	showCmd.Flags().BoolVarP(&showAllCollections, "all-collections", "a", false, "Resolve task ID across all configured collections")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output task as JSON")
}
