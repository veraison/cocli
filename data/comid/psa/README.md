# PSA Profile Support in CoRIM CLI

This directory contains example templates and CoMID files for working with the new PSA profile according to draft-fdb-rats-psa-endorsements-08.

## Profile URI

The new PSA profile uses the URI: `tag:arm.com,2025:psa#1.0.0`

## Example Files

### CoRIM Template
- `corim-psa.json` - CoRIM template with the new PSA profile URI

### CoMID Examples
- `psa-reference-values.json` - PSA software component reference values
- `psa-attestation-key.json` - PSA Initial Attestation Key (IAK) verification key
- `psa-certification-claims.json` - PSA Certified Security Assurance Certificate claims
- `psa-software-relations.json` - PSA software update/patch relationships

## Usage Examples

### Create a PSA CoRIM from JSON templates
```bash
# First convert JSON CoMIDs to CBOR format
cocli comid create --template=data/comid/psa/psa-reference-values.json --output=psa-ref-vals.cbor
cocli comid create --template=data/comid/psa/psa-attestation-key.json --output=psa-attest-key.cbor  
cocli comid create --template=data/comid/psa/psa-certification-claims.json --output=psa-cert-claims.cbor
cocli comid create --template=data/comid/psa/psa-software-relations.json --output=psa-sw-rels.cbor

# Create the PSA CoRIM
cocli corim create --template=data/corim/templates/corim-psa.json \
                   --comid=psa-ref-vals.cbor \
                   --comid=psa-attest-key.cbor \
                   --comid=psa-cert-claims.cbor \
                   --comid=psa-sw-rels.cbor \
                   --output=psa-corim.cbor
```

### Submit PSA CoRIM to Veraison
```bash
cocli corim submit \
        --corim-file=psa-corim.cbor \
        --api-server="https://veraison.example/endorsement-provisioning/v1/submit" \
        --media-type="application/corim-unsigned+cbor; profile=tag:arm.com,2025:psa#1.0.0"
```

### Display PSA CoRIM contents
```bash
cocli corim display --corim-file=psa-corim.cbor
```

## PSA Profile Features

The new PSA profile supports:

1. **Reference Values**: Measurements of PSA RoT firmware components
2. **Attestation Verification Keys**: Public keys for verifying PSA attestation tokens
3. **Certification Claims**: Links to PSA Certified Security Assurance Certificates
4. **Software Relations**: Modeling of software updates and patches with security criticality flags
