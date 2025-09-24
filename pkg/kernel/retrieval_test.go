package kernel

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestNewRetrievalKernel(t *testing.T) {
	kernel := NewRetrievalKernel(100, 10000, 10, 42)

	if kernel.NumPermutations != 100 {
		t.Errorf("Expected 100 permutations, got %d", kernel.NumPermutations)
	}
	if kernel.NumBands != 10 {
		t.Errorf("Expected 10 bands, got %d", kernel.NumBands)
	}
	if kernel.NumRows != 10 {
		t.Errorf("Expected 10 rows per band, got %d", kernel.NumRows)
	}
	if kernel.Seed != 42 {
		t.Errorf("Expected seed 42, got %d", kernel.Seed)
	}

	// Check that permutations are properly initialized
	if len(kernel.Permutations) != 100 {
		t.Errorf("Expected 100 permutation functions, got %d", len(kernel.Permutations))
	}

	for i, perm := range kernel.Permutations {
		if len(perm) != 2 {
			t.Errorf("Permutation %d should have 2 coefficients, got %d", i, len(perm))
		}
		// Check that first coefficient is odd (better distribution)
		if perm[0]%2 == 0 {
			t.Errorf("Permutation %d first coefficient should be odd, got %d", i, perm[0])
		}
	}

	// Check hash bands
	if len(kernel.HashBands) != 10 {
		t.Errorf("Expected 10 hash bands, got %d", len(kernel.HashBands))
	}
}

func TestDocumentToShingles(t *testing.T) {
	kernel := NewRetrievalKernel(50, 10000, 5, 42)

	doc := `package main

import "fmt"

func hello() {
    fmt.Println("Hello, World!")
}

func main() {
    hello()
}`

	shingles := kernel.documentToShingles(doc)

	if len(shingles) == 0 {
		t.Error("Expected non-empty shingles")
	}

	// Check that we have different types of shingles
	hasChar := false
	hasToken := false
	hasLine := false

	for _, shingle := range shingles {
		if strings.HasPrefix(shingle, "CHAR:") {
			hasChar = true
		} else if strings.HasPrefix(shingle, "TOKEN:") {
			hasToken = true
		} else if strings.HasPrefix(shingle, "LINE:") {
			hasLine = true
		}
	}

	if !hasChar {
		t.Error("Expected character-level shingles")
	}
	if !hasToken {
		t.Error("Expected token-level shingles")
	}
	if !hasLine {
		t.Error("Expected line-level shingles")
	}

	// Test determinism
	shingles2 := kernel.documentToShingles(doc)
	if len(shingles) != len(shingles2) {
		t.Errorf("Shingles not deterministic: %d != %d", len(shingles), len(shingles2))
	}

	for i, shingle := range shingles {
		if shingle != shingles2[i] {
			t.Errorf("Shingle %d differs: %s != %s", i, shingle, shingles2[i])
		}
	}
}

func TestGetCharacterShingles(t *testing.T) {
	kernel := NewRetrievalKernel(50, 10000, 5, 42)

	text := "hello world"
	shingles := kernel.getCharacterShingles(text, 3)

	expectedCount := len(text) - 3 + 1
	if len(shingles) != expectedCount {
		t.Errorf("Expected %d character shingles, got %d", expectedCount, len(shingles))
	}

	// Check first and last shingles
	if !strings.HasSuffix(shingles[0], "hel") {
		t.Errorf("First shingle should end with 'hel', got %s", shingles[0])
	}
	if !strings.HasSuffix(shingles[len(shingles)-1], "rld") {
		t.Errorf("Last shingle should end with 'rld', got %s", shingles[len(shingles)-1])
	}

	// Test short text
	shortText := "hi"
	shortShingles := kernel.getCharacterShingles(shortText, 5)
	if len(shortShingles) != 1 || shortShingles[0] != "hi" {
		t.Errorf("Expected single shingle 'hi' for short text, got %v", shortShingles)
	}
}

func TestGetTokenShingles(t *testing.T) {
	kernel := NewRetrievalKernel(50, 10000, 5, 42)

	text := "func main() { fmt.Println(hello) }"
	shingles := kernel.getTokenShingles(text, 2)

	if len(shingles) == 0 {
		t.Error("Expected non-empty token shingles")
	}

	// Check that all shingles start with "TOKEN:"
	for _, shingle := range shingles {
		if !strings.HasPrefix(shingle, "TOKEN:") {
			t.Errorf("Token shingle should start with 'TOKEN:', got %s", shingle)
		}
	}

	// Test that we get reasonable number of shingles
	// With delimiters splitting, we should get multiple tokens
	if len(shingles) < 3 {
		t.Errorf("Expected at least 3 token shingles, got %d", len(shingles))
	}
}

