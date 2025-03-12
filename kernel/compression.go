package kernel

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math"
	"math/rand"
)

// CompressionKernel implements semantic compression using kernel PCA
// for highly efficient storage of repository contents
type CompressionKernel struct {
	EmbeddingDim int         // Dimensionality of semantic embeddings
	Components   [][]float64 // Principal components for PCA
	Mean         []float64   // Mean vector for centering data
	Gamma        float64     // RBF kernel parameter
	Seed         int64       // Random seed for reproducibility
	RandomState  *rand.Rand  // Random state for reproducibility
	UseZlib      bool        // Whether to apply additional zlib compression
	ZlibLevel    int         // Compression level for zlib (1-9)
	QuantizeBits int         // Number of bits for quantization (8, 16, or 32)
}

// NewCompressionKernel creates a new compression kernel with specified parameters
func NewCompressionKernel(embeddingDim int, components int, gamma float64, seed int64, useZlib bool, zlibLevel, quantizeBits int) *CompressionKernel {
	// Validate parameters
	if zlibLevel < 1 || zlibLevel > 9 {
		zlibLevel = 6 // Default level
	}

	if quantizeBits != 8 && quantizeBits != 16 && quantizeBits != 32 {
		quantizeBits = 16 // Default to 16-bit quantization
	}

	// Initialize random number generator
	rng := rand.New(rand.NewSource(seed))

	// Generate random principal components
	// In a real implementation, these would be learned from data
	componentMatrix := make([][]float64, components)
	for i := range componentMatrix {
		componentMatrix[i] = make([]float64, embeddingDim)

		// Generate random unit vectors as principal components
		sumSquares := 0.0
		for j := range componentMatrix[i] {
			val := rng.NormFloat64()
			componentMatrix[i][j] = val
			sumSquares += val * val
		}

		// Normalize to unit length
		norm := math.Sqrt(sumSquares)
		for j := range componentMatrix[i] {
			componentMatrix[i][j] /= norm
		}
	}

	// Generate random mean vector
	meanVector := make([]float64, embeddingDim)
	for i := range meanVector {
		meanVector[i] = rng.NormFloat64() * 0.1 // Small random values
	}

	return &CompressionKernel{
		EmbeddingDim: embeddingDim,
		Components:   componentMatrix,
		Mean:         meanVector,
		Gamma:        gamma,
		Seed:         seed,
		RandomState:  rng,
		UseZlib:      useZlib,
		ZlibLevel:    zlibLevel,
		QuantizeBits: quantizeBits,
	}
}

// DataToFeatureVector converts raw data to a feature vector
func (k *CompressionKernel) DataToFeatureVector(data []byte) []float64 {
	// For simplicity, we'll use a sliding window approach
	// A more sophisticated approach would use meaningful features

	// Create a feature vector of the appropriate dimension
	vector := make([]float64, k.EmbeddingDim)

	// If data is too short, hash it to get more bits
	if len(data) < k.EmbeddingDim*4 {
		hash := sha256.Sum256(data)
		data = append(data, hash[:]...)
	}

	// Fill the vector using a sliding window
	for i := 0; i < min(len(data)/4, k.EmbeddingDim); i++ {
		// Convert 4 bytes to float32 and normalize
		val := float64(binary.BigEndian.Uint32(data[i*4 : i*4+4]))
		vector[i] = val/math.MaxUint32*2 - 1 // Scale to [-1, 1]
	}

	// Center data by subtracting mean
	for i := range vector {
		vector[i] -= k.Mean[i]
	}

	return vector
}

// Compress compresses data using kernel PCA
func (k *CompressionKernel) Compress(data []byte) ([]byte, error) {
	// Convert data to feature vector
	vector := k.DataToFeatureVector(data)

	// Project onto principal components
	projected := make([]float64, len(k.Components))
	for i := range k.Components {
		dot := 0.0
		for j := range vector {
			dot += vector[j] * k.Components[i][j]
		}
		projected[i] = dot
	}

	// Quantize the projected values
	var quantized []byte
	switch k.QuantizeBits {
	case 8:
		quantized = k.quantize8Bit(projected)
	case 16:
		quantized = k.quantize16Bit(projected)
	case 32:
		quantized = k.quantize32Bit(projected)
	}

	// Apply additional zlib compression if enabled
	if k.UseZlib {
		var compressed bytes.Buffer
		w, err := zlib.NewWriterLevel(&compressed, k.ZlibLevel)
		if err != nil {
			return nil, err
		}

		_, err = w.Write(quantized)
		if err != nil {
			return nil, err
		}

		err = w.Close()
		if err != nil {
			return nil, err
		}

		return compressed.Bytes(), nil
	}

	return quantized, nil
}

