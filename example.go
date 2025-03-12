package main

import (
	"fmt"
	"os"

	"github.com/systemshift/kit/kernel"
)

// This example demonstrates the core concepts of Kit using simplified prototypes
// of the kernel implementations. In a full implementation, these would be
// integrated into a complete version control system.

func main() {
	fmt.Println("Kit: A Kernel-Oriented Git for Efficient Storage and Intelligent Monorepo Management")
	fmt.Println("=============================================================================")
	fmt.Println()

	// Example data
	data1 := []byte("This is a sample repository state with some content")
	data2 := []byte("This is a similar repository state with minor changes to content")
	data3 := []byte("This is a completely different repository with unrelated content")

	// Code example 1
	code1 := `
func calculateTotal(items []Item) int {
    total := 0
    for _, item := range items {
        total += item.Price
    }
    return total
}
`

	// Code example 2 (semantically similar to code1)
	code2 := `
func computeSum(products []Item) int {
    sum := 0
    for _, p := range products {
        sum += p.Price
    }
    return sum
}
`

	// Code example 3 (different from code1)
	code3 := `
func filterItems(items []Item, minPrice int) []Item {
    var result []Item
    for _, item := range items {
        if item.Price >= minPrice {
            result = append(result, item)
        }
    }
    return result
}
`

	// Initialize the kernels
	integrityKernel := demonstrateIntegrityKernel(data1, data2, data3)
	semanticKernel := demonstrateSemanticKernel(code1, code2, code3)
	retrievalKernel := demonstrateRetrievalKernel(code1, code2, code3)
	compressionKernel := demonstrateCompressionKernel(data1)

	// Demonstrate end-to-end workflow
	demonstrateWorkflow(integrityKernel, semanticKernel, retrievalKernel, compressionKernel)
}

// demonstrateIntegrityKernel shows how the integrity kernel can be used for fast verification
func demonstrateIntegrityKernel(data1, data2, data3 []byte) *kernel.IntegrityKernel {
	fmt.Println("1. Integrity Kernel Demonstration")
	fmt.Println("--------------------------------")

	// Create a new integrity kernel
	fmt.Println("Creating integrity kernel with Random Fourier Features (RFF)...")
	k := kernel.NewIntegrityKernel(128, 64, 0.1, 42)
	fmt.Println("Kernel initialized with 128 features and 64-dimension input space")
	fmt.Println()

	// Compute hashes
	fmt.Println("Computing RFF hashes for repository states...")
	hash1 := k.ComputeHash(data1)
	hash2 := k.ComputeHash(data2)
	hash3 := k.ComputeHash(data3)
	fmt.Printf("Hash lengths: %d bytes (vs 32 bytes for SHA-256)\n", len(hash1)*8)
	fmt.Printf("Size comparison - RFF hash (repo 1): %d bytes, RFF hash (repo 2): %d bytes\n", len(hash1)*8, len(hash2)*8)
	fmt.Printf("First few values of repo 3 hash: %.4f %.4f %.4f...\n", hash3[0], hash3[1], hash3[2])
	fmt.Println()

	// Verify integrity
	fmt.Println("Verifying repository integrity...")
	sim12, isSimilar12 := k.VerifyIntegrity(data1, data2, 0.8)
	sim13, isSimilar13 := k.VerifyIntegrity(data1, data3, 0.8)

	fmt.Printf("Similarity between repo 1 and 2: %.4f (similar: %v)\n", sim12, isSimilar12)
	fmt.Printf("Similarity between repo 1 and 3: %.4f (similar: %v)\n", sim13, isSimilar13)
	fmt.Println()

	return k
}

// demonstrateSemanticKernel shows how the semantic kernel can be used for intelligent diffs and merges
func demonstrateSemanticKernel(code1, code2, code3 string) *kernel.SemanticKernel {
	fmt.Println("2. Semantic Kernel Demonstration")
	fmt.Println("--------------------------------")

	// Create a new semantic kernel
	fmt.Println("Creating semantic kernel with cosine similarity...")
	k := kernel.NewSemanticKernel(128, 0.7)
	fmt.Println("Kernel initialized with 128-dimension embeddings and 0.7 similarity threshold")
	fmt.Println()

	// Compute semantic diff
	fmt.Println("Computing semantic diff between code versions...")
	sim12, isSimilar12 := k.SemanticDiff(code1, code2)
	sim13, isSimilar13 := k.SemanticDiff(code1, code3)

	fmt.Printf("Semantic similarity between code 1 and 2: %.4f (similar: %v)\n", sim12, isSimilar12)
	fmt.Printf("Semantic similarity between code 1 and 3: %.4f (similar: %v)\n", sim13, isSimilar13)
	fmt.Println()

	// Demonstrate semantic merge
	fmt.Println("Demonstrating semantic-aware merge...")
	fmt.Println("1. When code is semantically similar:")
	mergeResult1, success1 := k.SemanticMerge(code1, code2, kernel.SmartMerge)
	fmt.Printf("Merge success: %v\n", success1)
	fmt.Println("Merge result preview:")
	fmt.Println("---")
	fmt.Println(mergeResult1[:150] + "...")
	fmt.Println("---")
	fmt.Println()

	fmt.Println("2. When code is semantically different:")
	mergeResult2, success2 := k.SemanticMerge(code1, code3, kernel.SmartMerge)
	fmt.Printf("Merge success: %v\n", success2)
	fmt.Println("Merge result preview (conflict):")
	fmt.Println("---")
	fmt.Println(mergeResult2[:150] + "...")
	fmt.Println("---")
	fmt.Println()

	return k
}

