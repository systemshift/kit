package kernel

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
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
// This implementation uses AST-based features for better semantic understanding
func (k *SemanticKernel) CodeToEmbedding(code string) []float64 {
	embedding := make([]float64, k.EmbeddingDim)

	// Try to parse as Go code first
	if goEmbedding := k.extractGoFeatures(code); goEmbedding != nil {
		copy(embedding, goEmbedding)
	} else {
		// Fall back to generic text-based features
		k.extractTextFeatures(code, embedding)
	}

	// Add structural features
	k.addStructuralFeatures(code, embedding)

	// Normalize to unit length
	k.normalizeVector(embedding)
	return embedding
}

// extractGoFeatures extracts features from Go AST
func (k *SemanticKernel) extractGoFeatures(code string) []float64 {
	embedding := make([]float64, k.EmbeddingDim)

	// Parse Go code
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return nil // Not valid Go code
	}

	// Extract AST-based features
	features := make(map[string]int)

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		nodeType := fmt.Sprintf("AST_%T", n)
		features[nodeType]++

		// Extract specific patterns
		switch v := n.(type) {
		case *ast.FuncDecl:
			if v.Name != nil {
				features["FUNC_"+v.Name.Name]++
			}
		case *ast.CallExpr:
			features["CALL"]++
		case *ast.IfStmt:
			features["IF"]++
		case *ast.ForStmt:
			features["FOR"]++
		case *ast.AssignStmt:
			features["ASSIGN"]++
		}

		return true
	})

	// Map features to embedding
	for feature, count := range features {
		k.addFeature(embedding, feature, float64(count))
	}

	return embedding
}

// extractTextFeatures extracts features from raw text
func (k *SemanticKernel) extractTextFeatures(code string, embedding []float64) {
	// Tokenize by common programming patterns
	patterns := []string{
		`\b(func|function|def)\b`,      // Function definitions
		`\b(if|else|elif)\b`,           // Conditionals
		`\b(for|while|do)\b`,           // Loops
		`\b(class|struct|type)\b`,      // Type definitions
		`\b(import|include|require)\b`, // Imports
		`\b(return|yield)\b`,           // Returns
		`[a-zA-Z_][a-zA-Z0-9_]*`,      // Identifiers
		`[0-9]+`,                       // Numbers
		`"[^"]*"`,                      // Strings
		`//.*|/\*.*?\*/`,               // Comments
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			k.addFeature(embedding, "TEXT_"+strings.ToUpper(match), 1.0)
		}
	}
}

// addStructuralFeatures adds features based on code structure
func (k *SemanticKernel) addStructuralFeatures(code string, embedding []float64) {
	lines := strings.Split(code, "\n")

	// Line count feature
	k.addFeature(embedding, "LINE_COUNT", math.Log1p(float64(len(lines))))

	// Indentation patterns
	indentLevels := make(map[int]int)
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if len(line) == 0 {
			continue
		}

		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 4 // Treat tab as 4 spaces
			} else {
				break
			}
		}
		indentLevels[indent/4]++ // Normalize to indent levels
	}

	for level, count := range indentLevels {
		feature := fmt.Sprintf("INDENT_L%d", level)
		k.addFeature(embedding, feature, float64(count))
	}

	// Complexity indicators
	complexityWords := []string{"if", "else", "for", "while", "switch", "case", "try", "catch"}
	for _, word := range complexityWords {
		count := strings.Count(strings.ToLower(code), word)
		k.addFeature(embedding, "COMPLEXITY_"+strings.ToUpper(word), float64(count))
	}
}

// addFeature adds a feature to the embedding vector
func (k *SemanticKernel) addFeature(embedding []float64, feature string, weight float64) {
	h := sha256.Sum256([]byte(feature))
	idx := int(binary.BigEndian.Uint32(h[:4]) % uint32(k.EmbeddingDim))
	embedding[idx] += weight
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
