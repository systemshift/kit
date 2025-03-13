# Kit CLI

A command-line interface for Kit, the kernel-oriented version control system.

## Overview

This CLI tool provides a basic interface to interact with Kit repositories. Kit uses kernel methods from machine learning to reimagine version control, enabling:

- Sublinear-time integrity verification through Random Fourier Features
- Semantic understanding of code using embedding similarities
- Efficient content discovery with MinHash and Locality-Sensitive Hashing
- Optimized storage with kernel-based compression

## Building the CLI

From the project root, run:

```bash
# Build only the CLI tool
make cli

# Build and run the CLI
make cli-run

# Install the CLI tool globally
make install
```

The compiled binary will be available at `bin/kit`.

## Commands

### Initialize a Repository

```bash
kit init
```

Creates a new Kit repository in the current directory.

### Add Files

```bash
kit add <file> [<file2> ...]
```

Adds one or more files to the staging area, using the advanced kernel-based compression for storage.

### Check Status

```bash
kit status
```

Shows the current status of the repository, including:
- Files staged for commit
- Modified but not staged files
- Untracked files

### Verify Repository Integrity

```bash
kit verify
```

Verifies the integrity of the repository using Random Fourier Features (RFF), enabling sublinear-time repository verification.

### Help

```bash
kit help
```

Displays help information for the Kit CLI.

## Implementation Details

Each command leverages one or more of Kit's kernel methods:

- **init**: Sets up the repository structure with configurations for all kernels
- **add**: Uses the compression kernel for efficient storage of file contents
- **status**: Checks the working tree and staging area
- **verify**: Uses the integrity kernel to validate repository state

## Future Commands

The following commands are planned for future implementation:

- **commit**: Create commits with semantic understanding of changes
- **diff**: Semantically aware diff using the semantic kernel
- **log**: View commit history
- **search**: Find semantically similar code using the retrieval kernel
- **merge**: Intelligent merging of branches with semantic conflict resolution