// Decompress decompresses data using kernel PCA
func (k *CompressionKernel) Decompress(compressed []byte) ([]byte, error) {
	var quantized []byte

	// Apply zlib decompression if enabled
	if k.UseZlib {
		r, err := zlib.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, err
		}
		defer r.Close()

		var decompressed bytes.Buffer
		_, err = io.Copy(&decompressed, r)
		if err != nil {
			return nil, err
		}

		quantized = decompressed.Bytes()
	} else {
		quantized = compressed
	}

	// Dequantize the values
	var projected []float64
	switch k.QuantizeBits {
	case 8:
		projected = k.dequantize8Bit(quantized)
	case 16:
		projected = k.dequantize16Bit(quantized)
	case 32:
		projected = k.dequantize32Bit(quantized)
	}

	// Reconstruct from principal components
	// Note: This is a simplified approximation of the original data
	reconstructed := make([]float64, k.EmbeddingDim)
	for i := range projected {
		for j := range reconstructed {
			reconstructed[j] += projected[i] * k.Components[i][j]
		}
	}

	// Add back the mean
	for i := range reconstructed {
		reconstructed[i] += k.Mean[i]
	}

	// Convert reconstructed vector to bytes
	result := make([]byte, k.EmbeddingDim*4)
	for i := range reconstructed {
		if i < k.EmbeddingDim {
			// Rescale from [-1, 1] to [0, 1]
			normalized := (reconstructed[i] + 1) / 2
			// Clamp to [0, 1]
			if normalized < 0 {
				normalized = 0
			} else if normalized > 1 {
				normalized = 1
			}
			// Convert to uint32 and write to result
			val := uint32(normalized * math.MaxUint32)
			binary.BigEndian.PutUint32(result[i*4:i*4+4], val)
		}
	}

	return result, nil
}

// Helper methods for quantization

func (k *CompressionKernel) quantize8Bit(values []float64) []byte {
	result := make([]byte, len(values))
	for i, v := range values {
		// Scale to range [-127, 127]
		scaled := int(math.Round(v * 127))
		// Clamp
		if scaled < -127 {
			scaled = -127
		} else if scaled > 127 {
			scaled = 127
		}
		// Store as unsigned byte (add 127 to get range [0, 254])
		result[i] = byte(scaled + 127)
	}
	return result
}

func (k *CompressionKernel) quantize16Bit(values []float64) []byte {
	result := make([]byte, len(values)*2)
	for i, v := range values {
		// Scale to range [-32767, 32767]
		scaled := int16(math.Round(v * 32767))
		// Store as 2 bytes
		binary.BigEndian.PutUint16(result[i*2:i*2+2], uint16(scaled))
	}
	return result
}

func (k *CompressionKernel) quantize32Bit(values []float64) []byte {
	result := make([]byte, len(values)*4)
	for i, v := range values {
		// Convert directly to float32
		binary.BigEndian.PutUint32(result[i*4:i*4+4], math.Float32bits(float32(v)))
	}
	return result
}

func (k *CompressionKernel) dequantize8Bit(data []byte) []float64 {
	values := make([]float64, len(data))
	for i, b := range data {
		// Convert from [0, 254] back to [-127, 127]
		scaled := int(b) - 127
		// Convert to float in range [-1, 1]
		values[i] = float64(scaled) / 127
	}
	return values
}

func (k *CompressionKernel) dequantize16Bit(data []byte) []float64 {
	values := make([]float64, len(data)/2)
	for i := range values {
		// Read int16
		val := int16(binary.BigEndian.Uint16(data[i*2 : i*2+2]))
		// Convert to float in range [-1, 1]
		values[i] = float64(val) / 32767
	}
	return values
}

func (k *CompressionKernel) dequantize32Bit(data []byte) []float64 {
	values := make([]float64, len(data)/4)
	for i := range values {
		// Read float32
		val := math.Float32frombits(binary.BigEndian.Uint32(data[i*4 : i*4+4]))
		// Convert to float64
		values[i] = float64(val)
	}
	return values
}

// CompressionStats provides statistics about the compression
type CompressionStats struct {
	OriginalSize     int     // Size of the original data in bytes
	CompressedSize   int     // Size of the compressed data in bytes
	CompressionRatio float64 // Ratio of original size to compressed size
}

// Compress data and return statistics
func (k *CompressionKernel) CompressWithStats(data []byte) ([]byte, CompressionStats, error) {
	compressed, err := k.Compress(data)
	if err != nil {
		return nil, CompressionStats{}, err
	}

	stats := CompressionStats{
		OriginalSize:     len(data),
		CompressedSize:   len(compressed),
		CompressionRatio: float64(len(data)) / float64(len(compressed)),
	}

	return compressed, stats, nil
}
