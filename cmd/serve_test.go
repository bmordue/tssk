package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()
	tmpDir := t.TempDir()

	// Create a minimal config
	cfg := store.Config{
		Backend:           "local",
		TasksFile:         ".tsks/tasks.jsonl",
		DocsDir:           ".tsks/docs",
		DisplayHashLength: 9,
	}
	cfgPath := filepath.Join(tmpDir, ".tssk.json")
	cfgData, _ := json.Marshal(cfg)
	if err := os.WriteFile(cfgPath, cfgData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create store
	st := store.New(tmpDir)

	// Add some test tasks
	tasks := []struct {
		title  string
		detail string
		status task.Status
		tags   []string
		deps   []string
	}{
		{"Task 1", "Detail for task 1", task.StatusTodo, []string{"bug"}, nil},
		{"Task 2", "Detail for task 2", task.StatusInProgress, []string{"feature"}, nil},
		{"Task 3", "Detail for task 3", task.StatusDone, nil, []string{"1"}},
		{"Task 4", "", task.StatusBlocked, []string{"bug", "critical"}, []string{"2"}},
	}

	for _, tt := range tasks {
		tsk, err := st.Add(tt.title, tt.detail, tt.deps, tt.tags)
		if err != nil {
			t.Fatal(err)
		}
		// Update status if not todo
		if tt.status != task.StatusTodo {
			_, err = st.UpdateStatus(tsk.ID, tt.status)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	return st
}

func TestHandleListTasks(t *testing.T) {
	st := setupTestStore(t)

	w := httptest.NewRecorder()

	handleListTasks(w, st)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", contentType)
	}

	var tasks []task.Task
	if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(tasks))
	}
}

func TestHandleGetTask(t *testing.T) {
	st := setupTestStore(t)

	tests := []struct {
		name           string
		taskID         string
		expectedStatus int
	}{
		{
			name:           "existing task",
			taskID:         "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "nonexistent task",
			taskID:         "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			handleGetTask(w, st, tt.taskID)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var taskOut taskDetailOutput
				if err := json.NewDecoder(w.Body).Decode(&taskOut); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if taskOut.ID != tt.taskID {
					t.Errorf("expected task ID %q, got %q", tt.taskID, taskOut.ID)
				}
			}
		})
	}
}

func TestHandleUpdateStatus(t *testing.T) {
	st := setupTestStore(t)

	tests := []struct {
		name           string
		taskID         string
		newStatus      string
		expectedStatus int
	}{
		{
			name:           "valid status update",
			taskID:         "1",
			newStatus:      "in-progress",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid status",
			taskID:         "1",
			newStatus:      "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "nonexistent task",
			taskID:         "999",
			newStatus:      "done",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := statusUpdateRequest{Status: tt.newStatus}
			bodyData, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/api/tasks/"+tt.taskID+"/status",
				bytes.NewReader(bodyData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleUpdateStatus(w, req, st, tt.taskID)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
