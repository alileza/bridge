package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// TestCreateOrLoad tests the createOrLoad function with various input scenarios.
func TestCreateOrLoad(t *testing.T) {
	// Define test cases
	tests := []struct {
		name         string
		filePath     string
		wantContent  []byte
		wantErr      bool
		setupFunc    func() string    // Optional setup function to prepare the test environment
		teardownFunc func(dir string) // Corrected to accept a directory path
	}{
		{
			name:        "File does not exist",
			filePath:    "nonexistent.json",
			wantContent: []byte("{}"),
			wantErr:     false,
			setupFunc: func() string {
				dir, _ := os.MkdirTemp("", "test")
				return dir
			},
			teardownFunc: func(dir string) {
				os.RemoveAll(dir)
			},
		},
		{
			name:        "Path is a directory",
			filePath:    "a_directory",
			wantContent: nil, // Adjusted expected behavior based on function implementation details
			wantErr:     true,
			setupFunc: func() string {
				dir, _ := os.MkdirTemp("", "test")
				os.Mkdir(filepath.Join(dir, "a_directory.json"), 0755) // Simulate directory with .json extension
				return dir
			},
			teardownFunc: func(dir string) {
				os.RemoveAll(dir)
			},
		},
		{
			name:        "Path is already a .json file",
			filePath:    "already_json.json",
			wantContent: []byte("{}"),
			wantErr:     false,
			setupFunc: func() string {
				dir, _ := os.MkdirTemp("", "test")
				os.WriteFile(filepath.Join(dir, "already_json.json"), []byte("{}"), 0755)
				return dir
			},
			teardownFunc: func(dir string) {
				os.RemoveAll(dir)
			},
		},
		{
			name:        "File exists but content is invalid/corrupted",
			filePath:    "invalid_content.json",
			wantContent: nil,  // Assuming function should return an error or nil content for invalid content
			wantErr:     true, // Assuming the presence of an error for invalid content
			setupFunc: func() string {
				dir, _ := os.MkdirTemp("", "test")
				os.WriteFile(filepath.Join(dir, "invalid_content.json"), []byte("not valid json"), 0755)
				return dir
			},
			teardownFunc: func(dir string) {
				os.RemoveAll(dir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanupDir string
			// Setup if needed
			if tt.setupFunc != nil {
				cleanupDir = tt.setupFunc()
				tt.filePath = filepath.Join(cleanupDir, tt.filePath)
			}

			gotContent, err := createOrLoad(tt.filePath)

			// Verify error handling
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: createOrLoad() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}

			// Verify function output
			if !reflect.DeepEqual(gotContent, tt.wantContent) {
				t.Errorf("%s: createOrLoad() = %v, want %v", tt.name, gotContent, tt.wantContent)
			}

			// Teardown if needed
			if tt.teardownFunc != nil && cleanupDir != "" {
				tt.teardownFunc(cleanupDir)
			}
		})
	}
}

func TestJSONFileStoragePersistsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes")

	s, err := NewJSONFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Set("go/keep", "https://keep")
	s.Set("go/update", "https://v1")
	s.Set("go/update", "https://v2")
	s.Set("go/delete", "https://gone")
	s.Delete("go/delete")

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Set(fmt.Sprintf("go/concurrent-%d", i), "https://c")
		}()
	}
	wg.Wait()

	// A fresh instance only sees what reached the file.
	reloaded, err := NewJSONFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := reloaded.Get("go/keep"); v != "https://keep" {
		t.Errorf("go/keep = %q", v)
	}
	if v, _ := reloaded.Get("go/update"); v != "https://v2" {
		t.Errorf("go/update = %q, want https://v2", v)
	}
	if _, err := reloaded.Get("go/delete"); err == nil {
		t.Error("deleted route came back after reload")
	}
	routes, _ := reloaded.List()
	if len(routes) != 22 {
		t.Errorf("reloaded %d routes, want 22", len(routes))
	}

	matches, _ := filepath.Glob(path + ".json.tmp-*")
	if len(matches) != 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}
