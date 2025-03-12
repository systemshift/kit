package kernel

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// SemanticKernel implements a cosine similarity based kernel
// for semantic diffing, merging, and conflict resolution
type SemanticKernel struct {
	EmbeddingDim int     // Dimensionality of semantic embeddings
	MinimumScore float64 // Threshold for considering content semantically similar
}

// NewSemanticKernel creates a new semantic kernel with the specified parameters
func NewSemanticKernel(embeddingDim int, minimumScore float64) *SemanticKernel {
	return &SemanticKernel{
		EmbeddingDim: embeddingDim,
		MinimumScore: minimumScore,
	}
}

// CodeToEmbedding converts source code to a semantic embedding
// Note: In a full implementation, this would use an actual code embedding model
// For this prototype, we'll use a simplified tokenization and hashing approach
func (k *SemanticKernel) CodeToEmbedding(code string) []float64 {
	// Create an embedding vector
	embedding := make([]float64, k.EmbeddingDim)

	// Simplified approach: tokenize code, compute token statistics
	// A real implementation would use AST parsing and ML-based embeddings

	// Split code into tokens (simplified)
	tokens := strings.Fields(code)
	tokenCounts := make(map[string]int)

	// Count token occurrences
	for _, token := range tokens {
		tokenCounts[token]++
	}

	// Generate embedding based on token statistics
	// This is a very simplified approach - production would use ML models
	for token, count := range tokenCounts {
		// Hash the token to get a deterministic index
		h := sha256.Sum256([]byte(token))
		idx := int(binary.BigEndian.Uint32(h[:4]) % uint32(k.EmbeddingDim))

		// Add token influence to embedding (weighted by count)
		embedding[idx] += float64(count)
	}

	// Normalize embedding to unit length (for cosine similarity)
	k.normalizeVector(embedding)

	return embedding
}

// normalizeVector normalizes a vector to unit length
func (k *SemanticKernel) normalizeVector(vector []float64) {
	// Compute L2 norm
	sum := 0.0
	for _, val := range vector {
		sum += val * val
	}
	norm := math.Sqrt(sum)

	// Normalize (avoid division by zero)
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}
}

// CosineSimilarity computes the cosine similarity between two embedding vectors
func (k *SemanticKernel) CosineSimilarity(embedding1, embedding2 []float64) float64 {
	// Assume embeddings are already normalized to unit length
	// Cosine similarity is just the dot product of normalized vectors
	dotProduct := 0.0
	for i := 0; i < k.EmbeddingDim; i++ {
		dotProduct += embedding1[i] * embedding2[i]
	}

	// Clamp to [-1, 1] to handle floating point errors
	if dotProduct > 1.0 {
		dotProduct = 1.0
	} else if dotProduct < -1.0 {
		dotProduct = -1.0
	}

	return dotProduct
}

// SemanticDiff computes the semantic difference between two code snippets
func (k *SemanticKernel) SemanticDiff(code1, code2 string) (float64, bool) {
	// Generate embeddings
	embedding1 := k.CodeToEmbedding(code1)
	embedding2 := k.CodeToEmbedding(code2)

	// Compute similarity
	similarity := k.CosineSimilarity(embedding1, embedding2)

	// Check if code is semantically similar (above threshold)
	isSimilar := similarity >= k.MinimumScore

	return similarity, isSimilar
}

// MergeStrategy represents the approach for merging semantically different code
type MergeStrategy int

const (
	KeepBase MergeStrategy = iota
	KeepIncoming
	SmartMerge
)

// SemanticMerge attempts to merge two code snippets based on semantic meaning
func (k *SemanticKernel) SemanticMerge(baseCode, incomingCode string, strategy MergeStrategy) (string, bool) {
	// Get semantic similarity
	similarity, isSimilar := k.SemanticDiff(baseCode, incomingCode)

	// If semantically equivalent, no conflict
	if isSimilar {
		// Return the more verbose or efficient version (in this prototype, we simply keep incoming)
		return incomingCode, true
	}

	// Handle based on merge strategy
	switch strategy {
	case KeepBase:
		return baseCode, false
	case KeepIncoming:
		return incomingCode, false
	case SmartMerge:
		// In a real implementation, this would use more sophisticated techniques
		// to merge code based on AST and semantic understanding
		// For this prototype, we'll do a simplified approach

		// If similarity is still reasonably high, attempt a merge
		if similarity > k.MinimumScore*0.7 {
			// This is a placeholder for smart merging logic
			// A real implementation would do AST-based merging
			return "// AUTO-MERGED (Semantic Score: " +
				formatFloat(similarity) + ")\n" +
				incomingCode, true
		}

		// Otherwise indicate conflict
		return "// SEMANTIC CONFLICT (Score: " + formatFloat(similarity) + ")\n" +
			"// BASE CODE:\n" + baseCode + "\n\n" +
			"// INCOMING CODE:\n" + incomingCode, false
	}

	// Default - indicate conflict
	return "", false
}

// Helper function for formatting float value
func formatFloat(val float64) string {
	// Round to 2 decimal places and convert to string
	s := fmt.Sprintf("%.2f", math.Round(val*100)/100)
	// Trim trailing zeros and decimal point if no fractional part
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}
