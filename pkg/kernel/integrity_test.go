package kernel

import (
	"math"
	"testing"
)

func TestNewIntegrityKernel(t *testing.T) {
	kernel := NewIntegrityKernel(100, 50, 0.5, 42)

	if kernel.Features != 100 {
		t.Errorf("Expected 100 features, got %d", kernel.Features)
	}
	if kernel.InputDim != 50 {
		t.Errorf("Expected input dimension 50, got %d", kernel.InputDim)
	}
	if kernel.Gamma != 0.5 {
		t.Errorf("Expected gamma 0.5, got %f", kernel.Gamma)
	}
	if len(kernel.Weights) != 100 {
		t.Errorf("Expected 100 weight vectors, got %d", len(kernel.Weights))
	}
	if len(kernel.Offsets) != 100 {
		t.Errorf("Expected 100 offset values, got %d", len(kernel.Offsets))
	}

	// Check that weights have correct dimension
	for i, w := range kernel.Weights {
		if len(w) != 50 {
			t.Errorf("Weight vector %d has dimension %d, expected 50", i, len(w))
		}
	}

	// Check that offsets are in valid range [0, 2π]
	for i, offset := range kernel.Offsets {
		if offset < 0 || offset > 2*math.Pi {
			t.Errorf("Offset %d is %f, expected range [0, 2π]", i, offset)
		}
	}
}

func TestDataToFeatureVector(t *testing.T) {
	kernel := NewIntegrityKernel(100, 20, 0.5, 42)

	testData := []byte("Hello, world!")
	vector := kernel.DataToFeatureVector(testData)

	if len(vector) != 20 {
		t.Errorf("Expected feature vector of length 20, got %d", len(vector))
	}

	// Check that values are normalized to [-1, 1] range (approximately)
	for i, val := range vector {
		if val < -1.1 || val > 1.1 {
			t.Errorf("Feature %d value %f is outside expected range [-1, 1]", i, val)
		}
	}

	// Test determinism - same input should give same output
	vector2 := kernel.DataToFeatureVector(testData)
	for i, val := range vector {
		if val != vector2[i] {
			t.Errorf("Feature vectors are not deterministic at index %d: %f != %f", i, val, vector2[i])
		}
	}

	// Test different inputs give different outputs
	vector3 := kernel.DataToFeatureVector([]byte("Different data"))
	same := true
	for i, val := range vector {
		if val != vector3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different inputs produced identical feature vectors")
	}
}

func TestComputeHash(t *testing.T) {
	kernel := NewIntegrityKernel(50, 20, 0.5, 42)

	testData := []byte("Test data for hashing")
	hash := kernel.ComputeHash(testData)

	if len(hash) != 50 {
		t.Errorf("Expected hash length 50, got %d", len(hash))
	}

	// Test determinism
	hash2 := kernel.ComputeHash(testData)
	for i, val := range hash {
		if math.Abs(val-hash2[i]) > 1e-10 {
			t.Errorf("Hash is not deterministic at index %d: %f != %f", i, val, hash2[i])
		}
	}

	// Test that RFF normalization is applied correctly
	// The hash should have bounded values due to cosine function
	for i, val := range hash {
		if math.Abs(val) > 1.0 {
			t.Errorf("Hash value %d is %f, expected bounded by cosine function", i, val)
		}
	}
}

func TestSimilarity(t *testing.T) {
	kernel := NewIntegrityKernel(100, 50, 0.5, 42)

	testData1 := []byte("Hello, world!")
	testData2 := []byte("Hello, world!")
	testData3 := []byte("Completely different content")

	hash1 := kernel.ComputeHash(testData1)
	hash2 := kernel.ComputeHash(testData2)
	hash3 := kernel.ComputeHash(testData3)

	// Test identical data (should be very high similarity, but RFF is approximate)
	sim12 := kernel.Similarity(hash1, hash2)
	if sim12 < 0.95 {
		t.Errorf("Expected high similarity for identical data, got %f", sim12)
	}

	// Test self-similarity (RFF approximation, allow some tolerance)
	sim11 := kernel.Similarity(hash1, hash1)
	if sim11 < 0.95 {
		t.Errorf("Expected high self-similarity, got %f", sim11)
	}

	// Test different data
	sim13 := kernel.Similarity(hash1, hash3)
	if sim13 > 0.9 {
		t.Errorf("Expected low similarity for different data, got %f", sim13)
	}

	// Test symmetry
	sim31 := kernel.Similarity(hash3, hash1)
	if math.Abs(sim13-sim31) > 1e-10 {
		t.Errorf("Similarity is not symmetric: %f != %f", sim13, sim31)
	}
}

