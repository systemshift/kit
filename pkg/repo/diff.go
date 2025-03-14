package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// DiffResult represents the result of a diff operation
type DiffResult struct {
	OldPath string      // Path in the old version
	NewPath string      // Path in the new version
	Chunks  []DiffChunk // Chunks of changes
}

// DiffChunk represents a chunk of changes in a diff
type DiffChunk struct {
	OldStart  int      // Starting line in old version
	OldLength int      // Number of lines in old version
	NewStart  int      // Starting line in new version
	NewLength int      // Number of lines in new version
	Lines     []string // Lines with prefixes (+, -, ' ')
}

// DiffOptions represents options for diff operations
type DiffOptions struct {
	ContextLines int  // Number of context lines to show
	Semantic     bool // Whether to use semantic diff
}

// DefaultDiffOptions provides default diff options
var DefaultDiffOptions = DiffOptions{
	ContextLines: 3,
	Semantic:     false,
}

// Diff compares two items and returns the differences
// The items could be commit IDs, file paths, or a mix
func (r *Repository) Diff(itemA, itemB string, options *DiffOptions) ([]DiffResult, error) {
	if options == nil {
		options = &DefaultDiffOptions
	}

	// If both items appear to be file paths, diff them directly
	if isFilePath(itemA) && isFilePath(itemB) {
		return r.DiffFiles(itemA, itemB, options)
	}

	// If itemA is empty, use HEAD
	if itemA == "" {
		head, err := r.resolveReference(r.State.HEAD)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve HEAD: %w", err)
		}
		itemA = head
	}

	// If itemB is empty, use working directory
	if itemB == "" {
		return r.DiffWorkingTree(itemA, options)
	}

	// If itemA is a file path, try to compare with itemB as a commit
	if isFilePath(itemA) {
		file1Content, err := r.readWorkingFile(itemA)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", itemA, err)
		}

		// Get the file from itemB commit
		tree, err := r.getTreeFromCommit(itemB)
		if err != nil {
			return nil, fmt.Errorf("failed to get tree for commit %s: %w", itemB, err)
		}

		// Look for a matching file in the tree
		found := false
		for path, entry := range tree.Entries {
			if path == itemA {
				found = true
				file2Content, err := r.readObject(entry.ObjID)
				if err != nil {
					return nil, fmt.Errorf("failed to read blob %s: %w", entry.ObjID, err)
				}

				// Compare the files
				chunks := diffContent(string(file2Content), string(file1Content), options.ContextLines)
				return []DiffResult{
					{
						OldPath: itemA,
						NewPath: itemA,
						Chunks:  chunks,
					},
				}, nil
			}
		}

		if !found {
			return nil, fmt.Errorf("file %s not found in commit %s", itemA, itemB)
		}
	}

	// If itemB is a file path, try to compare with itemA as a commit
	if isFilePath(itemB) {
		file2Content, err := r.readWorkingFile(itemB)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", itemB, err)
		}

		// Get the file from itemA commit
		tree, err := r.getTreeFromCommit(itemA)
		if err != nil {
			return nil, fmt.Errorf("failed to get tree for commit %s: %w", itemA, err)
		}

		// Look for a matching file in the tree
		found := false
		for path, entry := range tree.Entries {
			if path == itemB {
				found = true
				file1Content, err := r.readObject(entry.ObjID)
				if err != nil {
					return nil, fmt.Errorf("failed to read blob %s: %w", entry.ObjID, err)
				}

				// Compare the files
				chunks := diffContent(string(file1Content), string(file2Content), options.ContextLines)
				return []DiffResult{
					{
						OldPath: itemB,
						NewPath: itemB,
						Chunks:  chunks,
					},
				}, nil
			}
		}

		if !found {
			return nil, fmt.Errorf("file %s not found in commit %s", itemB, itemA)
		}
	}

	// Get the trees for both commits
	treeA, err := r.getTreeFromCommit(itemA)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for commit %s: %w", itemA, err)
	}

	treeB, err := r.getTreeFromCommit(itemB)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for commit %s: %w", itemB, err)
	}

	// Compare the trees
	return r.diffTrees(treeA, treeB, options)
}

