// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/cocli/internal/lint"
)

func lintFixture(t *testing.T, rules string, templates map[string]string) {
	t.Helper()

	fs = afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "rules.yaml", []byte(rules), 0400))

	for name, content := range templates {
		require.NoError(t, afero.WriteFile(fs, name, []byte(content), 0400))
	}
}

func Test_LintCmd_unknown_argument(t *testing.T) {
	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--unknown-argument=val"})

	assert.EqualError(t, cmd.Execute(), "unknown flag: --unknown-argument")
}

func Test_LintCmd_no_rules(t *testing.T) {
	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--file=a.json"})

	assert.EqualError(t, cmd.Execute(), "no rules file supplied")
}

func Test_LintCmd_no_files(t *testing.T) {
	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml"})

	assert.EqualError(t, cmd.Execute(), "no files or directories supplied")
}

func Test_LintCmd_rules_not_found(t *testing.T) {
	fs = afero.NewMemMapFs()

	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--file=a.json"})

	assert.ErrorContains(t, cmd.Execute(), "reading rules")
}

func Test_LintCmd_invalid_rules(t *testing.T) {
	lintFixture(t, "rules: []", nil)

	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--file=a.json"})

	assert.EqualError(t, cmd.Execute(), "no rules found")
}

func Test_LintCmd_no_files_found(t *testing.T) {
	lintFixture(t, "rules: [{id: a, given: $, then: {type: object}}]", nil)

	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--file=a.json", "--dir=nope"})

	assert.EqualError(t, cmd.Execute(), "no files found")
}

func Test_LintCmd_invalid_json(t *testing.T) {
	lintFixture(t, "rules: [{id: a, given: $, then: {type: object}}]",
		map[string]string{"a.json": "{"})

	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--file=a.json"})

	assert.EqualError(t, cmd.Execute(), "1/1 file(s) failed lint")
}

func Test_LintCmd_pass_fail_warn(t *testing.T) {
	rules := `
rules:
  - id: has-name
    given: $.name
    then: {count: {min: 1}}
  - id: soft
    severity: warn
    given: $.name
    then: {equals: other}
`
	lintFixture(t, rules, map[string]string{
		"d/ok.json":  `{"name": "x"}`, // warn only
		"d/bad.json": `{}`,
		"d/skip.txt": `not json`,
	})

	cmd := NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--file=d/ok.json"})
	assert.NoError(t, cmd.Execute())

	cmd = NewLintCmd()
	cmd.SetArgs([]string{"--rules=rules.yaml", "--dir=d"})
	assert.EqualError(t, cmd.Execute(), "1/2 file(s) failed lint")
}

// Conformance tests for the example rule sets in data/lint

type obj = map[string]any

func loadRules(t *testing.T, name string) *lint.RuleSet {
	t.Helper()

	data, err := os.ReadFile("../data/lint/" + name)
	require.NoError(t, err)

	rs, err := lint.Load(data)
	require.NoError(t, err)

	return rs
}

func loadDoc(t *testing.T, path string) obj {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc obj
	require.NoError(t, json.Unmarshal(data, &doc))

	return doc
}

func firstTriple(doc obj, kind string) obj {
	return doc["triples"].(obj)[kind].([]any)[0].(obj)
}

func measurements(doc obj) []any {
	return firstTriple(doc, "reference-values")["measurements"].([]any)
}

func measurement(doc obj, key string) obj {
	for _, m := range measurements(doc) {
		if m.(obj)["key"].(obj)["value"] == key {
			return m.(obj)
		}
	}

	panic("no measurement " + key)
}

func setMeasurements(doc obj, ms []any) {
	firstTriple(doc, "reference-values")["measurements"] = ms
}

func dropMeasurement(doc obj, key string) {
	var out []any

	for _, m := range measurements(doc) {
		if m.(obj)["key"].(obj)["value"] != key {
			out = append(out, m)
		}
	}

	setMeasurements(doc, out)
}

func dup(m obj) obj {
	b, _ := json.Marshal(m)

	var out obj
	_ = json.Unmarshal(b, &out)

	return out
}

type conformanceCase struct {
	name   string
	mutate func(doc obj)
	want   []string // IDs of the rules expected to fire, in order; none means clean
}

func runConformance(t *testing.T, rules, fixture string, cases []conformanceCase) {
	t.Helper()

	rs := loadRules(t, rules)

	for _, tc := range cases {
		doc := loadDoc(t, fixture)
		if tc.mutate != nil {
			tc.mutate(doc)
		}

		b, err := json.Marshal(doc)
		require.NoError(t, err)

		parsed, err := lint.Parse(b)
		require.NoError(t, err)

		var got []string
		for _, f := range rs.Check(parsed) {
			got = append(got, f.RuleID)
		}

		assert.Equal(t, tc.want, got, tc.name)
	}
}

