package storage

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"bridge/httpredirector"
)

type ShardFileStorage struct {
	Directory string
}

// NewShardFileStorage creates a new instance of ShardFileStorage.
func NewShardFileStorage(directory string) (*ShardFileStorage, error) {
	// Ensure the directory exists or try to create it.
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &ShardFileStorage{Directory: directory}, nil
}

// Set writes the URL content to a file named after the key.
func (fs *ShardFileStorage) Set(key string, value string) error {
	encodedKey := url.QueryEscape(key)
	filePath := filepath.Join(fs.Directory, encodedKey)

	// Open the file with flags to read and write, create if not exists, and truncate to update the value if it does.
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(value)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// Get reads the URL content from a file named after the key.
func (fs *ShardFileStorage) Get(key string) (string, error) {
	filePath := filepath.Join(fs.Directory, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("key not found")
		}
		return "", err
	}
	return string(data), nil
}

// Delete removes the file named after the key.
func (fs *ShardFileStorage) Delete(key string) error {
	filePath := filepath.Join(fs.Directory, key)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// List returns a list of Routes, each representing a file in the directory.
func (fs *ShardFileStorage) List() ([]httpredirector.Route, error) {
	var routes []httpredirector.Route

	err := filepath.Walk(fs.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil // Skip directories
		}
		relativePath, err := filepath.Rel(fs.Directory, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		routes = append(routes, httpredirector.Route{Key: relativePath, URL: string(data)})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error listing storage directory: %w", err)
	}

	return routes, nil
}

func (fs *ShardFileStorage) Reload() error {
	return nil
}
