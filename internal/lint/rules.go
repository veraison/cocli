// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

// Package lint implements a small rule engine to check JSON templates
// against user-supplied static rules.
//
// A rule selects a set of nodes with an RFC 9535 JSONPath expression ("given")
// and applies one or more assertions ("then") to it.  The "count" assertion
// applies to the whole node set; all the others apply to each selected node.
package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/theory/jsonpath"
	"gopkg.in/yaml.v3"
)

// Severity of a rule violation.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

// Range is an inclusive interval; an absent bound is unconstrained.
type Range struct {
	Min *int `yaml:"min"`
	Max *int `yaml:"max"`
}

func (r Range) contains(n int) bool {
	return (r.Min == nil || n >= *r.Min) && (r.Max == nil || n <= *r.Max)
}

func (r Range) String() string {
	switch {
	case r.Min != nil && r.Max != nil && *r.Min == *r.Max:
		return fmt.Sprintf("exactly %d", *r.Min)
	case r.Min != nil && r.Max != nil:
		return fmt.Sprintf("between %d and %d", *r.Min, *r.Max)
	case r.Min != nil:
		return fmt.Sprintf("at least %d", *r.Min)
	case r.Max != nil:
		return fmt.Sprintf("at most %d", *r.Max)
	}
	return "any"
}

// Value wraps an arbitrary YAML/JSON value, so that an explicit null can be
// told apart from an absent key.
type Value struct{ V any }

// UnmarshalYAML implements yaml.Unmarshaler.
func (v *Value) UnmarshalYAML(n *yaml.Node) error { return n.Decode(&v.V) }

// Then holds the assertions of a rule.  All the assertions that are set must
// hold.
type Then struct {
	// Count constrains the number of nodes selected by "given".
	Count *Range `yaml:"count"`

	// The following apply to every selected node.
	Length     *Range `yaml:"length"`      // string (runes), array or object size
	Equals     *Value `yaml:"equals"`      // deep equality
	Enum       []any  `yaml:"enum"`        // must equal one of
	NotEnum    []any  `yaml:"not-enum"`    // must equal none of
	Pattern    string `yaml:"pattern"`     // string must match (unanchored RE2)
	NotPattern string `yaml:"not-pattern"` // string must not match
	Type       string `yaml:"type"`        // JSON type name
	Schema     any    `yaml:"schema"`      // inline JSON Schema
}

// Rule is a single lint rule.
type Rule struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Severity    Severity `yaml:"severity"`
	Message     string   `yaml:"message"`
	Given       string   `yaml:"given"`
	Then        Then     `yaml:"then"`

	path       *jsonpath.Path
	pattern    *regexp.Regexp
	notPattern *regexp.Regexp
	schema     *jsonschema.Schema
}

// RuleSet is a collection of rules, as loaded from a rules file.
type RuleSet struct {
	Rules []*Rule `yaml:"rules"`
}

// Load parses and validates a rule set.  The input is YAML (JSON is accepted
// as a subset).  Unknown keys are rejected, so that typos in a rule do not
// silently disable it.
func Load(data []byte) (*RuleSet, error) {
	var rs RuleSet

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(&rs); err != nil {
		return nil, fmt.Errorf("parsing rules: %w", err)
	}

	if len(rs.Rules) == 0 {
		return nil, errors.New("no rules found")
	}

	seen := map[string]bool{}

	for i, r := range rs.Rules {
		if r == nil {
			return nil, fmt.Errorf("rule #%d: empty", i+1)
		}

		if r.ID == "" {
			return nil, fmt.Errorf("rule #%d: missing id", i+1)
		}

		if seen[r.ID] {
			return nil, fmt.Errorf("rule %q: duplicate id", r.ID)
		}

		seen[r.ID] = true

		if err := r.compile(); err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.ID, err)
		}
	}

	return &rs, nil
}

func (r *Rule) compile() error {
	var err error

	switch r.Severity {
	case "":
		r.Severity = SeverityError
	case SeverityError, SeverityWarn, SeverityInfo:
	default:
		return fmt.Errorf("invalid severity %q", r.Severity)
	}

	if r.Given == "" {
		return errors.New("missing given")
	}

	if r.path, err = jsonpath.Parse(r.Given); err != nil {
		return fmt.Errorf("given: %w", err)
	}

	t := &r.Then

	if t.Count == nil && t.Length == nil && t.Equals == nil && t.Enum == nil &&
		t.NotEnum == nil && t.Pattern == "" && t.NotPattern == "" &&
		t.Type == "" && t.Schema == nil {
		return errors.New("then: no assertion")
	}

	if t.Pattern != "" {
		if r.pattern, err = regexp.Compile(t.Pattern); err != nil {
			return fmt.Errorf("then.pattern: %w", err)
		}
	}

	if t.NotPattern != "" {
		if r.notPattern, err = regexp.Compile(t.NotPattern); err != nil {
			return fmt.Errorf("then.not-pattern: %w", err)
		}
	}

	switch t.Type {
	case "", "string", "number", "integer", "boolean", "object", "array", "null":
	default:
		return fmt.Errorf("then.type: unknown type %q", t.Type)
	}

	// Rule values come from YAML; bring them to the same representation as
	// decoded JSON documents (e.g., float64 numbers) so they compare equal.
	if t.Equals != nil {
		if t.Equals.V, err = normalise(t.Equals.V); err != nil {
			return fmt.Errorf("then.equals: %w", err)
		}
	}

	for _, l := range []struct {
		name string
		vals []any
	}{{"enum", t.Enum}, {"not-enum", t.NotEnum}} {
		for i := range l.vals {
			if l.vals[i], err = normalise(l.vals[i]); err != nil {
				return fmt.Errorf("then.%s: %w", l.name, err)
			}
		}
	}

	if t.Schema != nil {
		doc, err := normalise(t.Schema)
		if err != nil {
			return fmt.Errorf("then.schema: %w", err)
		}

		c := jsonschema.NewCompiler()
		if err := c.AddResource("rule.json", doc); err != nil {
			return fmt.Errorf("then.schema: %w", err)
		}

		if r.schema, err = c.Compile("rule.json"); err != nil {
			return fmt.Errorf("then.schema: %w", err)
		}
	}

	return nil
}

func normalise(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}