func Test_LintRules_psa_refval(t *testing.T) {
	ck := func(doc obj) obj { return measurements(doc)[0].(obj)["value"].(obj) }

	runConformance(t, "psa.yaml", "testcases/lint/psa-refval.json", []conformanceCase{
		{"conformant", nil, nil},
		{"mkey", func(d obj) { measurements(d)[0].(obj)["key"].(obj)["value"] = "psa.sw" }, []string{"psa-swcomp-mkey"}},
		{"authorized-by", func(d obj) {
			measurements(d)[0].(obj)["authorized-by"] = []any{obj{"type": "bytes", "value": "AQ=="}}
		}, []string{"psa-swcomp-authorized-by"}},
		{"no-digests", func(d obj) { delete(ck(d), "digests") }, []string{"psa-swcomp-mandatory-fields"}},
		{"two-cryptokeys", func(d obj) {
			keys := ck(d)["cryptokeys"].([]any)
			ck(d)["cryptokeys"] = append(keys, keys[0])
		}, []string{"psa-swcomp-mandatory-fields"}},
		{"numeric-alg", func(d obj) { ck(d)["digests"].([]any)[0].([]any)[0] = 1 }, []string{"psa-swcomp-digest"}},
		{"short-digest", func(d obj) { ck(d)["digests"].([]any)[0].([]any)[1] = "AAAA" }, []string{"psa-swcomp-digest"}},
		{"short-signer-id", func(d obj) { ck(d)["cryptokeys"].([]any)[0].(obj)["value"] = "AAAA" }, []string{"psa-swcomp-signer-id"}},
		{"version-scheme", func(d obj) {
			ck(d)["version"] = obj{"value": "1.0.0", "scheme": "semver"}
		}, []string{"psa-swcomp-version-scheme"}},
		{"short-impl-id", func(d obj) {
			firstTriple(d, "reference-values")["environment"].(obj)["class"].(obj)["id"].(obj)["value"] = "AAAA"
		}, []string{"psa-implementation-id"}},
		{"cert-num", func(d obj) { ck(d)["psa-cert-num"] = "123 - 45" }, []string{"psa-cert-num"}},
	})
}

func Test_LintRules_psa_corim_profile(t *testing.T) {
	runConformance(t, "psa.yaml", "../data/corim/templates/corim-full.json", []conformanceCase{
		{"conformant", nil, nil},
		{"profile", func(d obj) { d["profile"] = "tag:arm.com,2025:cca_platform#1.0.0" }, []string{"psa-profile"}},
	})
}

func Test_LintRules_psa_avk(t *testing.T) {
	avk := func(d obj) obj { return firstTriple(d, "attester-verification-keys") }

	runConformance(t, "psa.yaml", "../data/comid/templates/comid-psa-iakpub.json", []conformanceCase{
		{"conformant", nil, nil},
		{"two-keys", func(d obj) {
			keys := avk(d)["verification-keys"].([]any)
			avk(d)["verification-keys"] = append(keys, keys[0])
		}, []string{"psa-avk-single-key"}},
		{"key-type", func(d obj) {
			avk(d)["verification-keys"].([]any)[0].(obj)["type"] = "cose-key"
		}, []string{"psa-avk-type"}},
		{"instance-id", func(d obj) {
			avk(d)["environment"].(obj)["instance"].(obj)["value"] = "AAAA"
		}, []string{"psa-instance-id"}},
	})
}