// demonstrateRetrievalKernel shows how the retrieval kernel can be used for efficient code search
func demonstrateRetrievalKernel(code1, code2, code3 string) *kernel.RetrievalKernel {
	fmt.Println("3. Retrieval Kernel Demonstration")
	fmt.Println("--------------------------------")

	// Create a new retrieval kernel
	fmt.Println("Creating retrieval kernel with MinHash and LSH...")
	k := kernel.NewRetrievalKernel(100, 1000, 20, 42)
	fmt.Println("Kernel initialized with 100 permutations, 20 bands")
	fmt.Println()

	// Compute MinHash signatures
	fmt.Println("Computing MinHash signatures for code...")
	sig1 := k.MinHash(code1)
	sig2 := k.MinHash(code2)
	sig3 := k.MinHash(code3)
	fmt.Printf("Signature length: %d integers\n", len(sig1))
	fmt.Println()

	// Compute LSH signatures for fast retrieval
	fmt.Println("Computing LSH signatures for fast retrieval...")
	lsh1 := k.LSHSignature(sig1)
	lsh2 := k.LSHSignature(sig2)
	lsh3 := k.LSHSignature(sig3)
	fmt.Printf("LSH buckets: %d\n", len(lsh1))
	fmt.Printf("LSH signature examples - Code 1: %s, Code 2: %s\n", lsh1[0], lsh2[0])
	fmt.Printf("Any common buckets between unrelated code (1 & 3)?: %v\n", hasCommonBucket(lsh1, lsh3))
	fmt.Println()

	// Demonstrate similarity estimation
	fmt.Println("Estimating Jaccard similarity using MinHash...")
	estSim12 := k.ComputeJaccardSimilarity(sig1, sig2)
	estSim13 := k.ComputeJaccardSimilarity(sig1, sig3)

	fmt.Printf("Estimated similarity between code 1 and 2: %.4f\n", estSim12)
	fmt.Printf("Estimated similarity between code 1 and 3: %.4f\n", estSim13)
	fmt.Println()

	// Demonstrate fast similarity check with LSH
	fmt.Println("Testing fast similarity check with LSH...")
	isLikelySimilar12 := k.AreLikelySimilar(code1, code2)
	isLikelySimilar13 := k.AreLikelySimilar(code1, code3)

	fmt.Printf("Are code 1 and 2 likely similar? %v\n", isLikelySimilar12)
	fmt.Printf("Are code 1 and 3 likely similar? %v\n", isLikelySimilar13)
	fmt.Println()

	return k
}

// Helper function to check if two LSH signatures share any buckets
func hasCommonBucket(sig1, sig2 []string) bool {
	for _, b1 := range sig1 {
		for _, b2 := range sig2 {
			if b1 == b2 {
				return true
			}
		}
	}
	return false
}

// demonstrateCompressionKernel shows how the compression kernel can be used for efficient storage
func demonstrateCompressionKernel(data []byte) *kernel.CompressionKernel {
	fmt.Println("4. Compression Kernel Demonstration")
	fmt.Println("----------------------------------")

	// Create a new compression kernel
	fmt.Println("Creating compression kernel with kernel PCA...")
	k := kernel.NewCompressionKernel(64, 16, 0.1, 42, true, 6, 16)
	fmt.Println("Kernel initialized with 64-dimension embedding, 16 components")
	fmt.Println()

	// Create larger test data
	testData := make([]byte, 0, 10000)
	for i := 0; i < 100; i++ {
		testData = append(testData, data...)
	}

	// Compress and get statistics
	fmt.Println("Compressing test data...")
	compressed, stats, err := k.CompressWithStats(testData)
	if err != nil {
		fmt.Printf("Compression error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Original size: %d bytes\n", stats.OriginalSize)
	fmt.Printf("Compressed size: %d bytes\n", stats.CompressedSize)
	fmt.Printf("Compression ratio: %.2f:1\n", stats.CompressionRatio)
	fmt.Println()

	// Decompress
	fmt.Println("Decompressing data...")
	decompressed, err := k.Decompress(compressed)
	if err != nil {
		fmt.Printf("Decompression error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Decompressed size: %d bytes\n", len(decompressed))
	// Note: This is a lossy compression, so we don't expect perfect reconstruction
	fmt.Println("Note: Kernel PCA provides lossy compression - exact reconstruction is not expected")
	fmt.Println()

	return k
}

// demonstrateWorkflow shows how the kernels can work together in a version control system
func demonstrateWorkflow(
	integrityKernel *kernel.IntegrityKernel,
	semanticKernel *kernel.SemanticKernel,
	retrievalKernel *kernel.RetrievalKernel,
	compressionKernel *kernel.CompressionKernel) {

	fmt.Println("5. Kit Unified Workflow Demonstration")
	fmt.Println("-----------------------------------")
	fmt.Println("In a complete Kit implementation, these kernels would work together:")
	fmt.Println()

	fmt.Println("1. When adding files:")
	fmt.Println("   - Compression Kernel: Efficiently stores content")
	fmt.Println("   - Retrieval Kernel: Indexes content for semantic search")
	fmt.Println()

	fmt.Println("2. When committing changes:")
	fmt.Println("   - Integrity Kernel: Validates repository state")
	fmt.Println("   - Semantic Kernel: Identifies meaningful changes")
	fmt.Println()

	fmt.Println("3. When merging branches:")
	fmt.Println("   - Semantic Kernel: Intelligently resolves conflicts")
	fmt.Println("   - Integrity Kernel: Verifies merged state validity")
	fmt.Println()

	fmt.Println("4. When querying repository:")
	fmt.Println("   - Retrieval Kernel: Find semantically related code")
	fmt.Println("   - Compression Kernel: Efficiently retrieve content")
	fmt.Println()

	fmt.Println("This conceptual prototype demonstrates the theoretical foundation")
	fmt.Println("of Kit's kernel-oriented approach to version control.")
	fmt.Println()
}
