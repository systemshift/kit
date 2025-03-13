package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/systemshift/kit/pkg/repo"
)

const (
	// Version of the Kit tool
	Version = "0.1.0"
)

func main() {
	// Create a new flag set
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Kit v%s: A kernel-oriented version control system\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage: kit <command> [arguments]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  init             Initialize a new repository\n")
		fmt.Fprintf(os.Stderr, "  add <file>       Add file contents to the staging area\n")
		fmt.Fprintf(os.Stderr, "  status           Show the working tree status\n")
		fmt.Fprintf(os.Stderr, "  verify           Verify repository integrity using kernel methods\n")
		fmt.Fprintf(os.Stderr, "  help             Show help information for a command\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Parse flags
	flag.Parse()

	// Check if a command was provided
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get current working directory: %v\n", err)
		os.Exit(1)
	}

	// Dispatch command
	cmd := flag.Arg(0)
	switch cmd {
	case "init":
		initCmd(cwd)
	case "add":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: 'add' requires at least one file argument\n")
			os.Exit(1)
		}
		addCmd(cwd, flag.Args()[1:])
	case "status":
		statusCmd(cwd)
	case "verify":
		verifyCmd(cwd)
	case "help":
		flag.Usage()
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

// initCmd initializes a new repository
func initCmd(path string) {
	// Create a new repository
	repo, err := repo.NewRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create repository: %v\n", err)
		os.Exit(1)
	}

	// Initialize the repository
	err = repo.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize repository: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Initialized empty Kit repository in", filepath.Join(path, ".kit"))
}

// addCmd adds files to the staging area
func addCmd(path string, files []string) {
	// Check if this is a repository
	if !repo.IsRepository(path) {
		fmt.Fprintf(os.Stderr, "Error: Not a Kit repository\n")
		os.Exit(1)
	}

	// Create a repository instance
	r, err := repo.NewRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// Add each file
	for _, file := range files {
		err = r.Add(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to add file %s: %v\n", file, err)
			os.Exit(1)
		}
		fmt.Printf("Added %s\n", file)
	}
}

// statusCmd shows the repository status
func statusCmd(path string) {
	// Check if this is a repository
	if !repo.IsRepository(path) {
		fmt.Fprintf(os.Stderr, "Error: Not a Kit repository\n")
		os.Exit(1)
	}

	// Create a repository instance
	r, err := repo.NewRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// Get status
	status, err := r.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get repository status: %v\n", err)
		os.Exit(1)
	}

	// Print status
	fmt.Print(status)
}

// verifyCmd verifies the repository integrity
func verifyCmd(path string) {
	// Check if this is a repository
	if !repo.IsRepository(path) {
		fmt.Fprintf(os.Stderr, "Error: Not a Kit repository\n")
		os.Exit(1)
	}

	// Create a repository instance
	_, err := repo.NewRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// For now, just print a message since we haven't implemented full verification
	fmt.Println("Repository integrity verified using Random Fourier Features")
	fmt.Println("This is currently a placeholder for the full verification functionality")
}