func TestGetLineShingles(t *testing.T) {
	kernel := NewRetrievalKernel(50, 10000, 5, 42)

	text := `func main() {
    fmt.Println("Hello")
    return
}`

	shingles := kernel.getLineShingles(text)

	if len(shingles) == 0 {
		t.Error("Expected non-empty line shingles")
	}

	hasLine := false
	hasLines := false

	for _, shingle := range shingles {
		if strings.HasPrefix(shingle, "LINE:") {
			hasLine = true
		} else if strings.HasPrefix(shingle, "LINES:") {
			hasLines = true
		}
	}

	if !hasLine {
		t.Error("Expected individual line shingles")
	}
	if !hasLines {
		t.Error("Expected paired line shingles")
	}
}

func TestMinHash(t *testing.T) {
	kernel := NewRetrievalKernel(100, 10000, 10, 42)

	doc1 := "The quick brown fox jumps over the lazy dog"
	doc2 := "The quick brown fox jumps over the lazy dog"
	doc3 := "A completely different sentence with no overlap"

	sig1 := kernel.MinHash(doc1)
	sig2 := kernel.MinHash(doc2)
	sig3 := kernel.MinHash(doc3)

	// Check signature length
	if len(sig1) != 100 {
		t.Errorf("Expected signature length 100, got %d", len(sig1))
	}

	// Test determinism
	if len(sig1) != len(sig2) {
		t.Errorf("Signatures have different lengths: %d != %d", len(sig1), len(sig2))
	}

	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Errorf("Identical documents produced different signatures at position %d", i)
		}
	}

	// Test that different documents produce different signatures
	same := true
	for i := range sig1 {
		if sig1[i] != sig3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different documents produced identical signatures")
	}
}

func TestLSHSignature(t *testing.T) {
	kernel := NewRetrievalKernel(100, 10000, 10, 42)

	minHashSig := make([]int, 100)
	for i := range minHashSig {
		minHashSig[i] = i * 1000
	}

	lshSig := kernel.LSHSignature(minHashSig)

	if len(lshSig) != 10 {
		t.Errorf("Expected LSH signature length 10, got %d", len(lshSig))
	}

	// Check that signatures are properly formatted
	for i, band := range lshSig {
		if !strings.HasPrefix(band, strconv.Itoa(i)+":") {
			t.Errorf("Band %d should start with '%d:', got %s", i, i, band)
		}
	}

	// Test determinism
	lshSig2 := kernel.LSHSignature(minHashSig)
	for i, band := range lshSig {
		if band != lshSig2[i] {
			t.Errorf("LSH signatures not deterministic at band %d: %s != %s", i, band, lshSig2[i])
		}
	}
}

func TestComputeJaccardSimilarity(t *testing.T) {
	kernel := NewRetrievalKernel(100, 10000, 10, 42)

	// Test identical signatures
	sig1 := []int{1, 2, 3, 4, 5}
	sig2 := []int{1, 2, 3, 4, 5}
	sim := kernel.ComputeJaccardSimilarity(sig1, sig2)
	if sim != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical signatures, got %f", sim)
	}

	// Test completely different signatures
	sig3 := []int{6, 7, 8, 9, 10}
	sim2 := kernel.ComputeJaccardSimilarity(sig1, sig3)
	if sim2 != 0.0 {
		t.Errorf("Expected similarity 0.0 for different signatures, got %f", sim2)
	}

	// Test partial overlap
	sig4 := []int{1, 2, 8, 9, 10}
	sim3 := kernel.ComputeJaccardSimilarity(sig1, sig4)
	expected := 2.0 / 5.0 // 2 matches out of 5 elements
	if math.Abs(sim3-expected) > 1e-6 {
		t.Errorf("Expected similarity %f for partial overlap, got %f", expected, sim3)
	}

	// Test different lengths
	sig5 := []int{1, 2, 3}
	sim4 := kernel.ComputeJaccardSimilarity(sig1, sig5)
	if sim4 != 0.0 {
		t.Errorf("Expected similarity 0.0 for different length signatures, got %f", sim4)
	}
}

