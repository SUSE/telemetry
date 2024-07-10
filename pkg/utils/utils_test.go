package utils

import (
	"bytes"
	"testing"
)

// TestCompressDecompressGZIP tests both CompressGZIP and DecompressGZIP functions.
func TestCompressDecompressGZIP(t *testing.T) {
	mockData := []byte(`{"test": "This is a JSON file"}`)

	// Compress the data
	compressedData, err := CompressGZIP(mockData)
	if err != nil {
		t.Errorf("Error compressing data: %v", err)
	}

	// Decompress the data to verify
	decompressedData, err := DecompressGZIP(compressedData)
	if err != nil {
		t.Errorf("Error decompressing data: %v", err)
	}

	if !bytes.Equal(mockData, decompressedData) {
		t.Errorf("Decompressed data does not match original data:\n Expected: %v, Got: %v", mockData, decompressedData)
	}
}
