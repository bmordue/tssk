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

var showJSON bool

// showTaskOutput is the JSON structure for tssk show --json
type showTaskOutput struct {
	task.Task
	Detail string `json:"detail,omitempty"`
}

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show full details of a task",
	Long:  `Show the metadata and full markdown detail text for a task.`,
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
		return nil
	},
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output task as JSON")
}
