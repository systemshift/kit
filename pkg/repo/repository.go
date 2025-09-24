package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/systemshift/kit/pkg/kernel"
)

const (
	// DefaultKitDir is the default directory for Kit repository
	DefaultKitDir = ".kit"
	// DefaultKitConfig is the default configuration file
	DefaultKitConfig = "config"
	// DefaultKitObjectsDir is the default directory for storing objects
	DefaultKitObjectsDir = "objects"
	// DefaultKitRefsDir is the default directory for storing references
	DefaultKitRefsDir = "refs"
	// DefaultKitHeadFile is the default HEAD file
	DefaultKitHeadFile = "HEAD"
	// DefaultKitIndexFile is the default index file
	DefaultKitIndexFile = "index"
)

// RepositoryState represents the state of a repository
type RepositoryState struct {
	HEAD     string                   // Current HEAD reference
	Stage    map[string]string        // Staged files (path -> object ID)
	Tracked  map[string]string        // Tracked files (path -> object ID from latest commit)
	WorkTree map[string]WorkTreeEntry // Working tree files
}

// WorkTreeEntry represents a file in the working tree
type WorkTreeEntry struct {
	Path    string    // File path
	Size    int64     // File size
	ModTime time.Time // Last modification time
	Hash    string    // Hash of the file content
}

// Repository represents a Kit repository
type Repository struct {
	Path            string                   // Path to the repository root
	IntegrityKernel *kernel.IntegrityKernel  // For repository integrity verification
	SemanticKernel  *kernel.SemanticKernel   // For semantic diffing and merging
	RetrievalKernel *kernel.RetrievalKernel  // For efficient content search
	State           *RepositoryState         // Current repository state
}

// NewRepository creates a new repository instance
func NewRepository(path string) (*Repository, error) {
	// Create default kernels with optimized parameters
	integrityKernel := kernel.NewIntegrityKernel(256, 128, 0.5, 42)     // More features for better accuracy
	semanticKernel := kernel.NewSemanticKernel(512, 0.75)              // Higher dimension for better semantic understanding
	retrievalKernel := kernel.NewRetrievalKernel(200, 1000000, 20, 42) // MinHash with LSH for fast retrieval

	// Initialize repository state
	state := &RepositoryState{
		HEAD:     "refs/heads/main",
		Stage:    make(map[string]string),
		Tracked:  make(map[string]string),
		WorkTree: make(map[string]WorkTreeEntry),
	}

	// Create the repository
	repo := &Repository{
		Path:            filepath.Clean(path),
		IntegrityKernel: integrityKernel,
		SemanticKernel:  semanticKernel,
		RetrievalKernel: retrievalKernel,
		State:           state,
	}

	// Load index if repository exists
	if IsRepository(path) {
		if err := repo.LoadIndex(); err != nil {
			return nil, fmt.Errorf("failed to load index: %w", err)
		}
	}

	return repo, nil
}

// FindSimilarContent uses the RetrievalKernel to find files similar to the given content
func (r *Repository) FindSimilarContent(content string, threshold float64) (map[string]float64, error) {
	if r.RetrievalKernel == nil {
		return nil, fmt.Errorf("retrieval kernel not initialized")
	}

	results := make(map[string]float64)

	// Compare against all tracked files
	for path, objID := range r.State.Tracked {
		// Read the object data
		objData, err := r.readObject(objID)
		if err != nil {
			continue
		}

		// Estimate similarity using MinHash
		similarity := r.RetrievalKernel.EstimateSimilarity(content, string(objData))

		// Include if above threshold
		if similarity >= threshold {
			results[path] = similarity
		}
	}

	return results, nil
}

// FindDuplicateContent identifies potentially duplicate content in the repository
func (r *Repository) FindDuplicateContent() (map[string][]string, error) {
	if r.RetrievalKernel == nil {
		return nil, fmt.Errorf("retrieval kernel not initialized")
	}

	duplicates := make(map[string][]string)
	processed := make(map[string]bool)

	// Compare all tracked files against each other
	for path1, objID1 := range r.State.Tracked {
		if processed[path1] {
			continue
		}

		objData1, err := r.readObject(objID1)
		if err != nil {
			continue
		}

		var similar []string
		for path2, objID2 := range r.State.Tracked {
			if path1 == path2 || processed[path2] {
				continue
			}

			objData2, err := r.readObject(objID2)
			if err != nil {
				continue
			}

			// Check if likely similar using LSH (fast pre-filter)
			if r.RetrievalKernel.AreLikelySimilar(string(objData1), string(objData2)) {
				// Confirm with actual similarity calculation
				similarity := r.RetrievalKernel.EstimateSimilarity(string(objData1), string(objData2))
				if similarity > 0.8 { // High similarity threshold for duplicates
					similar = append(similar, path2)
					processed[path2] = true
				}
			}
		}

		if len(similar) > 0 {
			similar = append([]string{path1}, similar...) // Include the original file
			duplicates[path1] = similar
		}
		processed[path1] = true
	}

	return duplicates, nil
}

