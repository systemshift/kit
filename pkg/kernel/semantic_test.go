package kernel

import (
	"math"
	"strings"
	"testing"
)

func TestNewSemanticKernel(t *testing.T) {
	kernel := NewSemanticKernel(256, 0.75)

	if kernel.EmbeddingDim != 256 {
		t.Errorf("Expected embedding dimension 256, got %d", kernel.EmbeddingDim)
	}
	if kernel.MinimumScore != 0.75 {
		t.Errorf("Expected minimum score 0.75, got %f", kernel.MinimumScore)
	}
}

func TestCodeToEmbedding(t *testing.T) {
	kernel := NewSemanticKernel(128, 0.7)

	goCode := `package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}`

	embedding := kernel.CodeToEmbedding(goCode)

	if len(embedding) != 128 {
		t.Errorf("Expected embedding length 128, got %d", len(embedding))
	}

	// Check normalization - should be unit vector
	norm := 0.0
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if math.Abs(norm-1.0) > 1e-6 {
		t.Errorf("Expected unit vector (norm=1.0), got norm=%f", norm)
	}

	// Test determinism
	embedding2 := kernel.CodeToEmbedding(goCode)
	for i, val := range embedding {
		if math.Abs(val-embedding2[i]) > 1e-10 {
			t.Errorf("Embeddings are not deterministic at index %d: %f != %f", i, val, embedding2[i])
		}
	}
}

func TestExtractGoFeatures(t *testing.T) {
	kernel := NewSemanticKernel(64, 0.7)

	validGoCode := `package main

import "fmt"

func hello() string {
    return "Hello"
}

func main() {
    if true {
        fmt.Println(hello())
    }
}`

	invalidCode := "This is not valid Go code!"

	// Test valid Go code
	embedding1 := kernel.extractGoFeatures(validGoCode)
	if embedding1 == nil {
		t.Error("Expected valid Go code to return non-nil embedding")
	}
	if len(embedding1) != 64 {
		t.Errorf("Expected embedding length 64, got %d", len(embedding1))
	}

	// Test invalid code
	embedding2 := kernel.extractGoFeatures(invalidCode)
	if embedding2 != nil {
		t.Error("Expected invalid Go code to return nil embedding")
	}

	// Test that different valid Go codes produce different embeddings
	anotherGoCode := `package main

func add(a, b int) int {
    return a + b
}

func main() {
    result := add(1, 2)
    _ = result
}`

	embedding3 := kernel.extractGoFeatures(anotherGoCode)
	if embedding3 == nil {
		t.Error("Expected valid Go code to return non-nil embedding")
	}

	// Check they're different
	same := true
	for i, val := range embedding1 {
		if math.Abs(val-embedding3[i]) > 1e-10 {
			same = false
			break
		}
	}
	if same {
		t.Error("Different Go codes produced identical embeddings")
	}
}

func TestExtractTextFeatures(t *testing.T) {
	kernel := NewSemanticKernel(64, 0.7)
	embedding := make([]float64, 64)

	code := `function hello() {
    if (true) {
        console.log("Hello");
        return "world";
    }
}`

	kernel.extractTextFeatures(code, embedding)

	// Check that some features were extracted (embedding shouldn't be all zeros)
	nonZero := false
	for _, val := range embedding {
		if val != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("Expected text features to produce non-zero embedding")
	}
}

func TestAddStructuralFeatures(t *testing.T) {
	kernel := NewSemanticKernel(64, 0.7)
	embedding := make([]float64, 64)

	code := `    if condition:
        print("indented")
        if nested:
            print("more indented")
    else:
        print("else branch")`

	kernel.addStructuralFeatures(code, embedding)

	// Check that features were added
	nonZero := false
	for _, val := range embedding {
		if val != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("Expected structural features to produce non-zero embedding")
	}
}

func TestCosineSimilarity(t *testing.T) {
	kernel := NewSemanticKernel(4, 0.7)

	// Test vectors
	vec1 := []float64{1, 0, 0, 0}
	vec2 := []float64{0, 1, 0, 0}
	vec3 := []float64{1, 0, 0, 0}
	vec4 := []float64{0.7071, 0.7071, 0, 0} // 45-degree angle with vec1

	// Test orthogonal vectors
	sim12 := kernel.CosineSimilarity(vec1, vec2)
	if math.Abs(sim12) > 1e-6 {
		t.Errorf("Expected orthogonal vectors to have similarity ~0, got %f", sim12)
	}

	// Test identical vectors
	sim13 := kernel.CosineSimilarity(vec1, vec3)
	if math.Abs(sim13-1.0) > 1e-6 {
		t.Errorf("Expected identical vectors to have similarity 1.0, got %f", sim13)
	}

	// Test 45-degree angle
	sim14 := kernel.CosineSimilarity(vec1, vec4)
	expected := math.Sqrt(2) / 2 // cos(45°)
	if math.Abs(sim14-expected) > 1e-4 {
		t.Errorf("Expected similarity %f for 45-degree angle, got %f", expected, sim14)
	}

	// Test symmetry
	sim41 := kernel.CosineSimilarity(vec4, vec1)
	if math.Abs(sim14-sim41) > 1e-10 {
		t.Errorf("Cosine similarity is not symmetric: %f != %f", sim14, sim41)
	}
}

