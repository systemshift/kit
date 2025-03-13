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
	WeightsCos  [][]float64 // Random weights for cosine component
	WeightsSin  [][]float64 // Random weights for sine component
	InputDim    int         // Dimensionality of input space
	RandomState *rand.Rand  // Random state for reproducibility
}

// NewIntegrityKernel creates a new integrity kernel with the specified parameters
func NewIntegrityKernel(features, inputDim int, gamma float64, seed int64) *IntegrityKernel {
	rng := rand.New(rand.NewSource(seed))

	// Initialize random weights for RFF
	// These are drawn from Normal(0, 2γ) where γ is the RBF kernel parameter
	weightsCosMat := make([][]float64, features)
	weightsSinMat := make([][]float64, features)

	for i := 0; i < features; i++ {
		weightsCosMat[i] = make([]float64, inputDim)
		weightsSinMat[i] = make([]float64, inputDim)

		// Generate random weights from Normal(0, 2γ)
		for j := 0; j < inputDim; j++ {
			weightsCosMat[i][j] = rng.NormFloat64() * math.Sqrt(2*gamma)
			weightsSinMat[i][j] = rng.NormFloat64() * math.Sqrt(2*gamma)
		}
	}

	return &IntegrityKernel{
		Features:    features,
		Gamma:       gamma,
		WeightsCos:  weightsCosMat,
		WeightsSin:  weightsSinMat,
		InputDim:    inputDim,
		RandomState: rng,
	}
}

// DataToFeatureVector converts raw data to a normalized feature vector
func (k *IntegrityKernel) DataToFeatureVector(data []byte) []float64 {
	// For simplicity, we'll use a sliding window approach to convert bytes to features
	// A more sophisticated approach would use meaningful features from the repository

	// Hash the data to get a fixed-length representation
	hash := sha256.Sum256(data)

	// Convert hash to a feature vector
	vector := make([]float64, k.InputDim)

	// Slide over the hash with a 4-byte window (interpreting as uint32)
	for i := 0; i < min(len(hash)/4, k.InputDim); i++ {
		// Convert 4 bytes to uint32
		val := binary.BigEndian.Uint32(hash[i*4 : i*4+4])
		// Normalize to [-1, 1]
		vector[i] = float64(val)/math.MaxUint32*2 - 1
	}

	// If the input dimension is larger than what we can extract from the hash,
	// we'll pad with zeros
	for i := len(hash) / 4; i < k.InputDim; i++ {
		vector[i] = 0
	}

	return vector
}

// ComputeHash computes the RFF hash for the given data
func (k *IntegrityKernel) ComputeHash(data []byte) []float64 {
	// Convert data to feature vector
	vector := k.DataToFeatureVector(data)

	// Compute RFF hash
	hash := make([]float64, 2*k.Features)

	// Apply the random Fourier features transformation
	for i := 0; i < k.Features; i++ {
		// Compute dot product with random weights
		dotProdCos, dotProdSin := 0.0, 0.0
		for j := 0; j < k.InputDim; j++ {
			dotProdCos += vector[j] * k.WeightsCos[i][j]
			dotProdSin += vector[j] * k.WeightsSin[i][j]
		}

		// Store cosine and sine components for the approximation
		hash[i] = math.Cos(dotProdCos) / math.Sqrt(float64(k.Features))
		hash[i+k.Features] = math.Sin(dotProdSin) / math.Sqrt(float64(k.Features))
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

// Helper function to find the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
