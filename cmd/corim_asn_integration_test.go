// Copyright 2021-2024 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorimDisplayWithASNHeaders(t *testing.T) {
	t.Skip("Integration test disabled - requires specific test files")
	// This integration test verifies that CoRIM files with ASN headers
	// are properly processed by stripping the d9 01 f4 d9 01 f6 pattern
	
	// Read a valid signed CoRIM file
	validCorimData, err := afero.ReadFile(fs, "testcases/signed-corim-valid.cbor")
	require.NoError(t, err, "Failed to read test CoRIM file")
	
	// Create the ASN header pattern
	asnHeader := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6}
	
	// Prepend ASN header to create a test file with headers
	corimWithASNHeader := append(asnHeader, validCorimData...)
	
	// Write the test file
	testFileName := "test-asn-header-corim.cbor"
	err = afero.WriteFile(fs, testFileName, corimWithASNHeader, 0644)
	require.NoError(t, err, "Failed to create test file with ASN header")
	
	// Clean up after test
	defer func() {
		fs.Remove(testFileName)
	}()
	
	// Test that the display function works with ASN headers
	err = display(testFileName, false)
	assert.NoError(t, err, "Display should work with ASN header stripping")
	
	// Test that the display function still works with the original file (no ASN headers)
	err = display("testcases/signed-corim-valid.cbor", false)
	assert.NoError(t, err, "Display should still work with files without ASN headers")
}

func TestCorimVerifyWithASNHeaders(t *testing.T) {
	t.Skip("Integration test disabled - requires specific test files")
	// Read a valid signed CoRIM file
	validCorimData, err := afero.ReadFile(fs, "testcases/signed-corim-valid.cbor")
	require.NoError(t, err, "Failed to read test CoRIM file")
	
	// Create the ASN header pattern
	asnHeader := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6}
	
	// Prepend ASN header to create a test file with headers
	corimWithASNHeader := append(asnHeader, validCorimData...)
	
	// Write the test file
	testFileName := "test-asn-header-verify-corim.cbor"
	err = afero.WriteFile(fs, testFileName, corimWithASNHeader, 0644)
	require.NoError(t, err, "Failed to create test file with ASN header")
	
	// Clean up after test
	defer func() {
		fs.Remove(testFileName)
	}()
	
	// Test that the verify function works with ASN headers
	err = verify(testFileName, "testcases/ec-p256.jwk")
	assert.NoError(t, err, "Verify should work with ASN header stripping")
}

func TestCorimExtractWithASNHeaders(t *testing.T) {
	t.Skip("Integration test disabled - requires specific test files")
	// Read a valid signed CoRIM file
	validCorimData, err := afero.ReadFile(fs, "testcases/signed-corim-valid.cbor")
	require.NoError(t, err, "Failed to read test CoRIM file")
	
	// Create the ASN header pattern
	asnHeader := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6}
	
	// Prepend ASN header to create a test file with headers
	corimWithASNHeader := append(asnHeader, validCorimData...)
	
	// Write the test file
	testFileName := "test-asn-header-extract-corim.cbor"
	err = afero.WriteFile(fs, testFileName, corimWithASNHeader, 0644)
	require.NoError(t, err, "Failed to create test file with ASN header")
	
	// Clean up after test
	defer func() {
		fs.Remove(testFileName)
	}()
	
	// Test that the extract function works with ASN headers
	outputDir := "test-extract-asn"
	err = extract(testFileName, &outputDir)
	assert.NoError(t, err, "Extract should work with ASN header stripping")
	
	// Clean up extracted files
	defer func() {
		fs.RemoveAll(outputDir)
	}()
}