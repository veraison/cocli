# Linting `cocli` templates

Use `cocli lint` to check a JSON template (CoMID, CoRIM, CoEV, CoTS) against a
set of profile-specific static rules *before* you turn it into CBOR.
The rules are in a file that you control and supply when invoking the linter
on the template(s).  For example:

```sh
cocli lint --rules data/lint/psa.yaml --file my-comid.json
cocli lint --rules data/lint/cca-platform.yaml --dir templates/
```

Every "finding" is printed with the rule that raised it and the exact path of
the offending node.  The exit status is non-zero if at least one `error` was
found; `warn` and `info` are printed but do not fail the run.

In this directory you can find rulesets for the specs below.
Use them as they are, or as examples to copy from.

| File | Encodes |
|------|---------|
| [`psa.yaml`](./psa.yaml) | [draft-fdb-rats-psa-endorsements-10](https://www.ietf.org/archive/id/draft-fdb-rats-psa-endorsements-10.html) |
| [`cca-platform.yaml`](./cca-platform.yaml) | [§3.1 of draft-ydb-rats-cca-endorsements-04](https://www.ietf.org/archive/id/draft-ydb-rats-cca-endorsements-04.html#section-3.1) |
| [`cca-realm.yaml`](./cca-realm.yaml) | [§3.2 of draft-ydb-rats-cca-endorsements-04](https://www.ietf.org/archive/id/draft-ydb-rats-cca-endorsements-04.html#section-3.2) |

## Rules

The rules file is YAML.

Each rule specifies a *where* (`given`), i.e., which node in the JSON
tree to look at, and a *what* (`then`), i.e., which condition(s) must hold:

```yaml
rules:
  - id: no-debug                       # unique name, shown in the output
    severity: error                    # error (default) | warn | info
    message: debug builds not allowed  # optional, shown in the output
    given: $..['op-flags'][*]          # WHERE: which nodes to look at
    then:                              # WHAT: which condition must hold
      not-enum: [debug, notSecure]
```


## Nodes selection via JSONPath

`given` is an [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535) JSONPath.
Everything starts at `$`, which is the root of the template.

| Write | Selects |
|--------------------------------|-------------------------------------------------|
| `$.lang` | the top-level `lang` member |
| `$.triples` | a member, going down one level |
| `$.entities[*]` | every element of an array |
| `$.entities[0]` | the first element |
| `$..digests` | `digests`, wherever it is in the document |
| `$.ms[?@.n == 3]` | array elements for which the filter is true |
| `$.ms[?match(@.k, 'a[.].*')]` | array elements whose `k` fully matches a regex |

Important stuff:

1. **Keys with a hyphen need brackets.**  `$.triples.reference-values` is a
   syntax error; write `$.triples['reference-values']` instead.
   Almost every CoRIM key has a hyphen (`reference-values`, `raw-value`,
   `authorized-by`, ...), so...
2. **`match()` must match the whole string; `search()` matches only a part.**
   `match(@.key.value, 'cca[.]rotpk')` does *not* match `cca.rotpk.CM.0.0`;
   `match(@.key.value, 'cca[.]rotpk[.].*')` does. Write a literal dot as `[.]`
   to avoid troubles with YAML escaping.

If a filter mentions a member that is not in a node, those nodes are skipped.

## Asserting (i.e., `then`) vocabulary

`then` can encode multiple assertions in one go.
The assertions in one `then` must all hold.

**On the set of selected nodes**

| Assertion | Meaning |
|-----------|----------------------------------------------|
| `count` | how many nodes `given` selected: `{min, max}` |

**On each selected node**

| Assertion | Meaning |
|--|--|
| `equals` | the node equals this value |
| `enum` | the node equals one of these values |
| `not-enum` | the node equals none of these values |
| `pattern` | string matches this regex (not anchored; use `^...$`) |
| `not-pattern` | string does *not* match this regex |
| `length`| `{min, max}` on string length, array size or object size |
| `type` | `string`, `number`, `integer`, `boolean`, `object`, `array`, `null` |
| `schema` | the node validates against this inline JSON Schema |

Use `count` to assert *presence*.

## Recipes

**This SHOULD be present** (warn if the `lang` member is missing):

```yaml
- id: has-lang
  severity: warn
  given: $.lang
  then:
    count: { min: 1 }
```

**This MUST NOT be present:**

```yaml
- id: no-authorized-by
  given: $.triples['reference-values'][*].measurements[*]['authorized-by']
  then:
    count: { max: 0 }
```

**This MUST / MUST NOT be one of:**

```yaml
- id: roles
  given: $.entities[*].roles[*]
  then:
    enum: [tagCreator, creator, maintainer]
```

**This string MUST look like this:**

```yaml
- id: key-names
  given: $.triples['reference-values'][*].measurements[*].key.value
  then:
    pattern: '^cca\.'
```

**NOTE:** Use single quotes in YAML so the backslash is visible to the regex.

**This array MUST have between 1 and 3 elements:**

```yaml
- id: rotpk-keys
  given: $.triples['reference-values'][*].measurements[*].value.cryptokeys
  then:
    length: { min: 1, max: 3 }
```

**Only some measurements are affected** (select using a filter):

```yaml
- id: rotpk-has-one-key
  given: $.triples['reference-values'][*].measurements[?match(@.key.value, 'cca[.]rotpk[.].*')].value.cryptokeys
  then:
    length: { min: 1, max: 1 }
```

**Every node MUST have these fields** (mandatory fields are handled via `schema`
since `pattern`, `length` & friends only look at a member that is there):

```yaml
- id: swcomp-mandatory-fields
  given: $.triples['reference-values'][*].measurements[?@.key.value == 'cca.software-component']
  then:
    schema:
      type: object
      required: [value]
      properties:
        value:
          type: object
          required: [digests, cryptokeys]
```

**At most one of something *per triple*.**  `count` looks at the whole
document, so to scope a count to each triple, select the array and use the
schema keywords `contains` / `maxContains`:

```yaml
- id: one-platform-config-per-triple
  given: $.triples['reference-values'][*].measurements
  then:
    schema:
      contains:
        type: object
        required: [key]
        properties:
          key: { properties: { value: { const: cca.platform-config } } }
      minContains: 0
      maxContains: 1
```

## Output Exegesis

The pattern is `file: severity: rule-id: path: message (detail)`:

```console
"t.json": error: no-debug: $['triples']['reference-values'][0]['measurements'][0]['value']['op-flags'][1] ("debug" is forbidden)
"t.json": warn: has-lang: $.lang: set a language (found 0, want at least 1)
```

## Writing a new rule

1. Start with `given` and `then: { count: { min: 1 } }`, and run it on a
   template you know to be good.  If it complains, your path is selecting
   nothing.  Check things like hyphens and the filter.
2. Swap in the real assertion.
3. Run it against a template you know to be good (i.e., one which produces no
   output) and then against a copy that breaks that requirement (it must fire
   only that rule).
4. Put the spec section in `description:` so that future exegetes know why the
   rule exists :-)

## (current) Engine limitations

- Rules that compute something -- for example, comparing the decoded length of a
  `value` and its `mask`, or checking that the digests of a component have
  distinct algorithms -- are not supported
- A rule cannot see the CoRIM profile while linting a CoMID since they are
  separate files.  You need to pick the rule file that matches the profile.
