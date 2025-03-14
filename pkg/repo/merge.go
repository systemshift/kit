package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success      bool            // Whether the merge was successful
	FastForward  bool            // Whether it was a fast-forward merge
	Conflicts    []MergeConflict // List of conflicts (if any)
	MergedCommit string          // The resulting commit ID after merge
}

// MergeConflict represents a conflict during merge
type MergeConflict struct {
	Path         string // File path with conflict
	OurContent   string // Content from our branch
	TheirContent string // Content from their branch
	BaseContent  string // Common ancestor content
	Resolution   string // Resolved content (if any)
}

// MergeOptions represents options for merge operations
type MergeOptions struct {
	Strategy    MergeStrategy // Merge strategy to use
	NoCommit    bool          // Don't auto-commit after merge
	Message     string        // Custom commit message
	UseSemantic bool          // Use semantic kernel for resolution
}

// DefaultMergeOptions provides default merge options
var DefaultMergeOptions = MergeOptions{
	Strategy:    AutoMerge,
	NoCommit:    false,
	Message:     "",
	UseSemantic: true,
}

// MergeStrategy represents the approach for merging
type MergeStrategy int

const (
	AutoMerge MergeStrategy = iota // Automatically merge when possible
	Ours                           // Always prefer our version in conflicts
	Theirs                         // Always prefer their version in conflicts
	Manual                         // Require manual resolution
)

// Merge merges a branch into the current branch
func (r *Repository) Merge(branchName string, options *MergeOptions) (*MergeResult, error) {
	if options == nil {
		options = &DefaultMergeOptions
	}

	// 1. Get current branch
	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// 2. Get current branch commit ID
	currentCommitID, err := r.resolveReference(fmt.Sprintf("refs/heads/%s", currentBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current branch: %w", err)
	}

	// 3. Get target branch commit ID
	targetCommitID, err := r.resolveReference(fmt.Sprintf("refs/heads/%s", branchName))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target branch: %w", err)
	}

	// 4. Check for uncommitted changes
	if len(r.State.Stage) > 0 {
		return nil, fmt.Errorf("cannot merge with uncommitted changes, please commit or stash them first")
	}

	// 5. Find merge base (common ancestor)
	baseCommitID, err := r.FindMergeBase(currentCommitID, targetCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Create result
	result := &MergeResult{
		Success:      false,
		FastForward:  false,
		Conflicts:    []MergeConflict{},
		MergedCommit: "",
	}

	// 6. Check for fast-forward merge
	if baseCommitID == currentCommitID {
		// Current branch is an ancestor of target branch, we can fast-forward
		result.FastForward = true

		// Update the current branch to point to the target branch commit
		err = r.updateReference(fmt.Sprintf("refs/heads/%s", currentBranch), targetCommitID)
		if err != nil {
			return nil, fmt.Errorf("failed to update reference for fast-forward merge: %w", err)
		}

		// Update the repository state with files from target branch
		err = r.CheckoutBranch(currentBranch)
		if err != nil {
			return nil, fmt.Errorf("failed to update working tree after merge: %w", err)
		}

		result.Success = true
		result.MergedCommit = targetCommitID
		return result, nil
	}

	// 7. Not a fast-forward, perform 3-way merge
	// Get trees for base, ours, and theirs
	baseTree, err := r.getTreeFromCommit(baseCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base tree: %w", err)
	}

	ourTree, err := r.getTreeFromCommit(currentCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get our tree: %w", err)
	}

	theirTree, err := r.getTreeFromCommit(targetCommitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get their tree: %w", err)
	}

	// 8. Perform the merge
	mergedTree, conflicts, err := r.MergeTrees(baseTree, ourTree, theirTree, options)
	if err != nil {
		return nil, fmt.Errorf("failed to merge trees: %w", err)
	}

	// Store conflicts in result
	result.Conflicts = conflicts

	// If there are conflicts and strategy is Manual, return with conflicts
	if len(conflicts) > 0 && options.Strategy == Manual {
		// Write conflict markers to files
		err = r.WriteConflictMarkers(conflicts)
		if err != nil {
			return nil, fmt.Errorf("failed to write conflict markers: %w", err)
		}
		return result, nil
	}

	// 9. If no conflicts or they were auto-resolved, create merge commit
	if !options.NoCommit {
		// Serialize merged tree
		treeData, err := json.MarshalIndent(mergedTree, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal merged tree: %w", err)
		}

		// Store tree object
		treeHash := sha256.Sum256(treeData)
		treeID := hex.EncodeToString(treeHash[:])
		err = r.storeObject(treeID, treeData)
		if err != nil {
			return nil, fmt.Errorf("failed to store merged tree: %w", err)
		}

		// Create merge commit
		message := options.Message
		if message == "" {
			message = fmt.Sprintf("Merge branch '%s' into %s", branchName, currentBranch)
		}

		mergeCommitID, err := r.CreateMergeCommit(message, currentCommitID, targetCommitID, treeID)
		if err != nil {
			return nil, fmt.Errorf("failed to create merge commit: %w", err)
		}

		// Update reference
		err = r.updateReference(fmt.Sprintf("refs/heads/%s", currentBranch), mergeCommitID)
		if err != nil {
			return nil, fmt.Errorf("failed to update branch reference: %w", err)
		}

		result.MergedCommit = mergeCommitID
	}

	// Update tracked files and working tree to match merged tree
	for path, entry := range mergedTree.Entries {
		// Update tracked files
		r.State.Tracked[path] = entry.ObjID

		// Update working tree
		objectData, err := r.readObject(entry.ObjID)
		if err != nil {
			return nil, fmt.Errorf("failed to read object %s: %w", entry.ObjID, err)
		}

		// Write file to working directory
		filePath := filepath.Join(r.Path, path)
		err = os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		err = os.WriteFile(filePath, objectData, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", path, err)
		}

		// Update working tree state
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get file info for %s: %w", path, err)
		}

		r.State.WorkTree[path] = WorkTreeEntry{
			Path:    path,
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
			Hash:    entry.ObjID,
		}
	}

	// Save index
	err = r.SaveIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to save index after merge: %w", err)
	}

	result.Success = true
	return result, nil
}

