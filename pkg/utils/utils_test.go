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
		t.Fatalf("Error compressing data: %v", err)
	}

	// Decompress the data to verify
	decompressedData, err := DecompressGZIP(compressedData)
	if err != nil {
		t.Fatalf("Error decompressing data: %v", err)
	}

	if !bytes.Equal(mockData, decompressedData) {
		t.Fatalf("Decompressed data does not match original data:\n Expected: %v, Got: %v", mockData, decompressedData)
	}
}

// TestHumanReadableSize tests HumanReadableSize function
func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		mockData []byte
		expected string
	}{
		{make([]byte, 500), "500 B"},
		{make([]byte, 1024), "1.0 KB"},
		{make([]byte, 1536), "1.5 KB"},
		{make([]byte, 1048576), "1.0 MB"},
		{make([]byte, 1073741824), "1.0 GB"},
	}

	for _, test := range tests {
		result := HumanReadableSize(test.mockData)
		if result != test.expected {
			t.Errorf("Error generating human readable size. Generated: %s; Expected %s", result, test.expected)
		}
	}
}
