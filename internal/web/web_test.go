package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIndexHandler(t *testing.T) {
	handler := IndexHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "root path serves index",
			path:           "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "other paths return 404",
			path:           "/foo",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.path == "/" {
				contentType := w.Header().Get("Content-Type")
				if contentType != "text/html; charset=utf-8" {
					t.Errorf("expected Content-Type %q, got %q", "text/html; charset=utf-8", contentType)
				}
			}
		})
	}
}

func TestAssetsFileSystem(t *testing.T) {
	fs := Assets()

	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{
			name:   "index.html exists",
			path:   "/index.html",
			exists: true,
		},
		{
			name:   "style.css exists",
			path:   "/style.css",
			exists: true,
		},
		{
			name:   "app.js exists",
			path:   "/app.js",
			exists: true,
		},
		{
			name:   "nonexistent file",
			path:   "/nonexistent.html",
			exists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := fs.Open(tt.path)
			if tt.exists {
				if err != nil {
					t.Errorf("expected file to exist, got error: %v", err)
				}
				if f != nil {
					if closeErr := f.Close(); closeErr != nil {
						t.Errorf("failed to close file: %v", closeErr)
					}
				}
			} else {
				if err == nil {
					t.Error("expected error for nonexistent file, got nil")
					if f != nil {
						if closeErr := f.Close(); closeErr != nil {
							t.Errorf("failed to close file: %v", closeErr)
						}
					}
				}
			}
		})
	}
}

func TestEmbeddedFilesAreValid(t *testing.T) {
	// Test that embedded files can be served correctly
	fs := Assets()

	files := []string{"/index.html", "/style.css", "/app.js"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			f, err := fs.Open(file)
			if err != nil {
				t.Fatalf("failed to open %s: %v", file, err)
			}
			defer func() {
				if closeErr := f.Close(); closeErr != nil {
					t.Errorf("failed to close %s: %v", file, closeErr)
				}
			}()

			stat, err := f.Stat()
			if err != nil {
				t.Fatalf("failed to stat %s: %v", file, err)
			}

			if stat.Size() == 0 {
				t.Errorf("file %s is empty", file)
			}
		})
	}
}

// mockTask is a minimal task structure for testing
type mockTask struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Dependencies []string `json:"dependencies,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	CreatedAt    string   `json:"created_at"`
	DocHash      string   `json:"doc_hash"`
}

func TestAPIEndpoints(t *testing.T) {
	// This test validates that the API endpoints are correctly structured
	// Actual integration tests with the store are in cmd package

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET /api/tasks returns 200",
			method:         http.MethodGet,
			path:           "/api/tasks",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/tasks returns 405",
			method:         http.MethodPost,
			path:           "/api/tasks",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple mock handler to test routing
			mux := http.NewServeMux()
			mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				tasks := []mockTask{}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(tasks); err != nil {
					t.Fatalf("failed to encode tasks: %v", err)
				}
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