// DiffFiles compares two files and returns the differences
func (r *Repository) DiffFiles(file1Path, file2Path string, options *DiffOptions) ([]DiffResult, error) {
	// Read file contents
	file1Content, err := r.readWorkingFile(file1Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file1Path, err)
	}

	file2Content, err := r.readWorkingFile(file2Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file2Path, err)
	}

	// Perform diff based on options
	var chunks []DiffChunk
	if options.Semantic && (isCodeFile(file1Path) || isCodeFile(file2Path)) {
		// Use semantic diff for code files
		semanticChunks, err := r.semanticDiffContent(string(file1Content), string(file2Content), options.ContextLines)
		if err == nil {
			chunks = semanticChunks
		} else {
			// Fall back to regular diff if semantic diff fails
			chunks = diffContent(string(file1Content), string(file2Content), options.ContextLines)
		}
	} else {
		// Use regular diff for non-code files
		chunks = diffContent(string(file1Content), string(file2Content), options.ContextLines)
	}

	// Return the diff result
	return []DiffResult{
		{
			OldPath: file1Path,
			NewPath: file2Path,
			Chunks:  chunks,
		},
	}, nil
}

// DiffWorkingTree compares a commit with the working tree
func (r *Repository) DiffWorkingTree(commit string, options *DiffOptions) ([]DiffResult, error) {
	if options == nil {
		options = &DefaultDiffOptions
	}

	// Get the tree for the commit
	tree, err := r.getTreeFromCommit(commit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for commit %s: %w", commit, err)
	}

	// Create a list to hold the diff results
	results := []DiffResult{}

	// Compare each file in the tree with the working tree
	for path, entry := range tree.Entries {
		// Get the content from the blob
		blobContent, err := r.readObject(entry.ObjID)
		if err != nil {
			return nil, fmt.Errorf("failed to read blob %s: %w", entry.ObjID, err)
		}

		// Try to read the file from the working tree
		workingContent, err := r.readWorkingFile(path)
		if err != nil {
			// File doesn't exist in working tree, consider it deleted
			chunks := []DiffChunk{
				{
					OldStart:  1,
					OldLength: len(bytes.Split(blobContent, []byte{'\n'})),
					NewStart:  0,
					NewLength: 0,
					Lines:     prefixLines(string(blobContent), "-"),
				},
			}
			results = append(results, DiffResult{
				OldPath: path,
				NewPath: "/dev/null",
				Chunks:  chunks,
			})
			continue
		}

		// File exists in both commit and working tree, diff them
		if !bytes.Equal(blobContent, workingContent) {
			chunks := diffContent(string(blobContent), string(workingContent), options.ContextLines)
			results = append(results, DiffResult{
				OldPath: path,
				NewPath: path,
				Chunks:  chunks,
			})
		}
	}

	// Check for new files in working tree
	for path := range r.State.WorkTree {
		if _, ok := tree.Entries[path]; !ok {
			// File exists in working tree but not in commit, consider it new
			workingContent, err := r.readWorkingFile(path)
			if err != nil {
				continue // Shouldn't happen, but skip if it does
			}

			chunks := []DiffChunk{
				{
					OldStart:  0,
					OldLength: 0,
					NewStart:  1,
					NewLength: len(bytes.Split(workingContent, []byte{'\n'})),
					Lines:     prefixLines(string(workingContent), "+"),
				},
			}
			results = append(results, DiffResult{
				OldPath: "/dev/null",
				NewPath: path,
				Chunks:  chunks,
			})
		}
	}

	return results, nil
}

// getTreeFromCommit gets the tree object from a commit
func (r *Repository) getTreeFromCommit(commitID string) (*TreeObject, error) {
	// Read the commit object
	commitData, err := r.readObject(commitID)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit %s: %w", commitID, err)
	}

	// Unmarshal commit object
	var commit CommitObject
	if err := json.Unmarshal(commitData, &commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit %s: %w", commitID, err)
	}

	// Read the tree object
	treeData, err := r.readObject(commit.Tree)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree %s: %w", commit.Tree, err)
	}

	// Unmarshal tree object
	var tree TreeObject
	if err := json.Unmarshal(treeData, &tree); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tree %s: %w", commit.Tree, err)
	}

	return &tree, nil
}

