// Copyright 2026 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package lint

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Finding is a rule violation.
type Finding struct {
	RuleID   string
	Severity Severity
	// Path locates the offending node as an RFC 9535 normalized path.  For
	// "count" violations it is the rule's "given" expression, since there is
	// no single offending node.
	Path    string
	Message string
	Detail  string
}

func (f Finding) String() string {
	s := fmt.Sprintf("%s: %s: %s", f.Severity, f.RuleID, f.Path)
	if f.Message != "" {
		s += ": " + f.Message
	}

	if f.Detail != "" {
		s += " (" + f.Detail + ")"
	}

	return s
}

// HasErrors returns true if any of the findings has error severity.
func HasErrors(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == SeverityError {
			return true
		}
	}

	return false
}

// Parse decodes a JSON document for use with Check.
func Parse(data []byte) (any, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return doc, nil
}

// Check applies the rules to doc, which must have been obtained from Parse (or
// json.Unmarshal into an any).  Findings are returned in rule order, then in
// document order.
func (rs *RuleSet) Check(doc any) []Finding {
	var out []Finding

	for _, r := range rs.Rules {
		out = append(out, r.check(doc)...)
	}

	return out
}

func (r *Rule) finding(path, detail string) Finding {
	return Finding{
		RuleID:   r.ID,
		Severity: r.Severity,
		Path:     path,
		Message:  r.Message,
		Detail:   detail,
	}
}

func (r *Rule) check(doc any) (out []Finding) {
	// Be defensive against bugs in the JSONPath engine: a rule that cannot be
	// evaluated must be reported, not crash the process.
	defer func() {
		if p := recover(); p != nil {
			out = append(out, Finding{
				RuleID:   r.ID,
				Severity: SeverityError,
				Path:     r.Given,
				Detail:   fmt.Sprintf("rule could not be evaluated: %v", p),
			})
		}
	}()

	nodes := r.path.SelectLocated(doc)

	if c := r.Then.Count; c != nil && !c.contains(len(nodes)) {
		out = append(out, r.finding(
			r.Given, fmt.Sprintf("found %d, want %s", len(nodes), c),
		))
	}

	for _, n := range nodes {
		for _, detail := range r.checkNode(n.Node) {
			out = append(out, r.finding(n.Path.String(), detail))
		}
	}

	return out
}

// checkNode returns a description of every assertion that node violates.
func (r *Rule) checkNode(node any) []string {
	var out []string

	t := &r.Then

	if t.Type != "" && !isType(node, t.Type) {
		out = append(out, fmt.Sprintf("want type %s, got %s", t.Type, typeName(node)))
	}

	if t.Length != nil {
		if n, ok := length(node); !ok {
			out = append(out, fmt.Sprintf("length not applicable to %s", typeName(node)))
		} else if !t.Length.contains(n) {
			out = append(out, fmt.Sprintf("length is %d, want %s", n, t.Length))
		}
	}

	if t.Equals != nil && !reflect.DeepEqual(node, t.Equals.V) {
		out = append(out, fmt.Sprintf("want %s, got %s", show(t.Equals.V), show(node)))
	}

	if t.Enum != nil && !contains(t.Enum, node) {
		out = append(out, fmt.Sprintf("%s not in %s", show(node), show(t.Enum)))
	}

	if t.NotEnum != nil && contains(t.NotEnum, node) {
		out = append(out, fmt.Sprintf("%s is forbidden", show(node)))
	}

	if r.pattern != nil {
		if s, ok := node.(string); !ok {
			out = append(out, fmt.Sprintf("pattern not applicable to %s", typeName(node)))
		} else if !r.pattern.MatchString(s) {
			out = append(out, fmt.Sprintf("%q does not match %q", s, t.Pattern))
		}
	}

	if r.notPattern != nil {
		if s, ok := node.(string); !ok {
			out = append(out, fmt.Sprintf("not-pattern not applicable to %s", typeName(node)))
		} else if r.notPattern.MatchString(s) {
			out = append(out, fmt.Sprintf("%q matches forbidden %q", s, t.NotPattern))
		}
	}

	if r.schema != nil {
		if err := r.schema.Validate(node); err != nil {
			out = append(out, "schema: "+schemaDetail(err))
		}
	}

	return out
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}

	return fmt.Sprintf("%T", v)
}

func isType(v any, t string) bool {
	if t == "integer" {
		f, ok := v.(float64)
		return ok && f == float64(int64(f))
	}

	return typeName(v) == t
}

func length(v any) (int, bool) {
	switch x := v.(type) {
	case string:
		return utf8.RuneCountInString(x), true
	case []any:
		return len(x), true
	case map[string]any:
		return len(x), true
	}

	return 0, false
}

func contains(set []any, v any) bool {
	for _, e := range set {
		if reflect.DeepEqual(e, v) {
			return true
		}
	}

	return false
}

func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(b)
}

// schemaDetail summarises a schema validation error as the list of the
// (location, problem) pairs found, without the schema's location.
func schemaDetail(err error) string {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return oneLine(err.Error())
	}

	var parts []string

	for _, u := range ve.BasicOutput().Errors {
		if u.Error == nil {
			continue
		}

		loc := u.InstanceLocation
		if loc == "" {
			loc = "/"
		}

		parts = append(parts, loc+": "+oneLine(u.Error.String()))
	}

	if len(parts) == 0 {
		return "invalid"
	}

	return strings.Join(parts, "; ")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
