package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alileza/bridge/httpredirector"
)

type JSONFileStorage struct {
	FilePath string

	routes *sync.Map
	saveMu sync.Mutex // serialises snapshot+write so an older snapshot never wins
}

// NewJSONFileStorage creates a new instance of JSONFileStorage.
func NewJSONFileStorage(filePath string) (*JSONFileStorage, error) {
	if !strings.HasSuffix(filePath, ".json") {
		filePath = filePath + ".json"
	}

	f, err := createOrLoad(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var routes map[string]string
	if err := json.Unmarshal(f, &routes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file data: %w", err)
	}

	cm, err := toSyncMap(f)
	if err != nil {
		return nil, fmt.Errorf("failed to convert map to sync.Map: %w", err)
	}

	return &JSONFileStorage{
		FilePath: filePath,

		routes: cm,
	}, nil
}

// Set writes the routes to a JSON file.
func (fs *JSONFileStorage) Set(key string, value string) error {
	fs.routes.Store(key, value)
	return fs.saveToFile()
}

// List returns all the routes from the JSON file.
func (fs *JSONFileStorage) List() ([]httpredirector.Route, error) {
	var routes []httpredirector.Route
	fs.routes.Range(func(k, v any) bool {
		routes = append(routes, httpredirector.Route{Key: k.(string), URL: v.(string)})
		return true
	})
	return routes, nil
}

// Get reads the routes from the JSON file.
func (fs *JSONFileStorage) Get(key string) (string, error) {
	value, ok := fs.routes.Load(key)
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return value.(string), nil
}

// Delete removes the JSON file.
func (fs *JSONFileStorage) Delete(key string) error {
	fs.routes.Delete(key)
	return fs.saveToFile()
}

// Get reads the routes from the JSON file.
func (fs *JSONFileStorage) Reload() error {
	f, err := os.ReadFile(fs.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	cm, err := toSyncMap(f)
	if err != nil {
		return fmt.Errorf("failed to convert map to sync.Map: %w", err)
	}

	fs.routes = cm
	return nil
}

// saveToFile writes all in-memory routes to the file, replacing its contents.
// It writes to a temp file and renames it, so a crash never leaves a partial file.
func (ls *JSONFileStorage) saveToFile() error {
	ls.saveMu.Lock()
	defer ls.saveMu.Unlock()

	routes := make(map[string]string)
	ls.routes.Range(func(k, v any) bool {
		routes[k.(string)] = v.(string)
		return true
	})

	data, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("storage: error encoding routes: %s", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(ls.FilePath), filepath.Base(ls.FilePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("storage: error creating temp file: %s", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("storage: error writing routes: %s", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: error writing routes: %s", err)
	}
	if err := os.Rename(tmp.Name(), ls.FilePath); err != nil {
		return fmt.Errorf("storage: error replacing routes file: %s", err)
	}
	return nil
}

func toSyncMap(b []byte) (*sync.Map, error) {
	var routes map[string]string
	if err := json.Unmarshal(b, &routes); err != nil {
		return &sync.Map{}, fmt.Errorf("failed to unmarshal file data: %w", err)
	}

	sm := &sync.Map{}
	for k, v := range routes {
		sm.Store(k, v)
	}

	return sm, nil
}

func createOrLoad(filePath string) ([]byte, error) {
	const defaultContent = "{}"
	if !strings.HasSuffix(filePath, ".json") {
		filePath = filePath + ".json"
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.WriteFile(filePath, []byte(defaultContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if !json.Valid(f) {
		return nil, fmt.Errorf("invalid JSON in %s", filePath)
	}

	return f, nil
}
