// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripASNHeaderBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "data with ASN header",
			input:    []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6, 0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "data without ASN header",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "empty data",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "data shorter than ASN header",
			input:    []byte{0xd9, 0x01, 0xf4},
			expected: []byte{0xd9, 0x01, 0xf4},
		},
		{
			name:     "data starting with partial ASN header",
			input:    []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0x99, 0x01, 0x02},
			expected: []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0x99, 0x01, 0x02},
		},
		{
			name:     "data with ASN header only",
			input:    []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6},
			expected: []byte{},
		},
		{
			name:     "data with ASN header in middle (should not strip)",
			input:    []byte{0x01, 0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6, 0x02},
			expected: []byte{0x01, 0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6, 0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripASNHeaderBytes(tt.input)
			assert.Equal(t, tt.expected, result, "stripASNHeaderBytes() result mismatch")
		})
	}
}

func TestStripASNHeaderBytes_Immutability(t *testing.T) {
	// Test that the original slice is not modified
	original := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6, 0x01, 0x02, 0x03, 0x04}
	originalCopy := make([]byte, len(original))
	copy(originalCopy, original)

	result := stripASNHeaderBytes(original)

	// Original should remain unchanged
	assert.Equal(t, originalCopy, original, "Original slice was modified")
	
	// Result should be the stripped version
	expected := []byte{0x01, 0x02, 0x03, 0x04}
	assert.Equal(t, expected, result, "Result should be stripped")
}

func TestStripASNHeaderBytes_RealWorldScenario(t *testing.T) {
	// Test with a scenario similar to the real-world example from the issue
	// Simulating the beginning of a CoRIM file with ASN header
	asnHeader := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6}
	corimData := []byte{0xd2, 0x84, 0x58, 0x29, 0xa3, 0x01, 0x38, 0x22}
	
	inputWithHeader := append(asnHeader, corimData...)
	
	result := stripASNHeaderBytes(inputWithHeader)
	
	assert.Equal(t, corimData, result, "Should strip ASN header and return CoRIM data")
	assert.True(t, bytes.HasPrefix(inputWithHeader, asnHeader), "Input should start with ASN header")
	assert.False(t, bytes.HasPrefix(result, asnHeader), "Result should not start with ASN header")
}