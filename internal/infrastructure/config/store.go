package config

import (
	"encoding/json"
	"os"

	platconfig "cashflow_backend/internal/platform/config"
)

const defaultFilePerm = os.FileMode(0o600)

// Store persists and materializes a layered configuration. It mirrors
// Mattermost's config Store/BackingStore split without the diff tracking.
type Store interface {
	// LoadInto merges the stored JSON onto cfg. Only keys present in the
	// document override the pre-populated defaults.
	LoadInto(cfg *platconfig.Configuration) error
	// Save writes the configuration to durable storage.
	Save(cfg *platconfig.Configuration) error
}

// FileStore is a Store backed by a JSON file on disk.
type FileStore struct {
	path           string
	createIfMissing bool
}

// NewFileStore returns a FileStore for path. When createIfMissing is true and
// the file does not exist, it is created empty on the next Save.
func NewFileStore(path string, createIfMissing bool) (*FileStore, error) {
	return &FileStore{path: path, createIfMissing: createIfMissing}, nil
}

// Path returns the underlying file path.
func (s *FileStore) Path() string { return s.path }

// LoadInto reads the JSON file and merges present keys onto cfg. A missing file
// (when not required) leaves cfg untouched.
func (s *FileStore) LoadInto(cfg *platconfig.Configuration) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) && s.createIfMissing {
			return nil
		}
		return err
	}
	// Unmarshal onto the defaults-pre-populated struct so absent keys keep
	// their default values.
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, cfg)
}

// Save writes the configuration JSON to disk with 0600 permissions.
func (s *FileStore) Save(cfg *platconfig.Configuration) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, defaultFilePerm)
}

// MemoryStore keeps the configuration in memory (used for tests and runtime
// snapshots).
type MemoryStore struct {
	data []byte
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

func (s *MemoryStore) LoadInto(cfg *platconfig.Configuration) error {
	if len(s.data) == 0 {
		return nil
	}
	return json.Unmarshal(s.data, cfg)
}

func (s *MemoryStore) Save(cfg *platconfig.Configuration) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	s.data = data
	return nil
}