// FindMergeBase finds the common ancestor of two commits
func (r *Repository) FindMergeBase(commitA, commitB string) (string, error) {
	// Implementation of finding the lowest common ancestor in the commit graph
	// For simplicity, we'll use a breadth-first search approach

	// Get the history of commit A
	historyA := make(map[string]bool)
	queue := []string{commitA}

	for len(queue) > 0 {
		commit := queue[0]
		queue = queue[1:]

		// Check if we've already processed this commit
		if historyA[commit] {
			continue
		}

		// Mark this commit as part of history A
		historyA[commit] = true

		// Get the commit object
		commitData, err := r.readObject(commit)
		if err != nil {
			// Skip if we can't read the commit
			continue
		}

		// Unmarshal commit
		var commitObj CommitObject
		if err := json.Unmarshal(commitData, &commitObj); err != nil {
			continue
		}

		// Add parent to the queue
		if commitObj.Parent != "" {
			queue = append(queue, commitObj.Parent)
		}
	}

	// Now traverse commit B's history, stopping when we find a commit in A's history
	queue = []string{commitB}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		commit := queue[0]
		queue = queue[1:]

		// Check if we've already processed this commit
		if visited[commit] {
			continue
		}

		// Mark this commit as visited
		visited[commit] = true

		// Check if this commit is in A's history
		if historyA[commit] {
			return commit, nil
		}

		// Get the commit object
		commitData, err := r.readObject(commit)
		if err != nil {
			// Skip if we can't read the commit
			continue
		}

		// Unmarshal commit
		var commitObj CommitObject
		if err := json.Unmarshal(commitData, &commitObj); err != nil {
			continue
		}

		// Add parent to the queue
		if commitObj.Parent != "" {
			queue = append(queue, commitObj.Parent)
		}
	}

	// If we get here, there's no common ancestor (shouldn't happen in a proper repository)
	return "", fmt.Errorf("no common ancestor found")
}

