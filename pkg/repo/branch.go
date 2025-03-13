package repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Branch represents a branch in the repository
type Branch struct {
	Name      string // Branch name
	CommitID  string // Commit ID the branch points to
	IsCurrent bool   // Whether this is the current branch
}

// ListBranches returns a list of all branches in the repository
func (r *Repository) ListBranches() ([]Branch, error) {
	// Get branches directory
	branchesDir := filepath.Join(r.Path, DefaultKitDir, DefaultKitRefsDir, "heads")

	// Check if branches directory exists
	if _, err := os.Stat(branchesDir); os.IsNotExist(err) {
		return []Branch{}, nil
	}

	// Get current branch name
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		// If there's an error, just proceed without marking current branch
		currentBranch = ""
	}

	// Ensure 'main' branch is included if it exists
	mainBranchPath := filepath.Join(branchesDir, "main")
	if _, err := os.Stat(mainBranchPath); err == nil {
		// Main branch exists
	} else if os.IsNotExist(err) {
		// Create main branch if it doesn't exist but has commit history
		commitID, err := r.resolveReference("refs/heads/main")
		if err == nil && commitID != "" {
			// We have a commit but no branch file, create it
			r.updateReference("refs/heads/main", commitID)
		}
	}

	// Get all files in branches directory
	files, err := ioutil.ReadDir(branchesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read branches directory: %w", err)
	}

	// Build branches list
	branches := make([]Branch, 0, len(files))
	for _, file := range files {
		// Skip directories
		if file.IsDir() {
			continue
		}

		// Get branch name
		branchName := file.Name()

		// Read branch file to get commit ID
		branchPath := filepath.Join(branchesDir, branchName)
		commitID, err := ioutil.ReadFile(branchPath)
		if err != nil {
			continue // Skip branches we can't read
		}

		// Add branch to list
		branches = append(branches, Branch{
			Name:      branchName,
			CommitID:  strings.TrimSpace(string(commitID)),
			IsCurrent: branchName == currentBranch,
		})
	}

	return branches, nil
}

// CreateBranch creates a new branch from the current HEAD
func (r *Repository) CreateBranch(name string) error {
	// Check if branch name is valid
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for invalid characters in branch name
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("branch name contains invalid characters")
	}

	// Get current commit ID
	commitID, err := r.resolveReference(r.State.HEAD)
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	// Check if commit exists
	if commitID == "" {
		return fmt.Errorf("cannot create branch: no commit history")
	}

	// Check if branch already exists
	branchPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitRefsDir, "heads", name)
	if _, err := os.Stat(branchPath); err == nil {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	// Create branch reference
	if err := r.updateReference(fmt.Sprintf("refs/heads/%s", name), commitID); err != nil {
		return fmt.Errorf("failed to create branch reference: %w", err)
	}

	return nil
}

// CheckoutBranch switches to a different branch
func (r *Repository) CheckoutBranch(name string) error {
	// Check if branch exists
	branchPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitRefsDir, "heads", name)
	if _, err := os.Stat(branchPath); os.IsNotExist(err) {
		return fmt.Errorf("branch '%s' does not exist", name)
	}

	// Check for uncommitted changes
	if len(r.State.Stage) > 0 {
		return fmt.Errorf("you have uncommitted changes, please commit or stash them before switching branches")
	}

	// Get current branch tracking state for comparison
	oldTracked := make(map[string]string)
	for k, v := range r.State.Tracked {
		oldTracked[k] = v
	}

	// Get commit ID for the target branch
	targetCommitID, err := r.resolveReference(fmt.Sprintf("refs/heads/%s", name))
	if err != nil {
		return fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	// Read tree object for the target commit
	commitData, err := r.readObject(targetCommitID)
	if err != nil {
		return fmt.Errorf("failed to read commit object: %w", err)
	}

	// Unmarshal commit object
	var commit CommitObject
	if err := json.Unmarshal(commitData, &commit); err != nil {
		return fmt.Errorf("failed to unmarshal commit: %w", err)
	}

	// Read tree object for the commit
	treeData, err := r.readObject(commit.Tree)
	if err != nil {
		return fmt.Errorf("failed to read tree object: %w", err)
	}

	// Unmarshal tree object
	var tree TreeObject
	if err := json.Unmarshal(treeData, &tree); err != nil {
		return fmt.Errorf("failed to unmarshal tree: %w", err)
	}

	// Create a map to keep track of files to remove (files in old branch but not in new branch)
	filesToRemove := make(map[string]bool)
	for path := range oldTracked {
		filesToRemove[path] = true
	}

	// Reset tracked files
	r.State.Tracked = make(map[string]string)

	// Update working directory to match the branch content
	for path, entry := range tree.Entries {
		// Remove from filesToRemove since it's in the new branch
		delete(filesToRemove, path)

		// Track this file
		r.State.Tracked[path] = entry.ObjID

		// Get the object content
		objectData, err := r.readObject(entry.ObjID)
		if err != nil {
			return fmt.Errorf("failed to read object %s: %w", entry.ObjID, err)
		}

		// Get the absolute file path
		filePath := filepath.Join(r.Path, path)

		// Ensure parent directories exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}

		// Write the file
		if err := ioutil.WriteFile(filePath, objectData, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		// Update working tree
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
		}

		r.State.WorkTree[path] = WorkTreeEntry{
			Path:    path,
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
			Hash:    entry.ObjID,
		}
	}

	// Remove files from the old branch that aren't in the new branch
	for path := range filesToRemove {
		// For now, we just remove it from tracking but don't delete the files
		// This matches Git's behavior - files stay in the working directory but become untracked
		delete(r.State.WorkTree, path)
	}

	// Clear staging area - after checkout, nothing is staged
	r.State.Stage = make(map[string]string)

	// Update HEAD to point to the branch
	headPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitHeadFile)
	if err := ioutil.WriteFile(headPath, []byte(fmt.Sprintf("ref: refs/heads/%s\n", name)), 0644); err != nil {
		return fmt.Errorf("failed to update HEAD reference: %w", err)
	}

	// Update repository state
	r.State.HEAD = fmt.Sprintf("refs/heads/%s", name)

	// Save the updated index
	if err := r.SaveIndex(); err != nil {
		return fmt.Errorf("failed to save index after checkout: %w", err)
	}

	return nil
}

// GetCurrentBranch returns the name of the current branch
func (r *Repository) GetCurrentBranch() (string, error) {
	// Read HEAD file
	headPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitHeadFile)
	data, err := ioutil.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD file: %w", err)
	}

	// Check if HEAD is a symbolic reference
	content := string(data)
	if !strings.HasPrefix(content, "ref: refs/heads/") {
		return "", fmt.Errorf("HEAD is detached")
	}

	// Extract branch name
	branchRef := strings.TrimSpace(content[5:]) // Remove "ref: " prefix
	if !strings.HasPrefix(branchRef, "refs/heads/") {
		return "", fmt.Errorf("HEAD does not point to a branch")
	}

	return branchRef[11:], nil // Remove "refs/heads/" prefix
}
