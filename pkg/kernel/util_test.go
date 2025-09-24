package kernel

import (
	"math"
	"math/rand"
	"testing"
)

func TestMinMax(t *testing.T) {
	// Test integers
	if Min(5, 3) != 3 {
		t.Errorf("Min(5, 3) expected 3, got %d", Min(5, 3))
	}
	if Max(5, 3) != 5 {
		t.Errorf("Max(5, 3) expected 5, got %d", Max(5, 3))
	}

	// Test floats
	if MinFloat(5.5, 3.3) != 3.3 {
		t.Errorf("MinFloat(5.5, 3.3) expected 3.3, got %f", MinFloat(5.5, 3.3))
	}
	if MaxFloat(5.5, 3.3) != 5.5 {
		t.Errorf("MaxFloat(5.5, 3.3) expected 5.5, got %f", MaxFloat(5.5, 3.3))
	}

	// Test edge cases
	if Min(-1, -2) != -2 {
		t.Errorf("Min(-1, -2) expected -2, got %d", Min(-1, -2))
	}
	if Max(-1, -2) != -1 {
		t.Errorf("Max(-1, -2) expected -1, got %d", Max(-1, -2))
	}
}

func TestDotProduct(t *testing.T) {
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{4, 5, 6}
	// expected := 1*4 + 2*5 + 3*6 // 32

	result := DotProduct(vec1, vec2)
	if math.Abs(result-32) > 1e-10 {
		t.Errorf("DotProduct expected 32, got %f", result)
	}

	// Test zero vectors
	zero1 := []float64{0, 0, 0}
	zero2 := []float64{0, 0, 0}
	if DotProduct(zero1, zero2) != 0 {
		t.Errorf("Dot product of zero vectors should be 0, got %f", DotProduct(zero1, zero2))
	}

	// Test different lengths
	vec3 := []float64{1, 2}
	result2 := DotProduct(vec1, vec3)
	if result2 != 0 {
		t.Errorf("Dot product of different length vectors should be 0, got %f", result2)
	}
}

func TestL2Norm(t *testing.T) {
	vec := []float64{3, 4}
	// expected := 5.0 // sqrt(3^2 + 4^2)

	result := L2Norm(vec)
	if math.Abs(result-5.0) > 1e-10 {
		t.Errorf("L2Norm expected 5.0, got %f", result)
	}

	// Test unit vector
	unit := []float64{1, 0, 0}
	if math.Abs(L2Norm(unit)-1.0) > 1e-10 {
		t.Errorf("L2Norm of unit vector should be 1.0, got %f", L2Norm(unit))
	}

	// Test zero vector
	zero := []float64{0, 0, 0}
	if L2Norm(zero) != 0 {
		t.Errorf("L2Norm of zero vector should be 0, got %f", L2Norm(zero))
	}
}

func TestL1Norm(t *testing.T) {
	vec := []float64{3, -4, 5}
	// expected := 3 + 4 + 5 // 12

	result := L1Norm(vec)
	if math.Abs(result-12) > 1e-10 {
		t.Errorf("L1Norm expected 12, got %f", result)
	}

	// Test zero vector
	zero := []float64{0, 0, 0}
	if L1Norm(zero) != 0 {
		t.Errorf("L1Norm of zero vector should be 0, got %f", L1Norm(zero))
	}
}

func TestNormalizeL2(t *testing.T) {
	vec := []float64{3, 4, 0}
	originalNorm := L2Norm(vec)

	NormalizeL2(vec)

	newNorm := L2Norm(vec)
	if math.Abs(newNorm-1.0) > 1e-10 {
		t.Errorf("Normalized vector should have unit norm, got %f", newNorm)
	}

	// Check that direction is preserved (proportional to original)
	expectedRatio := 1.0 / originalNorm
	if math.Abs(vec[0]-(3*expectedRatio)) > 1e-10 {
		t.Errorf("Normalization changed direction")
	}

	// Test zero vector (should remain zero)
	zero := []float64{0, 0, 0}
	NormalizeL2(zero)
	for i, val := range zero {
		if val != 0 {
			t.Errorf("Normalized zero vector should remain zero, got %f at index %d", val, i)
		}
	}
}

