// yaml.js – example script demonstrating the yaml native module
//
// Expected console output ends with the line "OK".
// The script exercises:
//   1. yaml.parse      – parse a YAML string into a JS value
//   2. yaml.stringify  – serialize a JS value into YAML
//   3. yaml.validate   – check YAML syntax without parsing

const yaml = require("yaml");

// --- Test 1: Parse a simple YAML document ---
const configYaml = `
name: go-go-goja
version: 1.0
features:
  - repl
  - modules
  - plugins
metadata:
  author: go-go-golems
`;

const config = yaml.parse(configYaml);
if (config.name !== "go-go-goja") {
  throw new Error("yaml.parse: name mismatch");
}
if (config.version !== 1.0) {
  throw new Error("yaml.parse: version mismatch");
}
if (!Array.isArray(config.features) || config.features.length !== 3) {
  throw new Error("yaml.parse: features array mismatch");
}
if (config.metadata.author !== "go-go-golems") {
  throw new Error("yaml.parse: nested metadata mismatch");
}
console.log("parse: OK");

// --- Test 2: Stringify a JavaScript object ---
const manifest = {
  service: "api-gateway",
  port: 8080,
  tls: true,
  routes: [
    { path: "/health", method: "GET" },
    { path: "/api/v1/users", method: "POST" }
  ]
};

const manifestYaml = yaml.stringify(manifest);
if (!manifestYaml.includes("service: api-gateway")) {
  throw new Error("yaml.stringify: missing service field");
}
if (!manifestYaml.includes("port: 8080")) {
  throw new Error("yaml.stringify: missing port field");
}
console.log("stringify: OK");

// --- Test 3: Custom indent option ---
const nested = { a: { b: { c: 1 } } };
const indent4 = yaml.stringify(nested, { indent: 4 });
if (!indent4.includes("        c: 1")) {
  throw new Error("yaml.stringify: indent option not applied");
}
console.log("stringify with indent: OK");

// --- Test 4: Validate correct YAML ---
const validResult = yaml.validate("hello: world");
if (!validResult.valid) {
  throw new Error("yaml.validate: expected valid for correct YAML");
}
console.log("validate valid: OK");

// --- Test 5: Validate broken YAML ---
const invalidResult = yaml.validate("[bad");
if (invalidResult.valid) {
  throw new Error("yaml.validate: expected invalid for broken YAML");
}
if (!invalidResult.errors || invalidResult.errors.length === 0) {
  throw new Error("yaml.validate: expected errors for broken YAML");
}
console.log("validate invalid: OK");

// --- Test 6: Round-trip ---
const original = { items: [1, 2, 3], active: true, label: "test" };
const serialized = yaml.stringify(original);
const roundTripped = yaml.parse(serialized);
if (roundTripped.active !== true) {
  throw new Error("yaml round-trip: boolean mismatch");
}
if (roundTripped.label !== "test") {
  throw new Error("yaml round-trip: string mismatch");
}
if (roundTripped.items.length !== 3) {
  throw new Error("yaml round-trip: array length mismatch");
}
console.log("round-trip: OK");

console.log("OK");
