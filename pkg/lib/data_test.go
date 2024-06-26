package telemetrylib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShouldCompress tests ShouldCompress function
func TestShouldCompress(t *testing.T) {
	mockCompressFalse := []byte(`{"test": false}`)
	mockCompressTrue := []byte(`{
		"name": "This is a test",
		"test": "This should be compressed",
		"additional_data": "Aliqua enim officia eiusmod ad. Officia cillum dolore occaecat consectetur amet dolore commodo adipisicing ut ut. Sit eiusmod aliquip occaecat laborum aliquip qui duis ut elit duis. Eiusmod ullamco elit Lorem nostrud consequat adipisicing quis cupidatat. Aliqua nulla ad aliqua exercitation amet ea excepteur nisi anim officia in voluptate commodo exercitation. Minim cupidatat proident aliquip minim officia id occaecat ea est Lorem nulla irure nulla excepteur."
	}`)

	_, compressedTrue, err := ShouldCompress(mockCompressTrue)
	if err != nil {
		t.Fatalf("Error determining whether or not to compress data: %v", err)
	}
	assert.Equal(t, true, compressedTrue)

	_, compressedFalse, err := ShouldCompress(mockCompressFalse)
	if err != nil {
		t.Fatalf("Error determining whether or not to compress data: %v", err)
	}
	assert.Equal(t, false, compressedFalse)
}