func TestSemanticDiff(t *testing.T) {
	kernel := NewSemanticKernel(128, 0.8)

	// Test similar Go functions
	code1 := `func add(a, b int) int {
    return a + b
}`

	code2 := `func sum(x, y int) int {
    return x + y
}`

	similarity, isSimilar := kernel.SemanticDiff(code1, code2)

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0,1], got %f", similarity)
	}

	if similarity > 0.8 && !isSimilar {
		t.Errorf("High similarity %f should be marked as similar", similarity)
	}

	// Test very different code
	code3 := `import json
data = {"key": "value"}
print(json.dumps(data))`

	similarity2, _ := kernel.SemanticDiff(code1, code3)

	if similarity2 >= similarity {
		t.Errorf("Different languages should have lower similarity: %f >= %f", similarity2, similarity)
	}

	// Test identical code
	similarity3, isSimilar3 := kernel.SemanticDiff(code1, code1)
	if math.Abs(similarity3-1.0) > 1e-6 {
		t.Errorf("Identical code should have similarity 1.0, got %f", similarity3)
	}
	if !isSimilar3 {
		t.Error("Identical code should be marked as similar")
	}
}

func TestSemanticMerge(t *testing.T) {
	kernel := NewSemanticKernel(128, 0.8)

	// Test merging similar functions
	baseCode := `func multiply(a, b int) int {
    return a * b
}`

	incomingCode := `func product(x, y int) int {
    return x * y
}`

	merged, _ := kernel.SemanticMerge(baseCode, incomingCode, SmartMerge)

	if !strings.Contains(merged, incomingCode) {
		t.Error("Expected merged code to contain incoming code")
	}

	// Test merging very different code
	differentCode := `print("This is Python code")`

	merged2, success2 := kernel.SemanticMerge(baseCode, differentCode, SmartMerge)

	if success2 {
		t.Error("Expected merging very different code to fail")
	}
	if !strings.Contains(merged2, "CONFLICT") {
		t.Error("Expected conflict marker in merged result")
	}

	// Test keep strategies
	merged3, success3 := kernel.SemanticMerge(baseCode, differentCode, KeepBase)
	if merged3 != baseCode {
		t.Error("KeepBase strategy should return base code")
	}
	if success3 {
		t.Error("KeepBase with different code should return success=false")
	}

	merged4, success4 := kernel.SemanticMerge(baseCode, differentCode, KeepIncoming)
	if merged4 != differentCode {
		t.Error("KeepIncoming strategy should return incoming code")
	}
	if success4 {
		t.Error("KeepIncoming with different code should return success=false")
	}
}

func TestSemanticSimilarityProperties(t *testing.T) {
	kernel := NewSemanticKernel(64, 0.7)

	// Test that syntactically similar code has high semantic similarity
	template := `func %s(a, b int) int {
    %s
    return a %s b
}`

	code1 := strings.ReplaceAll(strings.ReplaceAll(template, "%s", "add"), "return a add b", "return a + b")
	code2 := strings.ReplaceAll(strings.ReplaceAll(template, "%s", "sum"), "return a sum b", "return a + b")
	code3 := strings.ReplaceAll(strings.ReplaceAll(template, "%s", "multiply"), "return a multiply b", "return a * b")

	sim12, _ := kernel.SemanticDiff(code1, code2)
	sim13, _ := kernel.SemanticDiff(code1, code3)

	// Functions with same operation should be more similar than different operations
	// Allow for small differences due to semantic approximation
	if sim12 < sim13-0.05 {
		t.Errorf("Expected same operation to be more similar: %f < %f", sim12, sim13)
	}

	// Test that comments don't drastically change similarity
	codeWithComments := `func add(a, b int) int {
    // This function adds two integers
    return a + b
}`

	codeWithoutComments := `func add(a, b int) int {
    return a + b
}`

	simComments, _ := kernel.SemanticDiff(codeWithComments, codeWithoutComments)
	if simComments < 0.7 {
		t.Errorf("Expected high similarity despite comments, got %f", simComments)
	}
}

// Benchmark tests
func BenchmarkCodeToEmbedding(b *testing.B) {
	kernel := NewSemanticKernel(256, 0.7)
	code := `package main

import (
    "fmt"
    "os"
    "strings"
)

func processFile(filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }

    lines := strings.Split(string(data), "\n")
    for i, line := range lines {
        if strings.TrimSpace(line) != "" {
            fmt.Printf("%d: %s\n", i+1, line)
        }
    }
    return nil
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: program <filename>")
        return
    }

    if err := processFile(os.Args[1]); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.CodeToEmbedding(code)
	}
}

func BenchmarkSemanticDiff(b *testing.B) {
	kernel := NewSemanticKernel(128, 0.7)
	code1 := `func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`

	code2 := `func fib(num int) int {
    if num <= 1 {
        return num
    }
    return fib(num-1) + fib(num-2)
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kernel.SemanticDiff(code1, code2)
	}
}