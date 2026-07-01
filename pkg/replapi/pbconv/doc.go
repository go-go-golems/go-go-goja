// Package pbconv converts between internal replapi/replsession DTOs and the
// public protobuf transport messages in goja.replapi.v1.
//
// The conversion boundary keeps the REPL execution service independent from
// generated transport code while still allowing HTTP/front-end callers to use a
// schema-first protobuf JSON contract.
package pbconv
