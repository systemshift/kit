package repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// SaveIndex saves the repository state to the index file
func (r *Repository) SaveIndex() error {
	// Check for nil state
	if r.State == nil {
		return fmt.Errorf("repository state is nil")
	}

	// Create index data structure
	index := struct {
		Stage    map[string]string        `json:"stage"`
		Tracked  map[string]string        `json:"tracked"`
		WorkTree map[string]WorkTreeEntry `json:"worktree"`
		HEAD     string                   `json:"head"`
	}{
		Stage:    r.State.Stage,
		Tracked:  r.State.Tracked,
		WorkTree: r.State.WorkTree,
		HEAD:     r.State.HEAD,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	indexPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitIndexFile)
	if err := ioutil.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// LoadIndex loads the repository state from the index file
func (r *Repository) LoadIndex() error {
	// Check if index file exists
	indexPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitIndexFile)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// No index file, initialize empty state
		r.State = &RepositoryState{
			HEAD:     "refs/heads/main",
			Stage:    make(map[string]string),
			Tracked:  make(map[string]string),
			WorkTree: make(map[string]WorkTreeEntry),
		}
		return nil
	}

	// Read index file
	data, err := ioutil.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	// Skip if the file is empty
	if len(data) == 0 {
		r.State = &RepositoryState{
			HEAD:     "refs/heads/main",
			Stage:    make(map[string]string),
			Tracked:  make(map[string]string),
			WorkTree: make(map[string]WorkTreeEntry),
		}
		return nil
	}

	// Unmarshal JSON
	var index struct {
		Stage    map[string]string        `json:"stage"`
		Tracked  map[string]string        `json:"tracked"`
		WorkTree map[string]WorkTreeEntry `json:"worktree"`
		HEAD     string                   `json:"head"`
	}

	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	// Update repository state
	r.State.Stage = index.Stage
	r.State.Tracked = index.Tracked
	r.State.WorkTree = index.WorkTree

	// Only update HEAD if it exists in the index
	if index.HEAD != "" {
		r.State.HEAD = index.HEAD
	} else {
		// Try to read HEAD from file
		headPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitHeadFile)
		if headData, err := ioutil.ReadFile(headPath); err == nil {
			content := string(headData)
			if len(content) > 5 && content[:4] == "ref:" {
				r.State.HEAD = content[5 : len(content)-1] // Remove "ref: " and trailing newline
			}
		}
	}

	return nil
}
