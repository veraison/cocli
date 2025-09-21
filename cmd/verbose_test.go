// Copyright 2025 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerboseLogging(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	// Test when verbose is disabled
	verbose = false
	
	// Capture stdout
	var buf bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	VerboseInfo("This should not appear")
	VerboseDebug("This should not appear")
	VerboseTrace("This should not appear")
	
	w.Close()
	os.Stdout = originalStdout
	
	// Nothing should be written when verbose is false
	buf.ReadFrom(r)
	assert.Empty(t, buf.String())

	// Test when verbose is enabled
	verbose = true
	
	// Test VerboseInfo
	r, w, _ = os.Pipe()
	os.Stdout = w
	
	VerboseInfo("Test info message with %s", "parameter")
	
	w.Close()
	os.Stdout = originalStdout
	
	buf.Reset()
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "[INFO] Test info message with parameter")

	// Test VerboseDebug
	r, w, _ = os.Pipe()
	os.Stdout = w
	
	VerboseDebug("Test debug message")
	
	w.Close()
	os.Stdout = originalStdout
	
	buf.Reset()
	buf.ReadFrom(r)
	output = buf.String()
	assert.Contains(t, output, "[DEBUG] Test debug message")

	// Test VerboseTrace
	r, w, _ = os.Pipe()
	os.Stdout = w
	
	VerboseTrace("Test trace message")
	
	w.Close()
	os.Stdout = originalStdout
	
	buf.Reset()
	buf.ReadFrom(r)
	output = buf.String()
	assert.Contains(t, output, "[TRACE] Test trace message")
}

func TestGetVerbose(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = false
	assert.False(t, GetVerbose())

	verbose = true
	assert.True(t, GetVerbose())
}

func TestVerboseOperation(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = true

	// Test successful operation
	r, w, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = w

	err := VerboseOperation("test operation", func() error {
		return nil
	})

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "[INFO] Starting test operation...")
	assert.Contains(t, output, "[INFO] test operation completed successfully")

	// Test failed operation
	r, w, _ = os.Pipe()
	os.Stdout = w

	testErr := fmt.Errorf("test error")
	err = VerboseOperation("failing operation", func() error {
		return testErr
	})

	w.Close()
	os.Stdout = originalStdout

	buf.Reset()
	buf.ReadFrom(r)
	output = buf.String()

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Contains(t, output, "[INFO] Starting failing operation...")
	assert.Contains(t, output, "[INFO] failing operation failed: test error")
}

func TestVerboseFileStats(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = true

	r, w, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = w

	VerboseFileStats("test.cbor", 1024)

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "[INFO] Processing file test.cbor (1024 bytes)")
}

func TestVerboseProgress(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = true

	r, w, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = w

	VerboseProgress(3, 10, "files processed")

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "[INFO] Progress: 3/10 files processed")
}

// Integration test for verbose logging in comid display command
func TestComidDisplayVerboseIntegration(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	// This test requires actual CBOR files - we'll create a minimal test
	// that verifies the verbose logging structure is in place
	verbose = true

	// Test that GetVerbose returns true when verbose flag is set
	assert.True(t, GetVerbose())

	// Test that verbose functions can be called without error
	require.NotPanics(t, func() {
		VerboseInfo("Test integration message")
		VerboseDebug("Test debug integration")
		VerboseTrace("Test trace integration")
	})
}

// Test verbose logging with different output levels
func TestVerboseLoggingLevels(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = true

	testCases := []struct {
		name     string
		function func(string, ...interface{})
		level    string
	}{
		{"Info Level", VerboseInfo, "[INFO]"},
		{"Debug Level", VerboseDebug, "[DEBUG]"},
		{"Trace Level", VerboseTrace, "[TRACE]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			originalStdout := os.Stdout
			os.Stdout = w

			tc.function("Test message for %s", tc.name)

			w.Close()
			os.Stdout = originalStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			assert.Contains(t, output, tc.level)
			assert.Contains(t, output, fmt.Sprintf("Test message for %s", tc.name))
		})
	}
}

// Test that verbose logging doesn't interfere with normal operation when disabled
func TestVerboseLoggingNoInterference(t *testing.T) {
	// Save original verbose state
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = false

	// Capture any output
	r, w, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = w

	// Call all verbose functions
	VerboseInfo("Should not appear")
	VerboseDebug("Should not appear")
	VerboseTrace("Should not appear")
	VerboseFileStats("test.file", 100)
	VerboseProgress(1, 10, "test operation")

	// Test VerboseOperation
	err := VerboseOperation("silent operation", func() error {
		return nil
	})

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should have no output when verbose is disabled
	assert.Empty(t, strings.TrimSpace(output))
	assert.NoError(t, err)
}