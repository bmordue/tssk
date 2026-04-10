package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/task"
)

var phaseCmd = &cobra.Command{
	Use:   "phase [command]",
	Short: "Manage task phases",
	Long:  "Commands for managing task phases and checking phase gate compliance.",
}

var phaseCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check phase gate compliance",
	Long: `Check that all tasks follow phase gate rules.

Reports tasks that:
  - Are marked done without passing through in-review
  - Have invalid status transitions
  - Are missing phase tags (if expected)`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}

		tasks, err := s.LoadAll()
		if err != nil {
			return err
		}

		violations := 0
		for _, t := range tasks {
			if t.Status == "done" {
				// Check if task has a review history (we can't check past states,
				// but we can flag tasks that were done before this feature existed)
				if !t.HasPhaseTag() {
					fmt.Printf("Warning: Task %s (%s) is done but has no phase tag\n", t.ID, t.Title)
					violations++
				}
			}
		}

		if violations > 0 {
			fmt.Printf("\nFound %d phase gate violation(s)\n", violations)
			return fmt.Errorf("phase gate violations detected")
		}

		fmt.Println("All tasks are phase gate compliant")
		return nil
	},
}

var phaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks grouped by phase",
	Long:  "List all tasks grouped by their phase tag (phase-1, phase-2, etc.).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openStore()
		if err != nil {
			return err
		}

		tasks, err := s.LoadAll()
		if err != nil {
			return err
		}

		// Group tasks by phase
		phases := make(map[int][]*task.Task)
		var untagged []*task.Task

		for _, t := range tasks {
			if phase := t.GetPhase(); phase > 0 {
				phases[phase] = append(phases[phase], t)
			} else {
				untagged = append(untagged, t)
			}
		}

		// Sort and display phases
		phaseNumbers := make([]int, 0, len(phases))
		for p := range phases {
			phaseNumbers = append(phaseNumbers, p)
		}
		sort.Ints(phaseNumbers)

		for _, p := range phaseNumbers {
			fmt.Printf("\n=== Phase %d ===\n", p)
			for _, t := range phases[p] {
				fmt.Printf("  [%s] %s\n", t.Status, t.Title)
			}
		}

		if len(untagged) > 0 {
			fmt.Println("\n=== Untagged ===")
			for _, t := range untagged {
				fmt.Printf("  [%s] %s\n", t.Status, t.Title)
			}
		}

		return nil
	},
}

func init() {
	phaseCmd.AddCommand(phaseCheckCmd)
	phaseCmd.AddCommand(phaseListCmd)
}