func TestCosineSimilarityUtil(t *testing.T) {
	// Test identical vectors
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{1, 2, 3}
	sim := CosineSimilarity(vec1, vec2)
	if math.Abs(sim-1.0) > 1e-10 {
		t.Errorf("Cosine similarity of identical vectors should be 1.0, got %f", sim)
	}

	// Test orthogonal vectors
	vec3 := []float64{1, 0, 0}
	vec4 := []float64{0, 1, 0}
	sim2 := CosineSimilarity(vec3, vec4)
	if math.Abs(sim2) > 1e-10 {
		t.Errorf("Cosine similarity of orthogonal vectors should be 0, got %f", sim2)
	}

	// Test opposite vectors
	vec5 := []float64{1, 1, 1}
	vec6 := []float64{-1, -1, -1}
	sim3 := CosineSimilarity(vec5, vec6)
	if math.Abs(sim3-(-1.0)) > 1e-10 {
		t.Errorf("Cosine similarity of opposite vectors should be -1.0, got %f", sim3)
	}

	// Test different lengths
	vec7 := []float64{1, 2}
	sim4 := CosineSimilarity(vec1, vec7)
	if sim4 != 0 {
		t.Errorf("Cosine similarity of different length vectors should be 0, got %f", sim4)
	}

	// Test zero vectors
	zero := []float64{0, 0, 0}
	sim5 := CosineSimilarity(vec1, zero)
	if sim5 != 0 {
		t.Errorf("Cosine similarity with zero vector should be 0, got %f", sim5)
	}
}

func TestEuclideanDistance(t *testing.T) {
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{4, 6, 8}
	// Distance = sqrt((4-1)^2 + (6-2)^2 + (8-3)^2) = sqrt(9 + 16 + 25) = sqrt(50)
	expected := math.Sqrt(50)

	result := EuclideanDistance(vec1, vec2)
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("EuclideanDistance expected %f, got %f", expected, result)
	}

	// Test self-distance
	dist := EuclideanDistance(vec1, vec1)
	if dist != 0 {
		t.Errorf("Self-distance should be 0, got %f", dist)
	}

	// Test different lengths
	vec3 := []float64{1, 2}
	dist2 := EuclideanDistance(vec1, vec3)
	if !math.IsInf(dist2, 1) {
		t.Errorf("Distance between different length vectors should be +Inf, got %f", dist2)
	}
}

func TestManhattanDistance(t *testing.T) {
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{4, 6, 8}
	// Distance = |4-1| + |6-2| + |8-3| = 3 + 4 + 5 = 12
	expected := 12.0

	result := ManhattanDistance(vec1, vec2)
	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("ManhattanDistance expected %f, got %f", expected, result)
	}

	// Test self-distance
	dist := ManhattanDistance(vec1, vec1)
	if dist != 0 {
		t.Errorf("Self-distance should be 0, got %f", dist)
	}
}

func TestRBFKernel(t *testing.T) {
	vec1 := []float64{0, 0}
	vec2 := []float64{0, 0}
	gamma := 1.0

	// Test identical vectors
	result := RBFKernel(vec1, vec2, gamma)
	if math.Abs(result-1.0) > 1e-10 {
		t.Errorf("RBF kernel of identical vectors should be 1.0, got %f", result)
	}

	// Test different vectors
	vec3 := []float64{1, 1}
	result2 := RBFKernel(vec1, vec3, gamma)
	expected := math.Exp(-gamma * 2) // Distance squared is 2
	if math.Abs(result2-expected) > 1e-10 {
		t.Errorf("RBF kernel expected %f, got %f", expected, result2)
	}

	// Test with different gamma
	result3 := RBFKernel(vec1, vec3, 0.5)
	expected3 := math.Exp(-0.5 * 2)
	if math.Abs(result3-expected3) > 1e-10 {
		t.Errorf("RBF kernel with gamma=0.5 expected %f, got %f", expected3, result3)
	}
}

func TestLinearKernel(t *testing.T) {
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{4, 5, 6}

	result := LinearKernel(vec1, vec2)
	expected := DotProduct(vec1, vec2)

	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("Linear kernel should equal dot product, got %f expected %f", result, expected)
	}
}

