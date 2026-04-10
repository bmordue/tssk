package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
	"github.com/bmordue/tssk/internal/web"
)

var (
	servePort int
	serveHost string
	serveOpen bool
)

// validTaskID matches task IDs that consist only of alphanumeric characters
// and hyphens, with hyphens only appearing between alphanumeric characters.
var validTaskID = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// taskDetailOutput is the JSON structure for a task with its detail text
type taskDetailOutput struct {
	task.Task
	Detail string `json:"detail,omitempty"`
}

// statusUpdateRequest is the expected body for status updates
type statusUpdateRequest struct {
	Status string `json:"status"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a web server to visualize tasks",
	Long:  `Start a local HTTP server that provides a web UI for visualizing and managing tasks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := openStore()
		if err != nil {
			return err
		}

		var mu sync.Mutex
		mux := http.NewServeMux()

		// Serve static assets
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(web.Assets())))

		// Serve index.html at root
		mux.Handle("/", web.IndexHandler())

		// API: List all tasks
		mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			handleListTasks(w, st)
		})

		// API: Get single task or update status
		mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
			// Extract task ID from path: /api/tasks/{id} or /api/tasks/{id}/status
			path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")

			// Check if this is a status update
			if strings.HasSuffix(path, "/status") {
				taskID := strings.TrimSuffix(path, "/status")
				if !validTaskID.MatchString(taskID) {
					http.Error(w, "Invalid task ID", http.StatusBadRequest)
					return
				}
				if r.Method == http.MethodPost {
					mu.Lock()
					defer mu.Unlock()
					handleUpdateStatus(w, r, st, taskID)
					return
				}
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Single task retrieval
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if !validTaskID.MatchString(path) {
				http.Error(w, "Invalid task ID", http.StatusBadRequest)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			handleGetTask(w, st, path)
		})

		addr := fmt.Sprintf("%s:%d", serveHost, servePort)
		fmt.Printf("🚀 tssk web UI available at http://%s\n", addr)

		if serveOpen {
			go func() {
				url := fmt.Sprintf("http://%s", addr)
				var openCmd *exec.Cmd
				switch runtime.GOOS {
				case "linux":
					openCmd = exec.Command("xdg-open", url)
				case "darwin":
					openCmd = exec.Command("open", url)
				case "windows":
					openCmd = exec.Command("cmd", "/c", "start", url)
				default:
					return
				}
				openCmd.Stdout = nil
				openCmd.Stderr = nil
				_ = openCmd.Run()
			}()
		}

		fmt.Printf("Press Ctrl+C to stop the server\n")
		return http.ListenAndServe(addr, mux)
	},
}

func handleListTasks(w http.ResponseWriter, st *store.Store) {
	tasks, err := st.LoadAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load tasks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode tasks: %v", err), http.StatusInternalServerError)
		return
	}
}

func handleGetTask(w http.ResponseWriter, st *store.Store, taskID string) {
	t, err := st.Get(taskID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get task: %v", err), http.StatusInternalServerError)
		return
	}

	output := taskDetailOutput{
		Task: *t,
	}

	detail, err := st.ReadDetail(t)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Failed to read detail: %v", err), http.StatusInternalServerError)
			return
		}
		// ErrNotFound is expected when a task has no detail file.
	} else if detail != "" {
		output.Detail = detail
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(output); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode task: %v", err), http.StatusInternalServerError)
		return
	}
}

func handleUpdateStatus(w http.ResponseWriter, r *http.Request, st *store.Store, taskID string) {
	var req statusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	status := task.Status(req.Status)
	if !status.IsValid() {
		http.Error(w, fmt.Sprintf("Invalid status: %s", req.Status), http.StatusBadRequest)
		return
	}

	_, err := st.UpdateStatus(taskID, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringVarP(&serveHost, "host", "", "localhost", "Host to listen on")
	serveCmd.Flags().BoolVarP(&serveOpen, "open", "o", false, "Open browser automatically")
}