// MergeTrees performs a 3-way merge of trees
func (r *Repository) MergeTrees(baseTree, ourTree, theirTree *TreeObject, options *MergeOptions) (*TreeObject, []MergeConflict, error) {
	// Create a new merged tree
	mergedTree := &TreeObject{
		Entries: make(map[string]TreeEntry),
	}

	// Collect all paths from all trees
	allPaths := make(map[string]bool)
	for path := range baseTree.Entries {
		allPaths[path] = true
	}
	for path := range ourTree.Entries {
		allPaths[path] = true
	}
	for path := range theirTree.Entries {
		allPaths[path] = true
	}

	// List to track conflicts
	conflicts := []MergeConflict{}

	// Process each path
	for path := range allPaths {
		baseEntry, baseExists := baseTree.Entries[path]
		ourEntry, ourExists := ourTree.Entries[path]
		theirEntry, theirExists := theirTree.Entries[path]

		// Case 1: File exists in base, ours, and theirs
		if baseExists && ourExists && theirExists {
			// If no changes on our side, take theirs
			if baseEntry.ObjID == ourEntry.ObjID && baseEntry.ObjID != theirEntry.ObjID {
				mergedTree.Entries[path] = theirEntry
				continue
			}

			// If no changes on their side, keep ours
			if baseEntry.ObjID == theirEntry.ObjID && baseEntry.ObjID != ourEntry.ObjID {
				mergedTree.Entries[path] = ourEntry
				continue
			}

			// If both sides made identical changes, no conflict
			if ourEntry.ObjID == theirEntry.ObjID {
				mergedTree.Entries[path] = ourEntry
				continue
			}

			// Both sides changed, attempt to merge the file contents
			baseContent, err := r.readObject(baseEntry.ObjID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read base content for %s: %w", path, err)
			}

			ourContent, err := r.readObject(ourEntry.ObjID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read our content for %s: %w", path, err)
			}

			theirContent, err := r.readObject(theirEntry.ObjID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read their content for %s: %w", path, err)
			}

			// Try to merge file contents
			var mergedContent string
			var hasConflict bool

			if options.UseSemantic && isCodeFile(path) {
				// Use semantic merge for code files
				mergedContent, hasConflict, err = r.SemanticMergeFiles(
					string(baseContent),
					string(ourContent),
					string(theirContent),
				)
			} else {
				// Use regular 3-way merge
				mergedContent, hasConflict, err = r.MergeFiles(
					string(baseContent),
					string(ourContent),
					string(theirContent),
					options.Strategy,
				)
			}

			if err != nil {
				return nil, nil, fmt.Errorf("failed to merge file %s: %w", path, err)
			}

			if hasConflict {
				// Add to conflicts list
				conflicts = append(conflicts, MergeConflict{
					Path:         path,
					BaseContent:  string(baseContent),
					OurContent:   string(ourContent),
					TheirContent: string(theirContent),
				})

				// Apply merge strategy for automatic resolution
				if options.Strategy == Ours {
					mergedContent = string(ourContent)
					hasConflict = false
				} else if options.Strategy == Theirs {
					mergedContent = string(theirContent)
					hasConflict = false
				}
			}

			// If we have a resolution, store it
			if !hasConflict {
				// Create a new blob for the merged content
				contentBytes := []byte(mergedContent)
				contentHash := sha256.Sum256(contentBytes)
				contentID := hex.EncodeToString(contentHash[:])

				// Store the merged content
				err = r.storeObject(contentID, contentBytes)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to store merged content for %s: %w", path, err)
				}

				// Add to merged tree
				mergedTree.Entries[path] = TreeEntry{
					Path:  path,
					Mode:  "100644", // Assume regular file
					Type:  "blob",
					ObjID: contentID,
				}
			}
		} else if !baseExists && ourExists && theirExists {
			// Case 2: File added in both ours and theirs
			// If identical, no conflict
			if ourEntry.ObjID == theirEntry.ObjID {
				mergedTree.Entries[path] = ourEntry
				continue
			}

			// Added differently in both, need to merge or report conflict
			ourContent, err := r.readObject(ourEntry.ObjID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read our content for %s: %w", path, err)
			}

			theirContent, err := r.readObject(theirEntry.ObjID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read their content for %s: %w", path, err)
			}

			// Try to merge without a base (harder)
			var mergedContent string
			var hasConflict bool

			if options.UseSemantic && isCodeFile(path) {
				// For semantic merge of new files, we can try with empty base
				var err error
				mergedContent, hasConflict, err = r.SemanticMergeFiles(
					"", // Empty base
					string(ourContent),
					string(theirContent),
				)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to merge new file %s with semantic merge: %w", path, err)
				}
			} else {
				// Without a base, always conflict unless strategy is specified
				hasConflict = true
				if options.Strategy == Ours {
					mergedContent = string(ourContent)
					hasConflict = false
				} else if options.Strategy == Theirs {
					mergedContent = string(theirContent)
					hasConflict = false
				} else {
					// Create a merge conflict
					mergedContent = "" // Will be filled with conflict markers
				}
			}

			if hasConflict {
				// Add to conflicts list
				conflicts = append(conflicts, MergeConflict{
					Path:         path,
					BaseContent:  "",
					OurContent:   string(ourContent),
					TheirContent: string(theirContent),
				})
			} else {
				// Store the merged content
				contentBytes := []byte(mergedContent)
				contentHash := sha256.Sum256(contentBytes)
				contentID := hex.EncodeToString(contentHash[:])

				err = r.storeObject(contentID, contentBytes)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to store merged content for %s: %w", path, err)
				}

				mergedTree.Entries[path] = TreeEntry{
					Path:  path,
					Mode:  "100644",
					Type:  "blob",
					ObjID: contentID,
				}
			}
		} else if baseExists && ourExists && !theirExists {
			// Case 3: File deleted in theirs but kept in ours
			// Keep our version
			mergedTree.Entries[path] = ourEntry
		} else if baseExists && !ourExists && theirExists {
			// Case 4: File deleted in ours but kept in theirs
			// Keep their version
			mergedTree.Entries[path] = theirEntry
		} else if !baseExists && ourExists && !theirExists {
			// Case 5: File added in ours only
			// Keep our version
			mergedTree.Entries[path] = ourEntry
		} else if !baseExists && !ourExists && theirExists {
			// Case 6: File added in theirs only
			// Keep their version
			mergedTree.Entries[path] = theirEntry
		}
		// If file was deleted in both, or never existed, don't add to merged tree
	}

	return mergedTree, conflicts, nil
}

