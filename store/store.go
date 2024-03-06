package store

import "os"

type LocalStore struct {
	Filepath string
}

func (s *LocalStore) WriteFile(data []byte) error {
	return os.WriteFile(s.Filepath, data, 0644)
}

func (s *LocalStore) ReadFile() ([]byte, error) {
	return os.ReadFile(s.Filepath)
}