func TestVerifyIntegrity(t *testing.T) {
	kernel := NewIntegrityKernel(100, 50, 0.5, 42)

	testData := []byte("Test data for verification")

	// Test identical data (RFF is approximate, so allow some tolerance)
	similarity, isValid := kernel.VerifyIntegrity(testData, testData, 0.95)
	if !isValid {
		t.Error("Expected identical data to pass verification")
	}
	if similarity < 0.95 {
		t.Errorf("Expected high similarity for identical data, got %f", similarity)
	}

	// Test different data
	differentData := []byte("Different test data")
	similarity2, isValid2 := kernel.VerifyIntegrity(testData, differentData, 0.99)
	if isValid2 {
		t.Error("Expected different data to fail strict verification")
	}
	if similarity2 > 0.9 {
		t.Errorf("Expected low similarity for different data, got %f", similarity2)
	}

	// Test with threshold based on actual similarity
	actualSim, isValid3 := kernel.VerifyIntegrity(testData, differentData, similarity2-0.01)
	if !isValid3 {
		t.Errorf("Expected different data to pass with threshold %f (actual similarity: %f)", similarity2-0.01, actualSim)
	}
}

func TestRFFApproximation(t *testing.T) {
	// Test that RFF provides a reasonable approximation to RBF kernel
	kernel := NewIntegrityKernel(1000, 10, 1.0, 42) // More features for better approximation

	// Generate test vectors
	vec1 := []float64{1, 2, 3, 4, 5, 0, 0, 0, 0, 0}
	vec2 := []float64{1.1, 2.1, 3.1, 4.1, 5.1, 0, 0, 0, 0, 0}
	vec3 := []float64{-1, -2, -3, -4, -5, 0, 0, 0, 0, 0}

	// Convert to byte data for kernel processing
	data1 := make([]byte, len(vec1)*8)
	data2 := make([]byte, len(vec2)*8)
	data3 := make([]byte, len(vec3)*8)

	for i, v := range vec1 {
		bits := math.Float64bits(v)
		for j := 0; j < 8; j++ {
			data1[i*8+j] = byte(bits >> (8 * (7 - j)))
		}
	}
	for i, v := range vec2 {
		bits := math.Float64bits(v)
		for j := 0; j < 8; j++ {
			data2[i*8+j] = byte(bits >> (8 * (7 - j)))
		}
	}
	for i, v := range vec3 {
		bits := math.Float64bits(v)
		for j := 0; j < 8; j++ {
			data3[i*8+j] = byte(bits >> (8 * (7 - j)))
		}
	}

	// Compute RFF similarities
	hash1 := kernel.ComputeHash(data1)
	hash2 := kernel.ComputeHash(data2)
	hash3 := kernel.ComputeHash(data3)

	sim12_rff := kernel.Similarity(hash1, hash2)
	sim13_rff := kernel.Similarity(hash1, hash3)

	// The RFF approximation should satisfy basic properties:
	// 1. Similar vectors should have higher similarity than dissimilar ones
	if sim12_rff <= sim13_rff {
		t.Errorf("Expected similar vectors to have higher RFF similarity: %f vs %f", sim12_rff, sim13_rff)
	}

	// 2. Self-similarity should be high (RFF is approximate)
	sim11_rff := kernel.Similarity(hash1, hash1)
	if sim11_rff < 0.95 {
		t.Errorf("Expected high self-similarity, got %f", sim11_rff)
	}
}

// Benchmark tests
func BenchmarkComputeHash(b *testing.B) {
	kernel := NewIntegrityKernel(256, 128, 0.5, 42)
	testData := make([]byte, 10000) // 10KB of data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.ComputeHash(testData)
	}
}

func BenchmarkSimilarity(b *testing.B) {
	kernel := NewIntegrityKernel(256, 128, 0.5, 42)
	testData := []byte("Test data for benchmarking")
	hash1 := kernel.ComputeHash(testData)
	hash2 := kernel.ComputeHash(append(testData, byte(1)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kernel.Similarity(hash1, hash2)
	}
}