// diffTrees compares two tree objects and returns the differences
func (r *Repository) diffTrees(treeA, treeB *TreeObject, options *DiffOptions) ([]DiffResult, error) {
	// Create a list to hold the diff results
	results := []DiffResult{}

	// Track all paths in both trees
	allPaths := make(map[string]bool)
	for path := range treeA.Entries {
		allPaths[path] = true
	}
	for path := range treeB.Entries {
		allPaths[path] = true
	}

	// Compare each file in the trees
	for path := range allPaths {
		entryA, okA := treeA.Entries[path]
		entryB, okB := treeB.Entries[path]

		// File deleted (exists in A but not B)
		if okA && !okB {
			blobContent, err := r.readObject(entryA.ObjID)
			if err != nil {
				return nil, fmt.Errorf("failed to read blob %s: %w", entryA.ObjID, err)
			}

			chunks := []DiffChunk{
				{
					OldStart:  1,
					OldLength: len(bytes.Split(blobContent, []byte{'\n'})),
					NewStart:  0,
					NewLength: 0,
					Lines:     prefixLines(string(blobContent), "-"),
				},
			}
			results = append(results, DiffResult{
				OldPath: path,
				NewPath: "/dev/null",
				Chunks:  chunks,
			})
			continue
		}

		// File added (exists in B but not A)
		if !okA && okB {
			blobContent, err := r.readObject(entryB.ObjID)
			if err != nil {
				return nil, fmt.Errorf("failed to read blob %s: %w", entryB.ObjID, err)
			}

			chunks := []DiffChunk{
				{
					OldStart:  0,
					OldLength: 0,
					NewStart:  1,
					NewLength: len(bytes.Split(blobContent, []byte{'\n'})),
					Lines:     prefixLines(string(blobContent), "+"),
				},
			}
			results = append(results, DiffResult{
				OldPath: "/dev/null",
				NewPath: path,
				Chunks:  chunks,
			})
			continue
		}

		// File modified (exists in both but different)
		if entryA.ObjID != entryB.ObjID {
			blobContentA, err := r.readObject(entryA.ObjID)
			if err != nil {
				return nil, fmt.Errorf("failed to read blob %s: %w", entryA.ObjID, err)
			}

			blobContentB, err := r.readObject(entryB.ObjID)
			if err != nil {
				return nil, fmt.Errorf("failed to read blob %s: %w", entryB.ObjID, err)
			}

			// If using semantic diff and appropriate file type, use semantic diff
			if options.Semantic && isCodeFile(path) {
				chunks, err := r.semanticDiffContent(string(blobContentA), string(blobContentB), options.ContextLines)
				if err != nil {
					// Fall back to regular diff if semantic diff fails
					chunks = diffContent(string(blobContentA), string(blobContentB), options.ContextLines)
				}
				results = append(results, DiffResult{
					OldPath: path,
					NewPath: path,
					Chunks:  chunks,
				})
			} else {
				// Use regular text diff
				chunks := diffContent(string(blobContentA), string(blobContentB), options.ContextLines)
				results = append(results, DiffResult{
					OldPath: path,
					NewPath: path,
					Chunks:  chunks,
				})
			}
		}
	}

	return results, nil
}

// readWorkingFile reads a file from the working tree
func (r *Repository) readWorkingFile(path string) ([]byte, error) {
	absPath := filepath.Join(r.Path, path)
	content, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// diffContent compares two strings line by line and returns the differences
// This is a simple implementation of the Myers diff algorithm
func diffContent(oldContent, newContent string, contextLines int) []DiffChunk {
	// Split content into lines
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Remove trailing empty line if present
	if len(oldLines) > 0 && oldLines[len(oldLines)-1] == "" {
		oldLines = oldLines[:len(oldLines)-1]
	}
	if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
		newLines = newLines[:len(newLines)-1]
	}

	// Find the longest common subsequence (LCS)
	lcs := longestCommonSubsequence(oldLines, newLines)

	// Convert LCS to edit script
	edits := convertToEdits(oldLines, newLines, lcs)

	// Group edits into chunks with context
	chunks := groupEditsIntoChunks(oldLines, newLines, edits, contextLines)

	return chunks
}

