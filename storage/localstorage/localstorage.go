package localstorage

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type LocalStorage struct {
	FilePath string

	routes sync.Map
}

func NewLocalStorage(filePath string) *LocalStorage {
	tmp := make(map[string]string)

	f, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("localstorage: error reading file: %s", err)
		log.Printf("localstorage: creating new file: %s", filePath)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			log.Fatalf("localstorage: error creating file: %s", err)
		}

	} else {
		if err := json.Unmarshal(f, &tmp); err != nil {
			log.Printf("localstorage: error decoding file: %s", err)
			log.Printf("localstorage: creating new file: %s", filePath)
			if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
				log.Fatalf("localstorage: error creating file: %s", err)
			}
		}
	}

	ls := &LocalStorage{
		FilePath: filePath,
	}

	for k, v := range tmp {
		ls.routes.Store(k, v)
	}

	return ls
}

func (ls *LocalStorage) Store(key any, value any) {
	ls.routes.Store(key, value)
	ls.saveToFile()
}

func (ls *LocalStorage) Load(key any) (value any, ok bool) {
	return ls.routes.Load(key)
}

func (ls *LocalStorage) Delete(key any) {
	ls.routes.Delete(key)
	ls.saveToFile()
}

func (ls *LocalStorage) Range(f func(key any, value any) bool) {
	ls.routes.Range(f)
}

func (ls *LocalStorage) saveToFile() {
	// Check if the file exists
	if _, err := os.Stat(ls.FilePath); err == nil {
		// File exists, read its contents
		existingData, err := os.ReadFile(ls.FilePath)
		if err != nil {
			log.Printf("localstorage: error reading file: %s", err)
			return
		}

		// Decode existing data into a map
		var existingMap map[string]string
		if err := json.Unmarshal(existingData, &existingMap); err != nil {
			log.Printf("localstorage: error decoding existing file data: %s", err)
			return
		}

		// Update existing data with new data
		ls.routes.Range(func(k, v any) bool {
			existingMap[k.(string)] = v.(string)
			return true
		})

		// Encode the updated data and write it back to the file
		newData, err := json.Marshal(existingMap)
		if err != nil {
			log.Printf("localstorage: error encoding updated data: %s", err)
			return
		}

		if err := os.WriteFile(ls.FilePath, newData, 0644); err != nil {
			log.Printf("localstorage: error writing updated data to file: %s", err)
		}
		return
	}

	// If the file doesn't exist, create a new one and write the data to it
	f, err := os.Create(ls.FilePath)
	if err != nil {
		log.Printf("localstorage: error creating file: %s", err)
		return
	}
	defer f.Close()

	tmp := make(map[string]string)
	ls.routes.Range(func(k, v any) bool {
		tmp[k.(string)] = v.(string)
		return true
	})

	if err := json.NewEncoder(f).Encode(tmp); err != nil {
		log.Printf("localstorage: error encoding file: %s", err)
	}
}
