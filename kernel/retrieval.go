package kernel

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// RetrievalKernel implements MinHash and Locality-Sensitive Hashing (LSH)
// for efficient semantic retrieval within monorepos
type RetrievalKernel struct {
	NumPermutations int        // Number of permutations for MinHash
	NumBands        int        // Number of bands for LSH
	NumRows         int        // Number of rows per band
	Seed            int64      // Random seed for permutations
	Permutations    [][]int    // Permutation functions
	HashBands       [][]int    // Band hashing functions
	RandomState     *rand.Rand // Random state for reproducibility
}

// NewRetrievalKernel creates a new retrieval kernel with specified parameters
func NewRetrievalKernel(numPermutations, universeSize int, numBands int, seed int64) *RetrievalKernel {
	if numBands > numPermutations {
		numBands = numPermutations
	}

	numRows := numPermutations / numBands

	// Create random number generator
	rng := rand.New(rand.NewSource(seed))

	// Generate permutation functions
	permutations := make([][]int, numPermutations)
	for i := range permutations {
		// Each permutation is a mapping of indices
		permutations[i] = rand.Perm(universeSize)
	}

	// Generate random coefficients for band hashing
	hashBands := make([][]int, numBands)
	for i := range hashBands {
		hashBands[i] = make([]int, 2)
		// Random coefficients for linear hash function: (ax + b) mod p
		hashBands[i][0] = rng.Intn(math.MaxInt32)
		hashBands[i][1] = rng.Intn(math.MaxInt32)
	}

	return &RetrievalKernel{
		NumPermutations: numPermutations,
		NumBands:        numBands,
		NumRows:         numRows,
		Seed:            seed,
		Permutations:    permutations,
		HashBands:       hashBands,
		RandomState:     rng,
	}
}

// MinHash computes the MinHash signature for a given document
// The document is represented as a set of shingles (n-grams)
func (k *RetrievalKernel) MinHash(document string) []int {
	// Convert document to shingles (n-grams of words for text, tokens for code)
	shingles := k.documentToShingles(document)

	// Initialize MinHash signature with maximum values
	signature := make([]int, k.NumPermutations)
	for i := range signature {
		signature[i] = math.MaxInt32
	}

	// For each shingle
	for _, shingle := range shingles {
		// Hash the shingle to get its index
		shingleIndex := k.hashShingle(shingle)

		// Update signature for each permutation
		for i := 0; i < k.NumPermutations; i++ {
			// Apply permutation to shingle index
			permutedIndex := k.Permutations[i][shingleIndex%len(k.Permutations[i])]

			// Update signature if permuted value is smaller
			if permutedIndex < signature[i] {
				signature[i] = permutedIndex
			}
		}
	}

	return signature
}

// LSHSignature computes the LSH signature for a MinHash signature
// This enables efficient near-neighbor queries
func (k *RetrievalKernel) LSHSignature(minHashSignature []int) []string {
	// Initialize LSH signature
	lshSignature := make([]string, k.NumBands)

	// Process each band
	for band := 0; band < k.NumBands; band++ {
		// Start index for this band
		startIdx := band * k.NumRows

		// Extract band values
		bandValues := minHashSignature[startIdx:min(startIdx+k.NumRows, len(minHashSignature))]

		// Compute band hash
		bandHash := k.hashBand(bandValues, band)

		// Convert to string and append to LSH signature
		lshSignature[band] = strconv.Itoa(band) + ":" + strconv.Itoa(bandHash)
	}

	return lshSignature
}

// ComputeJaccardSimilarity calculates Jaccard similarity between two MinHash signatures
// This provides a measure of document similarity
func (k *RetrievalKernel) ComputeJaccardSimilarity(sig1, sig2 []int) float64 {
	if len(sig1) != len(sig2) {
		return 0.0
	}

	// Count matching values
	matches := 0
	for i := range sig1 {
		if sig1[i] == sig2[i] {
			matches++
		}
	}

	// Jaccard similarity = proportion of matching values
	return float64(matches) / float64(len(sig1))
}

// EstimateSimilarity estimates similarity between two documents using MinHash
func (k *RetrievalKernel) EstimateSimilarity(doc1, doc2 string) float64 {
	// Compute MinHash signatures
	sig1 := k.MinHash(doc1)
	sig2 := k.MinHash(doc2)

	// Calculate Jaccard similarity
	return k.ComputeJaccardSimilarity(sig1, sig2)
}

// AreLikelySimilar determines if two documents are likely similar using LSH
// More efficient than computing full similarity, used for candidate generation
func (k *RetrievalKernel) AreLikelySimilar(doc1, doc2 string) bool {
	// Compute LSH signatures
	lsh1 := k.LSHSignature(k.MinHash(doc1))
	lsh2 := k.LSHSignature(k.MinHash(doc2))

	// Check for any matching bands
	// Documents are considered similar if they share at least one band
	for _, band1 := range lsh1 {
		for _, band2 := range lsh2 {
			if band1 == band2 {
				return true
			}
		}
	}

	return false
}

// Helper methods

// documentToShingles converts a document to a set of shingles (n-grams)
func (k *RetrievalKernel) documentToShingles(document string) []string {
	// In a real implementation, this would use proper tokenization
	// For simplicity, we'll use word trigrams

	// Split document into words
	words := strings.Fields(document)

	// Create shingles (word trigrams)
	shingles := make([]string, 0)

	if len(words) < 3 {
		// For short documents, use individual words
		return words
	}

	// Create trigrams
	for i := 0; i <= len(words)-3; i++ {
		shingle := words[i] + " " + words[i+1] + " " + words[i+2]
		shingles = append(shingles, shingle)
	}

	return shingles
}

// hashShingle hashes a shingle to an integer index
func (k *RetrievalKernel) hashShingle(shingle string) int {
	// Hash shingle to get a fixed-size representation
	hash := sha256.Sum256([]byte(shingle))

	// Convert first 4 bytes to uint32
	return int(binary.BigEndian.Uint32(hash[:4]))
}

// hashBand hashes a band of MinHash values to a single integer
func (k *RetrievalKernel) hashBand(bandValues []int, bandIndex int) int {
	// Simple hash function: add all values with coefficients
	hash := 0
	a := k.HashBands[bandIndex][0]
	b := k.HashBands[bandIndex][1]

	for i, value := range bandValues {
		// Linear hash function: (a*x + b) mod p
		// We use different weights for each position in the band
		hash = (hash + ((a*(value+i) + b) % 2147483647)) % 2147483647
	}

	return hash
}
