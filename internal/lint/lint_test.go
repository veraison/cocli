// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const doc = `{
  "name": "acme",
  "n": 3,
  "ms": [
    {"key": {"value": "a.x"}, "tags": ["p", "q"], "d": "sha-256:AAAA"},
    {"key": {"value": "a.y"}, "tags": [], "d": "md5:BBBB"},
    {"key": {"value": "b.z"}}
  ]
}`

func check(t *testing.T, rules string) []Finding {
	t.Helper()

	rs, err := Load([]byte(rules))
	require.NoError(t, err)

	d, err := Parse([]byte(doc))
	require.NoError(t, err)

	return rs.Check(d)
}

func Test_count(t *testing.T) {
	tests := []struct {
		name  string
		then  string
		fails bool
	}{
		{"exactly-ok", "count: {min: 2, max: 2}", false},
		{"min-fail", "count: {min: 3}", true},
		{"max-fail", "count: {max: 1}", true},
		{"absent-ok", "count: {max: 0}", false}, // given overridden below
	}

	for _, tt := range tests {
		given := "$.ms[?match(@.key.value, 'a[.].*')]"
		if tt.name == "absent-ok" {
			given = "$.nothing"
		}

		fs := check(t, "rules:\n- id: r\n  given: \""+given+"\"\n  then: {"+tt.then+"}\n")
		if tt.fails {
			require.Len(t, fs, 1, tt.name)
			assert.Equal(t, SeverityError, fs[0].Severity)
			assert.Contains(t, fs[0].Detail, "found 2")
		} else {
			assert.Empty(t, fs, tt.name)
		}
	}
}

func Test_node_assertions(t *testing.T) {
	tests := []struct {
		name   string
		given  string
		then   string
		failed []string // normalized paths of offending nodes
	}{
		{"equals-ok", "$.name", "equals: acme", nil},
		{"equals-fail", "$.name", "equals: other", []string{"$['name']"}},
		{"equals-number", "$.n", "equals: 3", nil},
		{"enum-fail", "$.ms[*].key.value", "enum: [a.x, a.y]", []string{"$['ms'][2]['key']['value']"}},
		{"not-enum", "$.ms[*].key.value", "not-enum: [b.z]", []string{"$['ms'][2]['key']['value']"}},
		{"pattern", "$.ms[*].d", "pattern: '^sha-'", []string{"$['ms'][1]['d']"}},
		{"not-pattern", "$.ms[*].d", "not-pattern: '^md5:'", []string{"$['ms'][1]['d']"}},
		{"pattern-wrong-type", "$.n", "pattern: x", []string{"$['n']"}},
		{"length", "$.ms[*].tags", "length: {min: 1}", []string{"$['ms'][1]['tags']"}},
		{"length-string", "$.name", "length: {max: 3}", []string{"$['name']"}},
		{"length-wrong-type", "$.n", "length: {min: 1}", []string{"$['n']"}},
		{"type", "$.n", "type: integer", nil},
		{"type-fail", "$.name", "type: number", []string{"$['name']"}},
		{"schema", "$.ms[*]", "schema: {required: [d]}", []string{"$['ms'][2]"}},
	}

	for _, tt := range tests {
		fs := check(t, "rules:\n- id: r\n  given: \""+tt.given+"\"\n  then: {"+tt.then+"}\n")

		var got []string
		for _, f := range fs {
			got = append(got, f.Path)
		}

		assert.Equal(t, tt.failed, got, tt.name)
	}
}

func Test_severity_and_message(t *testing.T) {
	fs := check(t, `
rules:
- id: w
  severity: warn
  message: custom
  given: $.name
  then: {equals: x}
`)
	require.Len(t, fs, 1)
	assert.Equal(t, SeverityWarn, fs[0].Severity)
	assert.Equal(t, "custom", fs[0].Message)
	assert.False(t, HasErrors(fs))
	assert.Contains(t, fs[0].String(), "warn: w: $['name']: custom")
}

func Test_Load_errors(t *testing.T) {
	tests := []struct{ name, rules, err string }{
		{"empty", "", "parsing rules"},
		{"no-rules", "rules: []", "no rules found"},
		{"no-id", "rules: [{given: $, then: {count: {min: 1}}}]", "missing id"},
		{"dup-id", "rules: [{id: a, given: $, then: {count: {min: 1}}}, {id: a, given: $, then: {count: {min: 1}}}]", "duplicate id"},
		{"no-given", "rules: [{id: a, then: {count: {min: 1}}}]", "missing given"},
		{"bad-path", "rules: [{id: a, given: '$.a-b', then: {count: {min: 1}}}]", "given:"},
		{"no-then", "rules: [{id: a, given: $}]", "no assertion"},
		{"bad-severity", "rules: [{id: a, severity: fatal, given: $, then: {count: {min: 1}}}]", "invalid severity"},
		{"bad-regex", "rules: [{id: a, given: $, then: {pattern: '('}}]", "then.pattern"},
		{"bad-type", "rules: [{id: a, given: $, then: {type: nope}}]", "unknown type"},
		{"bad-schema", "rules: [{id: a, given: $, then: {schema: {type: 3}}}]", "then.schema"},
		{"unknown-key", "rules: [{id: a, given: $, thenn: {}}]", "thenn"},
	}

	for _, tt := range tests {
		_, err := Load([]byte(tt.rules))
		assert.ErrorContains(t, err, tt.err, tt.name)
	}
}

func Test_Parse_error(t *testing.T) {
	_, err := Parse([]byte("{"))
	assert.ErrorContains(t, err, "parsing JSON")
}

// A filter that refers to a member that some of the selected nodes lack must
// skip those nodes instead of failing.
func Test_filter_on_missing_member(t *testing.T) {
	fs := check(t, `
rules:
- id: r
  given: $.ms[?match(@.tags[0], 'p')]
  then: {count: {min: 1}}
`)
	assert.Empty(t, fs)
}