// MergeFiles performs a 3-way merge of file contents
func (r *Repository) MergeFiles(baseContent, ourContent, theirContent string, strategy MergeStrategy) (string, bool, error) {
	// Simple line-based 3-way merge
	baseLines := strings.Split(baseContent, "\n")
	ourLines := strings.Split(ourContent, "\n")
	theirLines := strings.Split(theirContent, "\n")

	// Remove trailing empty lines
	if len(baseLines) > 0 && baseLines[len(baseLines)-1] == "" {
		baseLines = baseLines[:len(baseLines)-1]
	}
	if len(ourLines) > 0 && ourLines[len(ourLines)-1] == "" {
		ourLines = ourLines[:len(ourLines)-1]
	}
	if len(theirLines) > 0 && theirLines[len(theirLines)-1] == "" {
		theirLines = theirLines[:len(theirLines)-1]
	}

	// Build maps for faster lookup
	baseMap := make(map[string]int)
	for i, line := range baseLines {
		baseMap[line] = i
	}

	// Track which lines have been processed
	ourProcessed := make([]bool, len(ourLines))
	theirProcessed := make([]bool, len(theirLines))

	// Result lines
	var resultLines []string
	hasConflict := false

	// First pass: find unchanged and non-conflicting lines
	for i, ourLine := range ourLines {
		if ourProcessed[i] {
			continue
		}

		// Look for the same line in theirs
		found := false
		for j, theirLine := range theirLines {
			if theirProcessed[j] {
				continue
			}

			if ourLine == theirLine {
				// Line unchanged or changed identically
				resultLines = append(resultLines, ourLine)
				ourProcessed[i] = true
				theirProcessed[j] = true
				found = true
				break
			}
		}

		if !found {
			// Check if the line exists in base
			if baseIdx, exists := baseMap[ourLine]; exists {
				// Line unchanged in ours but changed or deleted in theirs
				// Need to check if there's a conflicting edit

				// Find closest match in their changes
				theirIdx := -1
				for j, processed := range theirProcessed {
					if !processed && j < len(theirLines) {
						if baseIdx-1 <= j && j <= baseIdx+1 {
							theirIdx = j
							break
						}
					}
				}

				if theirIdx != -1 && theirLines[theirIdx] != baseLines[baseIdx] {
					// Conflicting change
					hasConflict = true
				} else {
					// Non-conflicting, keep our change
					resultLines = append(resultLines, ourLine)
					ourProcessed[i] = true
				}
			} else {
				// Line added in ours
				resultLines = append(resultLines, ourLine)
				ourProcessed[i] = true
			}
		}
	}

	// Second pass: add any remaining their lines
	for j, theirLine := range theirLines {
		if !theirProcessed[j] {
			// Line unique to theirs
			resultLines = append(resultLines, theirLine)
		}
	}

	// If we detected conflicts, return a conflict marker string
	if hasConflict {
		// This is a simplified version, a real implementation would show
		// the exact conflicting sections with markers
		if strategy == Ours {
			return ourContent, false, nil
		} else if strategy == Theirs {
			return theirContent, false, nil
		}

		var sb strings.Builder
		sb.WriteString("<<<<<<< OURS\n")
		sb.WriteString(ourContent)
		sb.WriteString("\n=======\n")
		sb.WriteString(theirContent)
		sb.WriteString("\n>>>>>>> THEIRS\n")
		return sb.String(), true, nil
	}

	// Join result lines
	return strings.Join(resultLines, "\n"), false, nil
}

