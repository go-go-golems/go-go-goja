---
Title: YAML Module
Slug: yaml-module
Short: Parse, stringify, and validate YAML from JavaScript
Topics:
- yaml
- modules
- primitives
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `yaml` module provides native YAML parsing and serialization for go-go-goja runtimes. It is enabled by default when you use `engine.DefaultRegistryModules()`.

## Loading the module

```javascript
const yaml = require("yaml");
```

## API

### `yaml.parse(input)`

Parse a YAML string into a JavaScript value. Throws on parse errors.

**Parameters:**
- `input` (string): The YAML document to parse.

**Returns:** `any` — The parsed value (object, array, string, number, boolean, or null).

**Example:**
```javascript
const doc = yaml.parse("name: goja\nversion: 1.0");
console.log(doc.name);    // "goja"
console.log(doc.version); // 1.0
```

**Multi-document behavior:** If the input contains multiple YAML documents separated by `---`, only the first document is returned.

### `yaml.stringify(value, options?)`

Serialize a JavaScript value into a YAML string.

**Parameters:**
- `value` (any): The value to serialize.
- `options` (object, optional):
  - `indent` (number): Number of spaces per indentation level. Default is `2`.

**Returns:** `string` — The YAML representation.

**Example:**
```javascript
const out = yaml.stringify({ name: "test", items: [1, 2] });
console.log(out);
// name: test
// items:
//   - 1
//   - 2
```

**Custom indentation:**
```javascript
const out = yaml.stringify({ a: { b: 1 } }, { indent: 4 });
// a:
//     b: 1
```

### `yaml.validate(input)`

Validate YAML syntax without producing a parsed value.

**Parameters:**
- `input` (string): The YAML document to validate.

**Returns:** `object` — An object with:
- `valid` (boolean): `true` if the YAML is well-formed.
- `errors` (string[], optional): Array of error messages when `valid` is `false`.

**Example:**
```javascript
const result = yaml.validate("hello: world");
if (result.valid) {
    console.log("YAML is valid");
} else {
    console.error("YAML errors:", result.errors.join("; "));
}
```

## Type mappings

YAML types map to JavaScript types as follows:

| YAML tag | JavaScript type |
|---|---|
| `!!str` | `string` |
| `!!int` | `number` |
| `!!float` | `number` |
| `!!bool` | `boolean` |
| `!!null` | `null` |
| `!!map` | `object` |
| `!!seq` | `array` |

## Error handling

`parse` and `stringify` throw JavaScript `Error` objects on failure. `validate` never throws; it returns a result object.

```javascript
try {
    yaml.parse("[bad");
} catch (e) {
    console.error(e.message); // "yaml.parse: ..."
}

try {
    yaml.stringify({}, { foo: 1 }); // unknown option
} catch (e) {
    console.error(e.message); // "yaml.stringify: unknown option \"foo\""
}
```

## Complete example

```javascript
const yaml = require("yaml");

// Parse configuration
const config = yaml.parse(`
server:
  host: localhost
  port: 8080
  tls: true
`);

// Modify
config.server.port = 9090;

// Serialize back
const updated = yaml.stringify(config, { indent: 2 });
console.log(updated);

// Validate user input
const userInput = `{name: "test"}`;
const check = yaml.validate(userInput);
if (!check.valid) {
    console.error("Invalid YAML:", check.errors);
}
```

## See also

- `repl-usage` — REPL module usage examples
- `creating-modules` — Guide to creating native modules
