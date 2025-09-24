package kernel

import (
	"math"
	"math/rand"
)

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinFloat returns the minimum of two float64 values
func MinFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MaxFloat returns the maximum of two float64 values
func MaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// DotProduct computes the dot product of two vectors
func DotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// L2Norm computes the L2 (Euclidean) norm of a vector
func L2Norm(v []float64) float64 {
	sum := 0.0
	for _, val := range v {
		sum += val * val
	}
	return math.Sqrt(sum)
}

// L1Norm computes the L1 (Manhattan) norm of a vector
func L1Norm(v []float64) float64 {
	sum := 0.0
	for _, val := range v {
		sum += math.Abs(val)
	}
	return sum
}

// NormalizeL2 normalizes a vector to unit L2 norm in-place
func NormalizeL2(v []float64) {
	norm := L2Norm(v)
	if norm > 1e-10 { // Avoid division by zero
		for i := range v {
			v[i] /= norm
		}
	}
}

// CosineSimilarity computes cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProd := DotProduct(a, b)
	normA := L2Norm(a)
	normB := L2Norm(b)

	if normA < 1e-10 || normB < 1e-10 {
		return 0.0
	}

	similarity := dotProd / (normA * normB)

	// Clamp to [-1, 1] to handle floating point errors
	return math.Max(-1.0, math.Min(1.0, similarity))
}

// EuclideanDistance computes Euclidean distance between two vectors
func EuclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// ManhattanDistance computes Manhattan distance between two vectors
func ManhattanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	sum := 0.0
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum
}

// RBFKernel computes the RBF (Gaussian) kernel between two vectors
func RBFKernel(a, b []float64, gamma float64) float64 {
	distSq := 0.0
	for i := range a {
		diff := a[i] - b[i]
		distSq += diff * diff
	}
	return math.Exp(-gamma * distSq)
}

// LinearKernel computes the linear kernel between two vectors
func LinearKernel(a, b []float64) float64 {
	return DotProduct(a, b)
}

// PolynomialKernel computes the polynomial kernel between two vectors
func PolynomialKernel(a, b []float64, degree float64, coef0 float64) float64 {
	dot := DotProduct(a, b)
	return math.Pow(dot+coef0, degree)
}

// GenerateRandomVector generates a random vector of specified dimension
func GenerateRandomVector(dim int, rng *rand.Rand) []float64 {
	vec := make([]float64, dim)
	for i := range vec {
		vec[i] = rng.NormFloat64()
	}
	return vec
}

// GenerateRandomMatrix generates a random matrix with specified dimensions
func GenerateRandomMatrix(rows, cols int, rng *rand.Rand) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = GenerateRandomVector(cols, rng)
	}
	return matrix
}

// MatrixVectorProduct computes matrix-vector multiplication
func MatrixVectorProduct(matrix [][]float64, vector []float64) []float64 {
	if len(matrix) == 0 || len(matrix[0]) != len(vector) {
		return nil
	}

	result := make([]float64, len(matrix))
	for i, row := range matrix {
		result[i] = DotProduct(row, vector)
	}
	return result
}

// VectorAdd adds two vectors element-wise
func VectorAdd(a, b []float64) []float64 {
	if len(a) != len(b) {
		return nil
	}

	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

// VectorScale scales a vector by a scalar
func VectorScale(v []float64, scale float64) []float64 {
	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = val * scale
	}
	return result
}

// JaccardSimilarity computes Jaccard similarity between two sets represented as maps
func JaccardSimilarity(setA, setB map[string]bool) float64 {
	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
	}

	// Compute intersection
	intersection := 0
	for item := range setA {
		if setB[item] {
			intersection++
		}
	}

	// Union size = |A| + |B| - |intersection|
	union := len(setA) + len(setB) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// Entropy computes the Shannon entropy of a probability distribution
func Entropy(probs []float64) float64 {
	entropy := 0.0
	for _, p := range probs {
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// KLDivergence computes the Kullback-Leibler divergence between two distributions
func KLDivergence(p, q []float64) float64 {
	if len(p) != len(q) {
		return math.Inf(1)
	}

	kl := 0.0
	for i := range p {
		if p[i] > 0 && q[i] > 0 {
			kl += p[i] * math.Log(p[i]/q[i])
		} else if p[i] > 0 && q[i] == 0 {
			return math.Inf(1)
		}
	}
	return kl
}