// Initialize initializes a new repository at the given path
func (r *Repository) Initialize() error {
	// Create .kit directory and subdirectories
	kitDir := filepath.Join(r.Path, DefaultKitDir)

	// Check if repository already exists
	if _, err := os.Stat(kitDir); err == nil {
		return errors.New("repository already exists")
	}

	// Create required directories
	dirs := []string{
		kitDir,
		filepath.Join(kitDir, DefaultKitObjectsDir),
		filepath.Join(kitDir, DefaultKitRefsDir),
		filepath.Join(kitDir, DefaultKitRefsDir, "heads"),
		filepath.Join(kitDir, DefaultKitRefsDir, "tags"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create HEAD file pointing to main branch
	headPath := filepath.Join(kitDir, DefaultKitHeadFile)
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	// Create empty index file
	indexPath := filepath.Join(kitDir, DefaultKitIndexFile)
	if err := os.WriteFile(indexPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create index file: %w", err)
	}

	// Create basic configuration
	configPath := filepath.Join(kitDir, DefaultKitConfig)
	configContent := `[core]
	repositoryformatversion = 0
	filemode = false
	bare = false
[kit]
	integrityfeatures = 128
	integrityinputdim = 64
	integritygamma = 0.1
	semanticembeddingdim = 128
	semanticminimumscore = 0.7
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	return nil
}

// Add stages a file for commit
func (r *Repository) Add(path string) error {
	// Get absolute path
	absPath := filepath.Join(r.Path, path)

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Generate object ID
	hash := sha256.Sum256(content)
	objID := hex.EncodeToString(hash[:])

	// Store the object
	err = r.storeObject(objID, content)
	if err != nil {
		return fmt.Errorf("failed to store object: %w", err)
	}

	// Update stage
	r.State.Stage[path] = objID

	// Update working tree
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	r.State.WorkTree[path] = WorkTreeEntry{
		Path:    path,
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
		Hash:    objID,
	}

	// Save index
	if err := r.SaveIndex(); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// Status shows the status of the repository
func (r *Repository) Status() (string, error) {
	// Get current branch name
	branchName, err := r.GetCurrentBranch()
	if err != nil {
		branchName = "main" // Default to main if we can't determine branch
	}

	// Check for different file states
	modified := []string{}         // Modified but not staged
	staged := []string{}           // Staged for commit
	untracked := []string{}        // Not tracked by Git
	modified_tracked := []string{} // Modified since last commit (tracked files)

	// Get all files in working directory
	err = filepath.Walk(r.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .kit directory and subdirectories
		if strings.Contains(path, DefaultKitDir) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(r.Path, path)
		if err != nil {
			return err
		}

		// Check the file's status
		isStaged := false
		isTracked := false

		// Check if file is in staging area
		if _, ok := r.State.Stage[relPath]; ok {
			isStaged = true
			staged = append(staged, relPath)

			// Check if it's also modified since staging
			if entry, ok := r.State.WorkTree[relPath]; ok {
				fileInfo := info
				if entry.ModTime != fileInfo.ModTime() || entry.Size != fileInfo.Size() {
					modified = append(modified, relPath)
				}
			}
		}

		// Check if file is tracked (committed)
		if _, ok := r.State.Tracked[relPath]; ok {
			isTracked = true

			// If not staged but tracked, check if modified since last commit
			if !isStaged {
				// Get file hash
				content, err := os.ReadFile(path)
				if err == nil {
					hash := sha256.Sum256(content)
					objID := hex.EncodeToString(hash[:])

					// Compare with tracked version
					if objID != r.State.Tracked[relPath] {
						modified_tracked = append(modified_tracked, relPath)
					}
				}
			}
		}

		// If neither staged nor tracked, it's untracked
		if !isStaged && !isTracked {
			untracked = append(untracked, relPath)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	// Build status message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("On branch %s\n\n", branchName))

	if len(staged) > 0 {
		sb.WriteString("Changes to be committed:\n")
		for _, file := range staged {
			// Check if this is a new file or modified file
			if _, ok := r.State.Tracked[file]; ok {
				sb.WriteString(fmt.Sprintf("  modified: %s\n", file))
			} else {
				sb.WriteString(fmt.Sprintf("  new file: %s\n", file))
			}
		}
		sb.WriteString("\n")
	}

	if len(modified) > 0 {
		sb.WriteString("Changes not staged for commit:\n")
		for _, file := range modified {
			sb.WriteString(fmt.Sprintf("  modified: %s\n", file))
		}
		sb.WriteString("\n")
	}

	if len(modified_tracked) > 0 {
		sb.WriteString("Changes not staged for commit:\n")
		for _, file := range modified_tracked {
			sb.WriteString(fmt.Sprintf("  modified: %s\n", file))
		}
		sb.WriteString("\n")
	}

	if len(untracked) > 0 {
		sb.WriteString("Untracked files:\n")
		for _, file := range untracked {
			sb.WriteString(fmt.Sprintf("  %s\n", file))
		}
		sb.WriteString("\n")
	}

	if len(staged) == 0 && len(modified) == 0 && len(modified_tracked) == 0 && len(untracked) == 0 {
		sb.WriteString("nothing to commit, working tree clean\n")
	}

	return sb.String(), nil
}

// storeObject stores an object in the object database
func (r *Repository) storeObject(objID string, content []byte) error {
	objDir := filepath.Join(r.Path, DefaultKitDir, DefaultKitObjectsDir)
	objPath := filepath.Join(objDir, objID[:2], objID[2:])

	// Create subdirectory if it doesn't exist
	if err := os.MkdirAll(filepath.Join(objDir, objID[:2]), 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}

	// Write object to file
	if err := os.WriteFile(objPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write object: %w", err)
	}

	return nil
}

// readObject reads an object from the object database
func (r *Repository) readObject(objID string) ([]byte, error) {
	objPath := filepath.Join(r.Path, DefaultKitDir, DefaultKitObjectsDir, objID[:2], objID[2:])
	content, err := os.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", objID, err)
	}
	return content, nil
}

// IsRepository checks if the given path is a Kit repository
func IsRepository(path string) bool {
	kitDir := filepath.Join(path, DefaultKitDir)
	if _, err := os.Stat(kitDir); err != nil {
		return false
	}
	return true
}
