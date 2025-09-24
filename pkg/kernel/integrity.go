package kernel

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
)

// IntegrityKernel implements a Random Fourier Features (RFF) based kernel
// for efficient repository integrity verification without reconstructing
// full commit histories.
type IntegrityKernel struct {
	Features    int         // Number of random features
	Gamma       float64     // RBF kernel parameter
	Weights     [][]float64 // Random weights for RFF (shared for cos/sin)
	Offsets     []float64   // Random phase offsets for RFF
	InputDim    int         // Dimensionality of input space
	RandomState *rand.Rand  // Random state for reproducibility
}

// NewIntegrityKernel creates a new integrity kernel with the specified parameters
func NewIntegrityKernel(features, inputDim int, gamma float64, seed int64) *IntegrityKernel {
	rng := rand.New(rand.NewSource(seed))

	// Initialize random weights for RFF
	// These are drawn from Normal(0, 2γ) where γ is the RBF kernel parameter
	weightsMat := make([][]float64, features)
	offsets := make([]float64, features)

	for i := 0; i < features; i++ {
		weightsMat[i] = make([]float64, inputDim)

		// Generate random weights from Normal(0, 2γ)
		for j := 0; j < inputDim; j++ {
			weightsMat[i][j] = rng.NormFloat64() * math.Sqrt(2*gamma)
		}

		// Random phase offset uniform in [0, 2π]
		offsets[i] = rng.Float64() * 2 * math.Pi
	}

	return &IntegrityKernel{
		Features:    features,
		Gamma:       gamma,
		Weights:     weightsMat,
		Offsets:     offsets,
		InputDim:    inputDim,
		RandomState: rng,
	}
}

// DataToFeatureVector converts raw data to a normalized feature vector
func (k *IntegrityKernel) DataToFeatureVector(data []byte) []float64 {
	// Create multiple hash views of the data for better feature representation
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(append(data, 0x01)) // Augmented hash
	h3 := sha256.Sum256(append([]byte{0x02}, data...)) // Prefixed hash

	vector := make([]float64, k.InputDim)
	hashes := [3][32]byte{h1, h2, h3}

	// Extract features from multiple hash views
	idx := 0
	for hashIdx := 0; hashIdx < len(hashes) && idx < k.InputDim; hashIdx++ {
		hash := hashes[hashIdx]

		// Use 8-byte windows for more granular features
		for i := 0; i < len(hash)/8 && idx < k.InputDim; i++ {
			// Convert 8 bytes to uint64
			val := binary.BigEndian.Uint64(hash[i*8 : i*8+8])
			// Normalize to [-1, 1] with better distribution
			vector[idx] = (float64(val)/math.MaxUint64 - 0.5) * 2
			idx++
		}
	}

	// Add data length as a feature (normalized)
	if idx < k.InputDim {
		vector[idx] = math.Tanh(float64(len(data)) / 1000000.0) // Normalize around 1MB
		idx++
	}

	// Pad remaining dimensions with zeros
	for i := idx; i < k.InputDim; i++ {
		vector[i] = 0
	}

	return vector
}

// ComputeHash computes the RFF hash for the given data
func (k *IntegrityKernel) ComputeHash(data []byte) []float64 {
	// Convert data to feature vector
	vector := k.DataToFeatureVector(data)

	// Compute RFF hash using proper RFF formula
	hash := make([]float64, k.Features)

	// Apply the random Fourier features transformation
	// RFF approximation: φ(x) = sqrt(2/D) * cos(w^T x + b)
	// where w ~ N(0, 2γI) and b ~ Uniform(0, 2π)
	for i := 0; i < k.Features; i++ {
		// Compute dot product with random weights
		dotProd := 0.0
		for j := 0; j < k.InputDim; j++ {
			dotProd += vector[j] * k.Weights[i][j]
		}

		// Apply RFF transformation with proper normalization
		hash[i] = math.Sqrt(2.0/float64(k.Features)) * math.Cos(dotProd+k.Offsets[i])
	}

	return hash
}

// Similarity computes the approximate RBF kernel similarity between two hashes
func (k *IntegrityKernel) Similarity(hash1, hash2 []float64) float64 {
	// For RFF, the similarity is just the dot product of the hashes
	dotProd := 0.0
	for i := 0; i < len(hash1); i++ {
		dotProd += hash1[i] * hash2[i]
	}

	return dotProd
}

// VerifyIntegrity checks if two repositories have similar states
// Returns a similarity score and a boolean indicating if they're considered identical
func (k *IntegrityKernel) VerifyIntegrity(data1, data2 []byte, threshold float64) (float64, bool) {
	hash1 := k.ComputeHash(data1)
	hash2 := k.ComputeHash(data2)

	similarity := k.Similarity(hash1, hash2)

	return similarity, similarity >= threshold
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
