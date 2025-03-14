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
		fmt.Fprintf(os.Stderr, "  commit           Record changes to the repository\n")
		fmt.Fprintf(os.Stderr, "  branch [name]    List or create branches\n")
		fmt.Fprintf(os.Stderr, "  checkout <name>  Switch branches\n")
		fmt.Fprintf(os.Stderr, "  diff [options]   Show changes between commits or working directory\n")
		fmt.Fprintf(os.Stderr, "  merge [options]  Merge changes from another branch\n")
		fmt.Fprintf(os.Stderr, "  log              Show commit logs\n")
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
	case "commit":
		message := ""

		// Check for -m flag
		fs := flag.NewFlagSet("commit", flag.ExitOnError)
		fs.StringVar(&message, "m", "", "Commit message")

		// Parse the remaining arguments
		err := fs.Parse(flag.Args()[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to parse commit arguments: %v\n", err)
			os.Exit(1)
		}

		// Check if a message was provided
		if message == "" {
			fmt.Fprintf(os.Stderr, "Error: Commit message is required (use -m \"message\")\n")
			os.Exit(1)
		}

		commitCmd(cwd, message)
	case "branch":
		branchCmd(cwd, flag.Args()[1:])
	case "checkout":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: 'checkout' requires a branch name\n")
			os.Exit(1)
		}
		checkoutCmd(cwd, flag.Args()[1])
	case "diff":
		diffCmd(cwd, flag.Args()[1:])
	case "merge":
		mergeCmd(cwd, flag.Args()[1:])
	case "status":
		statusCmd(cwd)
	case "log":
		logCmd(cwd)
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
	r, err := repo.NewRepository(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// Perform repository verification
	result, err := r.VerifyIntegrity()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to verify repository integrity: %v\n", err)
		os.Exit(1)
	}

	// Print verification summary
	fmt.Println(result.Summary)

	// Set exit code based on verification status
	if !result.Status {
		os.Exit(2) // Non-zero exit code for verification failure
	}
}

// commitCmd records changes to the repository
func commitCmd(path string, message string) {
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

	// Commit changes
	commitID, err := r.Commit(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to commit changes: %v\n", err)
		os.Exit(1)
	}

	// Print commit ID (abbreviated)
	shortID := commitID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	fmt.Printf("[%s] %s\n", shortID, message)
}

// logCmd shows the commit log
func logCmd(path string) {
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

	// Get commit log
	log, err := r.Log()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get commit log: %v\n", err)
		os.Exit(1)
	}

	// Check if there are any commits
	if len(log) == 0 {
		fmt.Println("No commits yet")
		return
	}

	// Format and print log
	formattedLog := repo.FormatLog(log)
	fmt.Println(formattedLog)
}

// branchCmd handles branch operations (create/list)
func branchCmd(path string, args []string) {
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

	// Check if branch name was provided
	if len(args) > 0 {
		// Create a new branch
		err := r.CreateBranch(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create branch: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created branch '%s'\n", args[0])
		return
	}

	// List branches if no name provided
	branches, err := r.ListBranches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list branches: %v\n", err)
		os.Exit(1)
	}

	// Check if there are any branches
	if len(branches) == 0 {
		fmt.Println("No branches yet")
		return
	}

	// Print branches
	currentBranch, _ := r.GetCurrentBranch()
	for _, branch := range branches {
		if branch.Name == currentBranch {
			fmt.Printf("* %s\n", branch.Name)
		} else {
			fmt.Printf("  %s\n", branch.Name)
		}
	}
}

