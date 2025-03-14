package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CommitObject represents a commit in the repository
type CommitObject struct {
	Tree      string    `json:"tree"`      // Tree object ID
	Parent    string    `json:"parent"`    // Parent commit ID (empty for first commit)
	Parent2   string    `json:"parent2"`   // Second parent commit ID (for merge commits)
	Author    string    `json:"author"`    // Author name and email
	Committer string    `json:"committer"` // Committer name and email
	Message   string    `json:"message"`   // Commit message
	Timestamp time.Time `json:"timestamp"` // Commit timestamp
}

// TreeObject represents a tree in the repository (directory structure)
type TreeObject struct {
	Entries map[string]TreeEntry `json:"entries"` // Map of path to entry
}

// TreeEntry represents an entry in a tree
type TreeEntry struct {
	Path  string `json:"path"`   // File path
	Mode  string `json:"mode"`   // File mode (100644 for files, 040000 for directories)
	Type  string `json:"type"`   // Object type (blob or tree)
	ObjID string `json:"obj_id"` // Object ID
}

// Commit creates a new commit from the staging area
func (r *Repository) Commit(message string) (string, error) {
	// Check if there's anything to commit
	if len(r.State.Stage) == 0 {
		return "", fmt.Errorf("nothing to commit, working tree clean")
	}

	// Create a tree object from the staging area
	tree := TreeObject{
		Entries: make(map[string]TreeEntry),
	}

	for path, objID := range r.State.Stage {
		// For now, all objects are blob files
		tree.Entries[path] = TreeEntry{
			Path:  path,
			Mode:  "100644", // Regular file
			Type:  "blob",
			ObjID: objID,
		}
	}

	// Serialize tree object
	treeData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tree: %w", err)
	}

	// Compute tree hash
	treeHash := sha256.Sum256(treeData)
	treeID := hex.EncodeToString(treeHash[:])

	// Store tree object
	err = r.storeObject(treeID, treeData)
	if err != nil {
		return "", fmt.Errorf("failed to store tree: %w", err)
	}

	// Get parent commit ID
	parentID, err := r.resolveReference(r.State.HEAD)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	// Create commit object
	commit := CommitObject{
		Tree:      treeID,
		Parent:    parentID,
		Author:    "Kit User <kit@example.com>", // Hardcoded for now
		Committer: "Kit User <kit@example.com>", // Hardcoded for now
		Message:   message,
		Timestamp: time.Now(),
	}

	// Serialize commit object
	commitData, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal commit: %w", err)
	}

	// Compute commit hash
	commitHash := sha256.Sum256(commitData)
	commitID := hex.EncodeToString(commitHash[:])

	// Store commit object
	err = r.storeObject(commitID, commitData)
	if err != nil {
		return "", fmt.Errorf("failed to store commit: %w", err)
	}

	// Update HEAD reference
	err = r.updateReference(r.State.HEAD, commitID)
	if err != nil {
		return "", fmt.Errorf("failed to update HEAD: %w", err)
	}

	// Update tracked files with the staged files
	for path, objID := range r.State.Stage {
		r.State.Tracked[path] = objID
	}

	// Clear staging area after successful commit
	r.State.Stage = make(map[string]string)

	// Save the updated index
	err = r.SaveIndex()
	if err != nil {
		return "", fmt.Errorf("failed to save index after commit: %w", err)
	}

	return commitID, nil
}

// resolveReference resolves a reference to a commit ID
func (r *Repository) resolveReference(ref string) (string, error) {
	// If it's a symbolic reference, resolve it
	if ref == "HEAD" {
		data, err := ioutil.ReadFile(filepath.Join(r.Path, DefaultKitDir, ref))
		if err != nil {
			return "", err
		}

		content := string(data)
		if len(content) > 4 && content[:4] == "ref:" {
			// It's a symbolic ref, resolve it
			symRef := content[4:]
			symRef = filepath.Join(r.Path, DefaultKitDir, strings.TrimSpace(symRef))
			data, err := ioutil.ReadFile(symRef)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(data)), nil
		}
		return string(data), nil
	}

	// Otherwise, read the reference file directly
	refPath := filepath.Join(r.Path, DefaultKitDir, ref)
	data, err := ioutil.ReadFile(refPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// updateReference updates a reference to point to a commit ID
func (r *Repository) updateReference(ref, commitID string) error {
	// If it's HEAD, we need to find what it points to
	if ref == "HEAD" {
		headPath := filepath.Join(r.Path, DefaultKitDir, "HEAD")
		data, err := ioutil.ReadFile(headPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		if len(data) > 4 && string(data[:4]) == "ref:" {
			// It's a symbolic ref, update the target
			target := strings.TrimSpace(string(data[4:]))
			return r.updateReference(target, commitID)
		}

		// Direct HEAD, update it
		return ioutil.WriteFile(headPath, []byte(commitID), 0644)
	}

	// Make sure parent directories exist
	refPath := filepath.Join(r.Path, DefaultKitDir, ref)
	err := os.MkdirAll(filepath.Dir(refPath), 0755)
	if err != nil {
		return err
	}

	// Update the reference file
	return ioutil.WriteFile(refPath, []byte(commitID), 0644)
}
