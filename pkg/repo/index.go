package repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// IndexEntry represents an entry in the index file
type IndexEntry struct {
	Path    string    `json:"path"`
	ObjID   string    `json:"obj_id"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// Index represents the repository index
type Index struct {
	Entries map[string]IndexEntry `json:"entries"`
}

// SaveIndex saves the index to a file
func (r *Repository) SaveIndex() error {
	// Create index from staging area
	index := Index{
		Entries: make(map[string]IndexEntry),
	}

	// Add entries from staging area
	for path, objID := range r.State.Stage {
		entry, ok := r.State.WorkTree[path]
		if !ok {
			continue
		}

		index.Entries[path] = IndexEntry{
			Path:    path,
			ObjID:   objID,
			Size:    entry.Size,
			ModTime: entry.ModTime,
		}
	}

	// Marshal index to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write index to file
	indexPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitIndexFile)
	if err := ioutil.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// LoadIndex loads the index from a file
func (r *Repository) LoadIndex() error {
	// Get index file path
	indexPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitIndexFile)

	// Check if index file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Index file doesn't exist, nothing to load
		return nil
	}

	// Read index file
	data, err := ioutil.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	// If the file is empty, return early
	if len(data) == 0 {
		return nil
	}

	// Unmarshal index
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	// Update staging area and working tree
	for _, entry := range index.Entries {
		r.State.Stage[entry.Path] = entry.ObjID
		r.State.WorkTree[entry.Path] = WorkTreeEntry{
			Path:    entry.Path,
			Size:    entry.Size,
			ModTime: entry.ModTime,
			Hash:    entry.ObjID,
		}
	}

	return nil
}
