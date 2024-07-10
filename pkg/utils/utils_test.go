package utils

import (
	"bytes"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCompressDecompressGZIP tests both CompressGZIP and DecompressGZIP functions.
func TestCompressDecompressGZIP(t *testing.T) {
	mockData := []byte(`{"test": "This is a JSON file"}`)

	// Compress data
	compressedData, err := CompressGZIP(mockData)
	if err != nil {
		t.Errorf("Error compressing data: %v", err)
	}

	// Decompress data to verify
	decompressedData, err := DecompressGZIP(compressedData)
	if err != nil {
		t.Errorf("Error decompressing data: %v", err)
	}

	if !bytes.Equal(mockData, decompressedData) {
		t.Fatalf("Decompressed data does not match original data:\n Expected: %v\n Got: %v", string(mockData), string(decompressedData))
	}
}

// TestCompressWhenNeeded tests CompressWhenNeeded function
func TestCompressWhenNeeded(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectCompress bool
	}{
		{
			name:           "Should not compress",
			data:           []byte(`{"test": false}`),
			expectCompress: false,
		},
		{
			name: "Should compress",
			data: []byte(`{
				"name": "This is a test",
				"test": "This should be compressed",
				"additional_data": "Aliqua enim officia eiusmod ad. Officia cillum dolore occaecat consectetur amet dolore commodo adipisicing ut ut. Sit eiusmod aliquip occaecat laborum aliquip qui duis ut elit duis. Eiusmod ullamco elit Lorem nostrud consequat adipisicing quis cupidatat. Aliqua nulla ad aliqua exercitation amet ea excepteur nisi anim officia in voluptate commodo exercitation. Minim cupidatat proident aliquip minim officia id occaecat ea est Lorem nulla irure nulla excepteur."
			}`),
			expectCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressedData, compression, err := CompressWhenNeeded(tt.data)
			if err != nil {
				t.Fatalf("Error determining whether or not to compress data: %v", err)
			}

			if tt.expectCompress {
				assert.NotNil(t, compression)
				assert.Equal(t, "gzip", *compression)
				assert.NotEqual(t, tt.data, compressedData)
			} else {
				assert.Nil(t, compression)
				assert.Equal(t, tt.data, compressedData)
			}
		})
	}
}

// TestDecompressWhenNeeded tests DecompressWhenNeeded function
func TestDecompressWhenNeeded(t *testing.T) {
	compressedData, _ := CompressGZIP([]byte("test data"))
	tests := []struct {
		name         string
		data         []byte
		compression  sql.NullString
		expectedData []byte
		expectErr    bool
	}{
		{
			name:         "No compression",
			data:         []byte("test data"),
			compression:  sql.NullString{Valid: false},
			expectedData: []byte("test data"),
			expectErr:    false,
		},
		{
			name:         "Valid compression with successful decompression",
			data:         compressedData,
			compression:  sql.NullString{String: "gzip", Valid: true},
			expectedData: []byte("test data"),
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultData, err := DecompressWhenNeeded(tt.data, tt.compression)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if !bytes.Equal(resultData, tt.expectedData) {
				t.Errorf("expected data: %s, got: %s", tt.expectedData, resultData)
			}
		})
	}
}

// TestHumanReadableSize tests HumanReadableSize function
func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		mockData []byte
		expected string
	}{
		{make([]byte, 500), "500 B"},
		{make([]byte, 1024), "1.000 KiB"},
		{make([]byte, 1536), "1.500 KiB"},
		{make([]byte, 1048576), "1.000 MiB"},
		{make([]byte, 1073741824), "1.000 GiB"},
	}

	for _, test := range tests {
		result := HumanReadableSize(test.mockData)
		if result != test.expected {
			t.Errorf("Error generating human readable size. Generated: %s; Expected %s", result, test.expected)
		}
	}
}
