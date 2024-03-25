package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alileza/bridge/httpredirector"
)

type JSONFileStorage struct {
	FilePath string

	routes *sync.Map
}

// NewJSONFileStorage creates a new instance of JSONFileStorage.
func NewJSONFileStorage(filePath string) (*JSONFileStorage, error) {
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

func (ls *JSONFileStorage) saveToFile() error {
	// Check if the file exists
	if _, err := os.Stat(ls.FilePath); err == nil {
		// File exists, read its contents
		existingData, err := os.ReadFile(ls.FilePath)
		if err != nil {
			return fmt.Errorf("storage: error reading file: %s", err)
		}

		// Decode existing data into a map
		var existingMap map[string]string
		if err := json.Unmarshal(existingData, &existingMap); err != nil {
			return fmt.Errorf("storage: error decoding existing file data: %s", err)
		}

		// Update existing data with new data
		ls.routes.Range(func(k, v any) bool {
			existingMap[k.(string)] = v.(string)
			return true
		})

		// Encode the updated data and write it back to the file
		newData, err := json.Marshal(existingMap)
		if err != nil {
			return fmt.Errorf("storage: error encoding updated data: %s", err)
		}

		if err := os.WriteFile(ls.FilePath, newData, 0644); err != nil {
			return fmt.Errorf("storage: error writing updated data to file: %s", err)
		}
		return nil
	}

	// If the file doesn't exist, create a new one and write the data to it
	f, err := os.Create(ls.FilePath)
	if err != nil {
		return fmt.Errorf("storage: error creating file: %s", err)
	}
	defer f.Close()

	tmp := make(map[string]string)
	ls.routes.Range(func(k, v any) bool {
		tmp[k.(string)] = v.(string)
		return true
	})

	if err := json.NewEncoder(f).Encode(tmp); err != nil {
		return fmt.Errorf("storage: error encoding file: %s", err)
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

	ss := strings.Split(filePath, "/")
	dir := strings.Join(ss[:len(ss)-1], "/")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(defaultContent), 0755); err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return f, nil
}