// SemanticMergeFiles uses semantic understanding to perform smart merges
func (r *Repository) SemanticMergeFiles(baseContent, ourContent, theirContent string) (string, bool, error) {
	// If base content is empty but we have both our and their, try to determine if they are semantically similar
	if baseContent == "" && ourContent != "" && theirContent != "" {
		similarity, _ := r.SemanticKernel.SemanticDiff(ourContent, theirContent)

		// If very similar, favor one version
		if similarity > 0.9 {
			// Prefer the more detailed one (longer)
			if len(ourContent) >= len(theirContent) {
				return ourContent, false, nil
			} else {
				return theirContent, false, nil
			}
		} else {
			// Not similar enough, report conflict
			return "", true, nil
		}
	}

	// Check if our changes and their changes are semantically compatible
	ourSimilarity, _ := r.SemanticKernel.SemanticDiff(baseContent, ourContent)
	theirSimilarity, _ := r.SemanticKernel.SemanticDiff(baseContent, theirContent)
	combinedSimilarity, _ := r.SemanticKernel.SemanticDiff(ourContent, theirContent)

	// If both made similar changes, pick one
	if combinedSimilarity > 0.8 {
		// Choose the one that changed more from base
		if ourSimilarity < theirSimilarity {
			return ourContent, false, nil
		} else {
			return theirContent, false, nil
		}
	}

	// If the changes are different but not conflicting in meaning,
	// we could use the regular merge but with semantic annotations
	result, hasConflict, err := r.MergeFiles(baseContent, ourContent, theirContent, AutoMerge)
	if err != nil {
		return "", true, err
	}

	// Add semantic annotation if there was a conflict
	if hasConflict {
		// Add semantic similarity information
		var sb strings.Builder
		sb.WriteString("// SEMANTIC ANALYSIS:\n")
		sb.WriteString(fmt.Sprintf("// Our changes similarity to base: %.2f%%\n", ourSimilarity*100))
		sb.WriteString(fmt.Sprintf("// Their changes similarity to base: %.2f%%\n", theirSimilarity*100))
		sb.WriteString(fmt.Sprintf("// Combined similarity: %.2f%%\n", combinedSimilarity*100))
		sb.WriteString(result)
		return sb.String(), true, nil
	}

	return result, false, nil
}

// CreateMergeCommit creates a merge commit with two parents
func (r *Repository) CreateMergeCommit(message string, parent1, parent2, treeID string) (string, error) {
	// Create commit object with two parents
	commit := CommitObject{
		Tree:      treeID,
		Parent:    parent1,                      // First parent is the current branch
		Parent2:   parent2,                      // Second parent is the branch being merged
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

	return commitID, nil
}

// WriteConflictMarkers writes conflicts to files in standard format
func (r *Repository) WriteConflictMarkers(conflicts []MergeConflict) error {
	for _, conflict := range conflicts {
		// Create conflict marker content
		var content strings.Builder
		content.WriteString("<<<<<<< HEAD\n")
		content.WriteString(conflict.OurContent)
		content.WriteString("\n=======\n")
		content.WriteString(conflict.TheirContent)
		content.WriteString("\n>>>>>>> THEIRS\n")

		// Write to file
		filePath := filepath.Join(r.Path, conflict.Path)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", conflict.Path, err)
		}

		err = os.WriteFile(filePath, []byte(content.String()), 0644)
		if err != nil {
			return fmt.Errorf("failed to write conflict markers to %s: %w", conflict.Path, err)
		}
	}

	return nil
}

// ResolveConflict marks a conflict as resolved
func (r *Repository) ResolveConflict(path string, resolution string) error {
	// Update the file with resolved content
	filePath := filepath.Join(r.Path, path)
	err := os.WriteFile(filePath, []byte(resolution), 0644)
	if err != nil {
		return fmt.Errorf("failed to write resolved content to %s: %w", path, err)
	}

	// Add the file to staging area to mark as resolved
	return r.Add(path)
}