func TestPolynomialKernel(t *testing.T) {
	vec1 := []float64{1, 2}
	vec2 := []float64{3, 4}
	degree := 2.0
	coef0 := 1.0

	result := PolynomialKernel(vec1, vec2, degree, coef0)
	dotProd := DotProduct(vec1, vec2) // 1*3 + 2*4 = 11
	expected := math.Pow(dotProd+coef0, degree) // (11+1)^2 = 144

	if math.Abs(result-expected) > 1e-10 {
		t.Errorf("Polynomial kernel expected %f, got %f", expected, result)
	}
}

func TestGenerateRandomVector(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dim := 5

	vec := GenerateRandomVector(dim, rng)

	if len(vec) != dim {
		t.Errorf("Expected vector dimension %d, got %d", dim, len(vec))
	}

	// Test determinism with same seed
	rng2 := rand.New(rand.NewSource(42))
	vec2 := GenerateRandomVector(dim, rng2)

	for i := range vec {
		if vec[i] != vec2[i] {
			t.Errorf("Random vector generation not deterministic at index %d", i)
		}
	}
}

func TestGenerateRandomMatrix(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	rows, cols := 3, 4

	matrix := GenerateRandomMatrix(rows, cols, rng)

	if len(matrix) != rows {
		t.Errorf("Expected %d rows, got %d", rows, len(matrix))
	}

	for i, row := range matrix {
		if len(row) != cols {
			t.Errorf("Row %d expected %d columns, got %d", i, cols, len(row))
		}
	}
}

func TestMatrixVectorProduct(t *testing.T) {
	matrix := [][]float64{
		{1, 2, 3},
		{4, 5, 6},
	}
	vector := []float64{1, 1, 1}

	result := MatrixVectorProduct(matrix, vector)

	expected := []float64{6, 15} // [1+2+3, 4+5+6]

	if len(result) != len(expected) {
		t.Errorf("Expected result length %d, got %d", len(expected), len(result))
	}

	for i, val := range expected {
		if math.Abs(result[i]-val) > 1e-10 {
			t.Errorf("MatrixVectorProduct[%d] expected %f, got %f", i, val, result[i])
		}
	}

	// Test mismatched dimensions
	wrongVector := []float64{1, 1}
	result2 := MatrixVectorProduct(matrix, wrongVector)
	if result2 != nil {
		t.Error("Expected nil result for mismatched dimensions")
	}
}

func TestVectorAdd(t *testing.T) {
	vec1 := []float64{1, 2, 3}
	vec2 := []float64{4, 5, 6}
	expected := []float64{5, 7, 9}

	result := VectorAdd(vec1, vec2)

	if len(result) != len(expected) {
		t.Errorf("Expected result length %d, got %d", len(expected), len(result))
	}

	for i, val := range expected {
		if math.Abs(result[i]-val) > 1e-10 {
			t.Errorf("VectorAdd[%d] expected %f, got %f", i, val, result[i])
		}
	}

	// Test different lengths
	vec3 := []float64{1, 2}
	result2 := VectorAdd(vec1, vec3)
	if result2 != nil {
		t.Error("Expected nil result for different length vectors")
	}
}

func TestVectorScale(t *testing.T) {
	vec := []float64{1, 2, 3}
	scale := 2.5
	expected := []float64{2.5, 5.0, 7.5}

	result := VectorScale(vec, scale)

	if len(result) != len(expected) {
		t.Errorf("Expected result length %d, got %d", len(expected), len(result))
	}

	for i, val := range expected {
		if math.Abs(result[i]-val) > 1e-10 {
			t.Errorf("VectorScale[%d] expected %f, got %f", i, val, result[i])
		}
	}

	// Original vector should be unchanged
	if vec[0] != 1 || vec[1] != 2 || vec[2] != 3 {
		t.Error("VectorScale modified original vector")
	}
}