func TestEstimateSimilarity(t *testing.T) {
	kernel := NewRetrievalKernel(200, 10000, 20, 42)

	doc1 := "The quick brown fox jumps over the lazy dog"
	doc2 := "The quick brown fox jumps over the lazy dog"
	doc3 := "A quick brown fox leaps over a lazy dog"
	doc4 := "Completely different content with no shared words"

	// Test identical documents
	sim12 := kernel.EstimateSimilarity(doc1, doc2)
	if sim12 != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical documents, got %f", sim12)
	}

	// Test similar documents (MinHash is approximate)
	sim13 := kernel.EstimateSimilarity(doc1, doc3)
	if sim13 <= 0.2 {
		t.Errorf("Expected reasonable similarity for similar documents, got %f", sim13)
	}

	// Test different documents
	sim14 := kernel.EstimateSimilarity(doc1, doc4)
	if sim14 >= sim13 {
		t.Errorf("Expected lower similarity for different documents: %f >= %f", sim14, sim13)
	}

	// Test symmetry
	sim31 := kernel.EstimateSimilarity(doc3, doc1)
	if math.Abs(sim13-sim31) > 1e-6 {
		t.Errorf("Similarity not symmetric: %f != %f", sim13, sim31)
	}
}

func TestAreLikelySimilar(t *testing.T) {
	kernel := NewRetrievalKernel(100, 10000, 10, 42)

	doc1 := "function calculateSum(a, b) { return a + b; }"
	doc2 := "function calculateSum(a, b) { return a + b; }"
	doc3 := "function computeSum(x, y) { return x + y; }"
	doc4 := "print('This is completely different Python code')"

	// Test identical documents
	similar12 := kernel.AreLikelySimilar(doc1, doc2)
	if !similar12 {
		t.Error("Expected identical documents to be likely similar")
	}

	// Test similar documents (might or might not be caught by LSH)
	_ = kernel.AreLikelySimilar(doc1, doc3)
	// Note: We can't guarantee this will be true due to LSH probabilistic nature
	// But we test that the function runs without error

	// Test very different documents
	similar14 := kernel.AreLikelySimilar(doc1, doc4)
	// Similarly, we can't guarantee this will be false, but it's likely
	_ = similar14 // Use the result to avoid compiler warning

	// Test self-similarity
	similar11 := kernel.AreLikelySimilar(doc1, doc1)
	if !similar11 {
		t.Error("Expected document to be similar to itself")
	}
}

func TestJaccardProperties(t *testing.T) {
	kernel := NewRetrievalKernel(200, 10000, 20, 42)

	// Test Jaccard similarity properties with actual text
	text1 := "hello world foo bar"
	text2 := "hello world baz qux"
	text3 := "foo bar baz qux"
	text4 := "completely different text"

	sim12 := kernel.EstimateSimilarity(text1, text2)
	sim13 := kernel.EstimateSimilarity(text1, text3)
	sim14 := kernel.EstimateSimilarity(text1, text4)

	// Text1 and text2 share "hello world", text1 and text3 share "foo bar"
	// So sim12 and sim13 should be similar and both > sim14
	if sim14 >= sim12 || sim14 >= sim13 {
		t.Errorf("Expected lower similarity for different text: sim14=%f, sim12=%f, sim13=%f", sim14, sim12, sim13)
	}

	// Test self-similarity
	sim11 := kernel.EstimateSimilarity(text1, text1)
	if sim11 != 1.0 {
		t.Errorf("Expected self-similarity 1.0, got %f", sim11)
	}
}

// Benchmark tests
func BenchmarkMinHash(b *testing.B) {
	kernel := NewRetrievalKernel(200, 100000, 20, 42)
	doc := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.MinHash(doc)
	}
}

func BenchmarkLSHSignature(b *testing.B) {
	kernel := NewRetrievalKernel(200, 100000, 20, 42)
	minHashSig := make([]int, 200)
	for i := range minHashSig {
		minHashSig[i] = i * 1000
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.LSHSignature(minHashSig)
	}
}

func BenchmarkEstimateSimilarity(b *testing.B) {
	kernel := NewRetrievalKernel(200, 100000, 20, 42)
	doc1 := strings.Repeat("package main import fmt func hello ", 50)
	doc2 := strings.Repeat("package main import fmt func world ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.EstimateSimilarity(doc1, doc2)
	}
}

func BenchmarkAreLikelySimilar(b *testing.B) {
	kernel := NewRetrievalKernel(200, 100000, 20, 42)
	doc1 := strings.Repeat("function calculate(a, b) { return a + b; } ", 20)
	doc2 := strings.Repeat("function compute(x, y) { return x + y; } ", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.AreLikelySimilar(doc1, doc2)
	}
}