func Test_LintRules_cca_platform_refval(t *testing.T) {
	const fixture = "testcases/lint/cca-platform-refval.json"

	setKey := func(from, to string) func(d obj) {
		return func(d obj) { measurement(d, from)["key"].(obj)["value"] = to }
	}

	runConformance(t, "cca-platform.yaml", fixture, []conformanceCase{
		{"conformant", nil, nil},
		{"unknown-mkey", setKey("cca.software-component", "cca.sw"), []string{"cca-platform-mkey"}},
		{"old-style-rotpk-mkey", setKey("cca.rotpk.CM.0.0", "cca.rotpk.CM.3"), []string{"cca-platform-mkey"}},
		{"rotpk-array-out-of-range", setKey("cca.rotpk.CM.0.0", "cca.rotpk.CM.8.0"), []string{"cca-platform-mkey"}},
		{"rotpk-entry-out-of-range", setKey("cca.rotpk.CM.0.0", "cca.rotpk.DM.0.6"), []string{"cca-platform-mkey"}},
		{"rotpk-two-keys", func(d obj) {
			v := measurement(d, "cca.rotpk.CM.0.0")["value"].(obj)
			keys := v["cryptokeys"].([]any)
			v["cryptokeys"] = append(keys, keys[0])
		}, []string{"cca-platform-rotpk-shape"}},
		{"rotpk-short-key", func(d obj) {
			measurement(d, "cca.rotpk.CM.0.0")["value"].(obj)["cryptokeys"].([]any)[0].(obj)["value"] = "AAAA"
		}, []string{"cca-platform-signer-id"}},
		{"authorized-by", func(d obj) {
			measurement(d, "cca.platform-config")["authorized-by"] = []any{obj{"type": "bytes", "value": "AQ=="}}
		}, []string{"cca-platform-authorized-by"}},
		{"two-platform-configs", func(d obj) {
			setMeasurements(d, append(measurements(d), dup(measurement(d, "cca.platform-config"))))
		}, []string{"cca-platform-config-count"}},
		{"two-manufacturing-configs", func(d obj) {
			setMeasurements(d, append(measurements(d), dup(measurement(d, "cca.platform-manufacturing-config"))))
		}, []string{"cca-platform-manufacturing-config-count"}},
		{"no-platform-config", func(d obj) { dropMeasurement(d, "cca.platform-config") }, []string{"cca-platform-config-present"}},
		{"config-not-masked", func(d obj) {
			measurement(d, "cca.platform-config")["value"].(obj)["raw-value"] = obj{"type": "bytes", "value": "AQID"}
		}, []string{"cca-platform-config-shape"}},
		{"swcomp-numeric-alg", func(d obj) {
			measurement(d, "cca.software-component")["value"].(obj)["digests"].([]any)[0].([]any)[0] = 1
		}, []string{"cca-platform-swcomp-digest"}},
		{"swcomp-no-cryptokeys", func(d obj) {
			delete(measurement(d, "cca.software-component")["value"].(obj), "cryptokeys")
		}, []string{"cca-platform-swcomp-mandatory-fields"}},
		{"swcomp-version-scheme", func(d obj) {
			measurement(d, "cca.software-component")["value"].(obj)["version"] = obj{"value": "1", "scheme": "semver"}
		}, []string{"cca-platform-swcomp-version-scheme"}},
		{"short-impl-id", func(d obj) {
			firstTriple(d, "reference-values")["environment"].(obj)["class"].(obj)["id"].(obj)["value"] = "AAAA"
		}, []string{"cca-platform-implementation-id"}},
	})
}

func Test_LintRules_cca_platform_profile(t *testing.T) {
	runConformance(t, "cca-platform.yaml", "testcases/lint/cca-platform-corim.json", []conformanceCase{
		{"conformant", nil, nil},
		{"profile", func(d obj) { d["profile"] = "tag:arm.com,2025:cca_platform#1.0.0" }, []string{"cca-platform-profile"}},
	})
}

func Test_LintRules_cca_realm_refval(t *testing.T) {
	const fixture = "../data/comid/templates/comid-cca-realm-refval.json"

	runConformance(t, "cca-realm.yaml", fixture, []conformanceCase{
		{"conformant", nil, nil},
		{"no-rim", func(d obj) { dropMeasurement(d, "cca.rim") }, []string{"cca-realm-rim-mandatory"}},
		{"unknown-mkey", func(d obj) { measurement(d, "cca.rem0")["key"].(obj)["value"] = "cca.rem4" }, []string{"cca-realm-mkey"}},
		{"no-digests", func(d obj) { delete(measurement(d, "cca.rem1")["value"].(obj), "digests") }, []string{"cca-realm-digests"}},
		{"rpv-not-bytes", func(d obj) {
			measurement(d, "cca.rpv")["value"].(obj)["raw-value"].(obj)["type"] = "masked"
		}, []string{"cca-realm-rpv"}},
		{"authorized-by", func(d obj) {
			measurement(d, "cca.rim")["authorized-by"] = []any{obj{"type": "bytes", "value": "AQ=="}}
		}, []string{"cca-realm-authorized-by"}},
	})

	runConformance(t, "cca-realm.yaml", "testcases/lint/cca-realm-corim.json", []conformanceCase{
		{"conformant", nil, nil},
		{"profile", func(d obj) { d["profile"] = "tag:arm.com,2025:cca_realm#1.0.0" }, []string{"cca-realm-profile"}},
	})
}
