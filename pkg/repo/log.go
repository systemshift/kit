package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// CommitLog represents a commit in the log output
type CommitLog struct {
	ID        string    // Commit ID
	Author    string    // Author name and email
	Timestamp time.Time // Commit timestamp
	Message   string    // Commit message
}

// Log returns the commit history of the repository
func (r *Repository) Log() ([]*CommitLog, error) {
	// Get current commit ID from HEAD
	commitID, err := r.resolveReference(r.State.HEAD)
	if err != nil {
		if os.IsNotExist(err) {
			// No commits yet
			return []*CommitLog{}, nil
		}
		return nil, fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	// Traverse commit history
	var log []*CommitLog
	for commitID != "" {
		// Read commit object
		commitData, err := r.readObject(commitID)
		if err != nil {
			// If we can't read the commit, stop the traversal
			break
		}

		// Unmarshal commit object
		var commit CommitObject
		if err := json.Unmarshal(commitData, &commit); err != nil {
			return nil, fmt.Errorf("failed to unmarshal commit %s: %w", commitID, err)
		}

		// Add commit to log
		log = append(log, &CommitLog{
			ID:        commitID,
			Author:    commit.Author,
			Timestamp: commit.Timestamp,
			Message:   commit.Message,
		})

		// Move to parent commit
		commitID = commit.Parent
	}

	return log, nil
}

// FormatLog formats a commit log for display
func FormatLog(log []*CommitLog) string {
	var sb strings.Builder

	for i, commit := range log {
		// Add separator except for the first line
		if i > 0 {
			sb.WriteString("\n")
		}

		// Format commit ID (use full ID for now as per standard git)
		sb.WriteString(fmt.Sprintf("commit %s\n", commit.ID))

		// Format author and timestamp
		sb.WriteString(fmt.Sprintf("Author: %s\n", commit.Author))
		sb.WriteString(fmt.Sprintf("Date:   %s\n\n", commit.Timestamp.Format(time.RFC1123)))

		// Format message with 4-space indent
		for _, line := range strings.Split(commit.Message, "\n") {
			sb.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}

	return sb.String()
}
