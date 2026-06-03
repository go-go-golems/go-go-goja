---
Title: Events Module
Slug: events-module
Short: Node-style EventEmitter backed by Go-native dispatch
Topics:
- events
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `events` module exports a Go-native `EventEmitter` constructor that mirrors a subset of the Node.js `events.EventEmitter` API. It is aliased as both `events` and `node:events`.

This implementation is not goroutine-safe outside the owning goja runtime goroutine. All listener registration, emission, and removal happen synchronously on that goroutine through the runtime owner.

## JavaScript usage

```javascript
const EventEmitter = require("events");

const emitter = new EventEmitter();

emitter.on("data", (chunk) => {
  console.log("received:", chunk);
});

emitter.once("end", () => {
  console.log("stream ended");
});

emitter.emit("data", "hello");
emitter.emit("end");
```

## Constructor

### `new EventEmitter()`

Creates a new emitter instance backed by a Go-native dispatch table.

## Instance methods

### `on(name, listener)` / `addListener(name, listener)`

Registers a listener function for `name`. Both methods are aliases.

### `once(name, listener)`

Registers a listener that fires exactly once and is automatically removed before invocation.

### `off(name, listener)` / `removeListener(name, listener)`

Removes the first listener that strictly matches the provided function reference.

### `removeAllListeners(name?)`

Without arguments, removes every listener on the emitter. With `name`, removes only listeners for that event.

### `emit(name, ...args)`

Invokes all listeners for `name` in registration order. Returns `true` if at least one listener was called, `false` otherwise.

Emitting the `"error"` event with no listeners causes a thrown error rather than a silent return.

### `listeners(name)`

Returns an array of listener functions. For `once` listeners, returns the wrapper rather than the original function unless the emitter unwraps them internally.

### `rawListeners(name)`

Returns the raw listener array including wrapper objects for `once` registrations.

### `listenerCount(name)`

Returns the number of registered listeners for `name`.

### `eventNames()`

Returns an array of string event names that currently have one or more listener. Symbol names are omitted from this list.

## Properties

### `EventEmitter.default`

Points to the constructor itself, matching Node.js conventions.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Emitter does not fire callbacks as expected | The call is being made from a goroutine other than the runtime owner | Use `runtimeServices.PostWithCustomContext` or equivalent to dispatch back onto the owner |
| "listener must be a function" error | A non-function value was passed to `on` / `once` / `addListener` | Pass a function reference |
| "unhandled error event" thrown | `emit("error", ...)` with no error listeners | Attach an `"error"` handler or wrap calls in a guard |