// semanticDiffContent performs a semantic diff on code content
func (r *Repository) semanticDiffContent(oldContent, newContent string, contextLines int) ([]DiffChunk, error) {
	// First check if there's a semantic difference using the semantic kernel
	similarity, _ := r.SemanticKernel.SemanticDiff(oldContent, newContent)

	// If very similar, return a special chunk indicating semantic equivalence
	if similarity > 0.9 {
		return []DiffChunk{
			{
				OldStart:  1,
				OldLength: len(strings.Split(oldContent, "\n")),
				NewStart:  1,
				NewLength: len(strings.Split(newContent, "\n")),
				Lines: []string{
					fmt.Sprintf("// SEMANTIC SIMILARITY: %.2f%%", similarity*100),
					"// Code has been refactored but maintains the same meaning",
				},
			},
		}, nil
	}

	// For less similar code, fall back to regular diff but add semantic annotations
	chunks := diffContent(oldContent, newContent, contextLines)

	// Add semantic analysis as first chunk
	analysisChunk := DiffChunk{
		OldStart:  0,
		OldLength: 0,
		NewStart:  0,
		NewLength: 0,
		Lines: []string{
			fmt.Sprintf("// SEMANTIC DIFF (Similarity: %.2f%%)", similarity*100),
			"// Code changes may affect behavior",
		},
	}

	// Add the analysis chunk at the beginning
	return append([]DiffChunk{analysisChunk}, chunks...), nil
}