func TestJaccardSimilarity(t *testing.T) {
	// Test identical sets
	set1 := map[string]bool{"a": true, "b": true, "c": true}
	set2 := map[string]bool{"a": true, "b": true, "c": true}
	sim := JaccardSimilarity(set1, set2)
	if sim != 1.0 {
		t.Errorf("Jaccard similarity of identical sets should be 1.0, got %f", sim)
	}

	// Test disjoint sets
	set3 := map[string]bool{"d": true, "e": true, "f": true}
	sim2 := JaccardSimilarity(set1, set3)
	if sim2 != 0.0 {
		t.Errorf("Jaccard similarity of disjoint sets should be 0.0, got %f", sim2)
	}

	// Test partial overlap
	set4 := map[string]bool{"a": true, "b": true, "d": true}
	sim3 := JaccardSimilarity(set1, set4)
	expected := 2.0 / 4.0 // 2 intersection, 4 union
	if math.Abs(sim3-expected) > 1e-10 {
		t.Errorf("Jaccard similarity expected %f, got %f", expected, sim3)
	}

	// Test empty sets
	empty1 := map[string]bool{}
	empty2 := map[string]bool{}
	sim4 := JaccardSimilarity(empty1, empty2)
	if sim4 != 1.0 {
		t.Errorf("Jaccard similarity of empty sets should be 1.0, got %f", sim4)
	}
}

func TestEntropy(t *testing.T) {
	// Test uniform distribution
	uniform := []float64{0.5, 0.5}
	entropy := Entropy(uniform)
	expected := 1.0 // log2(2)
	if math.Abs(entropy-expected) > 1e-10 {
		t.Errorf("Entropy of uniform binary distribution expected %f, got %f", expected, entropy)
	}

	// Test certain event
	certain := []float64{1.0, 0.0}
	entropy2 := Entropy(certain)
	if entropy2 != 0.0 {
		t.Errorf("Entropy of certain event should be 0.0, got %f", entropy2)
	}

	// Test quaternary uniform
	quaternary := []float64{0.25, 0.25, 0.25, 0.25}
	entropy3 := Entropy(quaternary)
	expected3 := 2.0 // log2(4)
	if math.Abs(entropy3-expected3) > 1e-10 {
		t.Errorf("Entropy of uniform quaternary distribution expected %f, got %f", expected3, entropy3)
	}
}

func TestKLDivergence(t *testing.T) {
	// Test identical distributions
	p := []float64{0.5, 0.3, 0.2}
	q := []float64{0.5, 0.3, 0.2}
	kl := KLDivergence(p, q)
	if kl != 0.0 {
		t.Errorf("KL divergence of identical distributions should be 0.0, got %f", kl)
	}

	// Test different distributions
	q2 := []float64{0.4, 0.4, 0.2}
	kl2 := KLDivergence(p, q2)
	if kl2 <= 0 {
		t.Errorf("KL divergence should be positive for different distributions, got %f", kl2)
	}

	// Test q has zero where p is positive (should be infinity)
	q3 := []float64{0.5, 0.5, 0.0}
	kl3 := KLDivergence(p, q3)
	if !math.IsInf(kl3, 1) {
		t.Errorf("KL divergence should be +Inf when q=0 and p>0, got %f", kl3)
	}

	// Test different lengths
	q4 := []float64{0.5, 0.5}
	kl4 := KLDivergence(p, q4)
	if !math.IsInf(kl4, 1) {
		t.Errorf("KL divergence should be +Inf for different length distributions, got %f", kl4)
	}
}

// Benchmark tests
func BenchmarkDotProduct(b *testing.B) {
	vec1 := make([]float64, 1000)
	vec2 := make([]float64, 1000)
	rng := rand.New(rand.NewSource(42))

	for i := range vec1 {
		vec1[i] = rng.NormFloat64()
		vec2[i] = rng.NormFloat64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DotProduct(vec1, vec2)
	}
}

func BenchmarkL2Norm(b *testing.B) {
	vec := make([]float64, 1000)
	rng := rand.New(rand.NewSource(42))

	for i := range vec {
		vec[i] = rng.NormFloat64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = L2Norm(vec)
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	vec1 := make([]float64, 1000)
	vec2 := make([]float64, 1000)
	rng := rand.New(rand.NewSource(42))

	for i := range vec1 {
		vec1[i] = rng.NormFloat64()
		vec2[i] = rng.NormFloat64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CosineSimilarity(vec1, vec2)
	}
}