// checkoutCmd switches branches
func checkoutCmd(path string, branchName string) {
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

	// Check if current branch is already the requested branch
	currentBranch, err := r.GetCurrentBranch()
	if err == nil && currentBranch == branchName {
		fmt.Printf("Already on branch '%s'\n", branchName)
		return
	}

	// Switch to the branch
	err = r.CheckoutBranch(branchName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to checkout branch: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Switched to branch '%s'\n", branchName)
}

// mergeCmd merges changes from another branch
func mergeCmd(path string, args []string) {
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

	// Parse options
	fs := flag.NewFlagSet("merge", flag.ExitOnError)

	// Merge options
	noCommit := fs.Bool("no-commit", false, "Do not create a commit after merge")
	message := fs.String("m", "", "Custom merge commit message")
	semantic := fs.Bool("semantic", true, "Use semantic merge for conflicts")
	strategyStr := fs.String("strategy", "auto", "Merge strategy: auto, ours, theirs, manual")

	// Parse args
	err = fs.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to parse merge arguments: %v\n", err)
		os.Exit(1)
	}

	// Get branch to merge
	remainingArgs := fs.Args()
	if len(remainingArgs) != 1 {
		fmt.Fprintf(os.Stderr, "Error: 'merge' requires exactly one branch name\n")
		fmt.Fprintf(os.Stderr, "Usage: kit merge [options] <branch>\n")
		os.Exit(1)
	}

	branchName := remainingArgs[0]

	// Convert strategy string to enum
	var strategy repo.MergeStrategy
	switch *strategyStr {
	case "auto":
		strategy = repo.AutoMerge
	case "ours":
		strategy = repo.Ours
	case "theirs":
		strategy = repo.Theirs
	case "manual":
		strategy = repo.Manual
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown merge strategy '%s'\n", *strategyStr)
		os.Exit(1)
	}

	// Create merge options
	options := &repo.MergeOptions{
		Strategy:    strategy,
		NoCommit:    *noCommit,
		Message:     *message,
		UseSemantic: *semantic,
	}

	// Perform merge
	result, err := r.Merge(branchName, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to merge: %v\n", err)
		os.Exit(1)
	}

	// Show result
	if result.FastForward {
		fmt.Printf("Fast-forward merge of '%s' into current branch\n", branchName)
	} else if result.Success {
		if len(result.Conflicts) > 0 {
			fmt.Printf("Merge completed with %d conflicts resolved\n", len(result.Conflicts))
		} else {
			fmt.Println("Merge completed successfully")
		}
	} else {
		fmt.Printf("Merge resulted in %d conflicts\n", len(result.Conflicts))
		fmt.Println("Fix conflicts and then commit the result")
	}
}

// diffCmd shows differences between commits or working directory
func diffCmd(path string, args []string) {
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

	// Parse options
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	semantic := fs.Bool("semantic", false, "Use semantic diff")
	context := fs.Int("context", 3, "Number of context lines")

	// Parse args (ignoring unknown flags, which might be commit IDs)
	err = fs.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to parse diff arguments: %v\n", err)
		os.Exit(1)
	}

	// Create diff options
	options := &repo.DiffOptions{
		ContextLines: *context,
		Semantic:     *semantic,
	}

	// Get remaining args (for commit IDs)
	remainingArgs := fs.Args()
	var commitA, commitB string

	// Handle different diff modes based on number of arguments
	switch len(remainingArgs) {
	case 0:
		// No commits specified, diff working tree vs HEAD
		commitA = "" // HEAD
		commitB = "" // Working directory
	case 1:
		// One commit specified, diff working tree vs that commit
		commitA = remainingArgs[0]
		commitB = "" // Working directory
	case 2:
		// Two commits specified, diff between those commits
		commitA = remainingArgs[0]
		commitB = remainingArgs[1]
	default:
		fmt.Fprintf(os.Stderr, "Error: Too many arguments for diff\n")
		os.Exit(1)
	}

	// Perform the diff
	diff, err := r.Diff(commitA, commitB, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to perform diff: %v\n", err)
		os.Exit(1)
	}

	// Format and print the diff
	output := repo.FormatDiff(diff)
	if output == "" {
		fmt.Println("No differences")
	} else {
		fmt.Print(output)
	}
}
