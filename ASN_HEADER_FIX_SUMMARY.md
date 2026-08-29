# ASN Header Bytes Support in cocli

## Issue Summary
GitHub Issue: [#23 - Enhance cocli corim command to skip over ASN header bytes](https://github.com/veraison/cocli/issues/23)

Several vendors distribute CoRIM manifest files with ASN header bytes (`d9 01 f4 d9 01 f6`) at the beginning. Previously, cocli would fail to process these files with errors like:

```
Error: error decoding signed CoRIM from file.cbor: failed CBOR decoding for COSE-Sign1 signed CoRIM: cbor: invalid COSE_Sign1_Tagged object
```

## Solution Implemented

### Changes Made

1. **Added ASN Header Stripping Function** (`cmd/common.go`):
   - Added `stripASNHeaderBytes()` function that detects and removes the ASN header pattern `d9 01 f4 d9 01 f6`
   - Function is safe and only strips headers when the exact pattern is found at the beginning of the data
   - Returns original data unchanged if no ASN header is detected

2. **Updated CoRIM Commands**:
   - **`corim display`** (`cmd/corimDisplay.go`): Added ASN header stripping before CBOR decoding
   - **`corim verify`** (`cmd/corimVerify.go`): Added ASN header stripping before COSE signature verification
   - **`corim extract`** (`cmd/corimExtract.go`): Added ASN header stripping before tag extraction

3. **Preserved corim submit**: The `corim submit` command was intentionally left unchanged as it should preserve the original file format when submitting to servers.

### Implementation Details

The ASN header bytes `d9 01 f4 d9 01 f6` represent:
- `tagged-corim-type-choice #6.500` (`d9 01 f4`)
- `tagged-signed-corim #6.502` (`d9 01 f6`)

These are remnants from an older draft of the CoRIM specification and are automatically detected and stripped.

### Code Example

```go
// stripASNHeaderBytes removes ASN header bytes from CoRIM files if present.
func stripASNHeaderBytes(data []byte) []byte {
    // ASN header pattern: d9 01 f4 d9 01 f6
    asnHeaderPattern := []byte{0xd9, 0x01, 0xf4, 0xd9, 0x01, 0xf6}
    
    // Check if the data starts with the ASN header pattern
    if len(data) >= len(asnHeaderPattern) && bytes.HasPrefix(data, asnHeaderPattern) {
        // Strip the ASN header bytes
        return data[len(asnHeaderPattern):]
    }
    
    // Return original data if no ASN header is found
    return data
}
```

## Testing

### Unit Tests
- Comprehensive unit tests for `stripASNHeaderBytes()` function covering:
  - Files with ASN headers
  - Files without ASN headers
  - Edge cases (empty data, partial headers, etc.)
  - Data integrity (original slice remains unmodified)

### Integration Tests
- End-to-end tests for all affected CoRIM commands
- Tests with real CoRIM files that have ASN headers prepended
- Verification that existing functionality remains unchanged

### Test Results
All existing tests pass, confirming backward compatibility:
```bash
$ make test
PASS
ok      github.com/veraison/cocli/cmd   1.159s
```

## Usage Examples

### Before Fix
```bash
$ cocli corim display -f PS10xx-G75YG100-E3S-16TB.cbor
Error: error decoding CoRIM (signed or unsigned) from PS10xx-G75YG100-E3S-16TB.cbor: expected map (CBOR Major Type 5), found Major Type 6
```

### After Fix
```bash
$ cocli corim display -f PS10xx-G75YG100-E3S-16TB.cbor
Meta:
{
  "signer": {
    "name": "...",
    "uri": "..."
  },
  ...
}
CoRIM:
{
  "corim-id": "...",
  ...
}
```

## Verification Commands

All CoRIM processing commands now work seamlessly with files that have ASN headers:

```bash
# Display CoRIM content
cocli corim display -f corim-with-asn-headers.cbor

# Verify CoRIM signature  
cocli corim verify -f corim-with-asn-headers.cbor -k signing-key.jwk

# Extract embedded tags
cocli corim extract -f corim-with-asn-headers.cbor -o output-dir/
```

## Backwards Compatibility

- ✅ Files without ASN headers continue to work exactly as before
- ✅ All existing functionality is preserved
- ✅ No breaking changes to command-line interface
- ✅ No performance impact for files without ASN headers

## Files Modified

1. `cmd/common.go` - Added `stripASNHeaderBytes()` function
2. `cmd/corimDisplay.go` - Added ASN header stripping to display command
3. `cmd/corimVerify.go` - Added ASN header stripping to verify command  
4. `cmd/corimExtract.go` - Added ASN header stripping to extract command
5. `cmd/common_test.go` - Added comprehensive unit tests
6. `cmd/corim_asn_integration_test.go` - Added integration tests

## Resolution

This fix resolves GitHub issue #23 by automatically detecting and stripping ASN header bytes from CoRIM files, allowing cocli to process vendor-distributed CoRIM files without requiring manual preprocessing. Users no longer need to manually strip the first 6 bytes before using cocli commands.