// longestCommonSubsequence finds the longest common subsequence of lines
func longestCommonSubsequence(a, b []string) [][]int {
	// Create a matrix of LCS lengths
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill the matrix
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Build the LCS
	lcs := [][]int{}
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([][]int{{i - 1, j - 1}}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// Edit represents an edit operation (insert, delete, or unchanged)
type Edit struct {
	Type      string // "insert", "delete", or "unchanged"
	OldIndex  int    // Index in old content
	NewIndex  int    // Index in new content
	LineValue string // The line content
}

// convertToEdits converts an LCS to a list of edit operations
func convertToEdits(oldLines, newLines []string, lcs [][]int) []Edit {
	edits := []Edit{}
	oldIdx, newIdx := 0, 0

	// Map LCS indices for easier lookup
	lcsMap := make(map[int]map[int]bool)
	for _, pair := range lcs {
		if lcsMap[pair[0]] == nil {
			lcsMap[pair[0]] = make(map[int]bool)
		}
		lcsMap[pair[0]][pair[1]] = true
	}

	// Process all lines and generate edits
	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		// Common line (part of LCS)
		if oldIdx < len(oldLines) && newIdx < len(newLines) &&
			lcsMap[oldIdx] != nil && lcsMap[oldIdx][newIdx] {
			edits = append(edits, Edit{
				Type:      "unchanged",
				OldIndex:  oldIdx,
				NewIndex:  newIdx,
				LineValue: oldLines[oldIdx],
			})
			oldIdx++
			newIdx++
		} else if oldIdx < len(oldLines) && (newIdx >= len(newLines) ||
			(oldIdx+1 < len(oldLines) && newIdx < len(newLines) &&
				lcsMap[oldIdx+1] != nil && lcsMap[oldIdx+1][newIdx])) {
			// Line deleted from old content
			edits = append(edits, Edit{
				Type:      "delete",
				OldIndex:  oldIdx,
				NewIndex:  -1,
				LineValue: oldLines[oldIdx],
			})
			oldIdx++
		} else if newIdx < len(newLines) {
			// Line added in new content
			edits = append(edits, Edit{
				Type:      "insert",
				OldIndex:  -1,
				NewIndex:  newIdx,
				LineValue: newLines[newIdx],
			})
			newIdx++
		}
	}

	return edits
}

// groupEditsIntoChunks groups edits into chunks with context lines
func groupEditsIntoChunks(oldLines, newLines []string, edits []Edit, contextLines int) []DiffChunk {
	chunks := []DiffChunk{}
	if len(edits) == 0 {
		return chunks
	}

	// Initialize the first chunk
	currentChunk := DiffChunk{
		OldStart:  -1,
		OldLength: 0,
		NewStart:  -1,
		NewLength: 0,
		Lines:     []string{},
	}

	// Unroll edits into chunks
	for i, edit := range edits {
		isChange := edit.Type != "unchanged"
		isFirstOrLast := i == 0 || i == len(edits)-1
		isNearChange := false

		// Check if this is near a change
		for j := max(0, i-contextLines); j < min(len(edits), i+contextLines+1); j++ {
			if j != i && edits[j].Type != "unchanged" {
				isNearChange = true
				break
			}
		}

		// If this is a change, first/last line, or near a change, include it
		if isChange || isFirstOrLast || isNearChange {
			// If starting a new chunk
			if currentChunk.OldStart == -1 {
				if edit.OldIndex != -1 {
					currentChunk.OldStart = edit.OldIndex + 1 // 1-based indexing for output
				} else {
					currentChunk.OldStart = 0
				}

				if edit.NewIndex != -1 {
					currentChunk.NewStart = edit.NewIndex + 1 // 1-based indexing for output
				} else {
					currentChunk.NewStart = 0
				}
			}

			// Add line to chunk
			var prefix string
			switch edit.Type {
			case "insert":
				prefix = "+"
				currentChunk.NewLength++
			case "delete":
				prefix = "-"
				currentChunk.OldLength++
			case "unchanged":
				prefix = " "
				currentChunk.OldLength++
				currentChunk.NewLength++
			}
			currentChunk.Lines = append(currentChunk.Lines, prefix+edit.LineValue)
		} else if len(currentChunk.Lines) > 0 {
			// Not including this line and we have a chunk, so finalize it
			chunks = append(chunks, currentChunk)
			currentChunk = DiffChunk{
				OldStart:  -1,
				OldLength: 0,
				NewStart:  -1,
				NewLength: 0,
				Lines:     []string{},
			}
		}
	}

	// Add the last chunk if it has lines
	if len(currentChunk.Lines) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// FormatDiff formats a diff result into a string
func FormatDiff(results []DiffResult) string {
	var buf strings.Builder

	for _, result := range results {
		// File header
		if result.OldPath == "/dev/null" {
			buf.WriteString(fmt.Sprintf("--- /dev/null\n"))
		} else {
			buf.WriteString(fmt.Sprintf("--- a/%s\n", result.OldPath))
		}

		if result.NewPath == "/dev/null" {
			buf.WriteString(fmt.Sprintf("+++ /dev/null\n"))
		} else {
			buf.WriteString(fmt.Sprintf("+++ b/%s\n", result.NewPath))
		}

		// Chunks
		for _, chunk := range result.Chunks {
			// Chunk header
			buf.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
				chunk.OldStart, chunk.OldLength,
				chunk.NewStart, chunk.NewLength))

			// Chunk lines
			for _, line := range chunk.Lines {
				buf.WriteString(line + "\n")
			}
		}

		buf.WriteString("\n")
	}

	return buf.String()
}

// prefixLines adds a prefix to each line in a string
func prefixLines(content, prefix string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	// Remove trailing empty line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		result = append(result, prefix+line)
	}
	return result
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isCodeFile returns true if the file is likely to contain code
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExtensions := map[string]bool{
		".go":    true,
		".js":    true,
		".ts":    true,
		".jsx":   true,
		".tsx":   true,
		".py":    true,
		".java":  true,
		".c":     true,
		".cpp":   true,
		".h":     true,
		".cs":    true,
		".rb":    true,
		".php":   true,
		".rs":    true,
		".swift": true,
		".kt":    true,
	}
	return codeExtensions[ext]
}

// isFilePath returns true if the string is likely a file path rather than a commit ID
func isFilePath(path string) bool {
	// Commit IDs are typically 40 or 64 character hex strings
	// If it contains file extension or directory separators, it's likely a file path
	return strings.Contains(path, ".") || strings.Contains(path, "/") || strings.Contains(path, "\\")
}
