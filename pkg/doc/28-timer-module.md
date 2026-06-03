---
Title: Timer Module
Slug: timer-module
Short: Promise-based sleep and delay helpers
Topics:
- timer
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

The `timer` module exposes a single Promise-based helper: `sleep`. It lets JavaScript code pause asynchronously without blocking the runtime owner goroutine.

The module requires a runtime with owner services because the backing goroutine dispatches resolution back onto the goja runtime through the owner event loop.

## JavaScript usage

```javascript
const timer = require("timer");

async function delayedHello() {
  await timer.sleep(500);
  console.log("hello after 500 ms");
}

delayedHello();
```

## Module API

### `sleep(ms)`

Returns a Promise that resolves after `ms` milliseconds. The timer is canceled if either the owner call context or the runtime lifetime context ends before the delay completes.

If `ms` is negative, the Promise rejects with an error string.

## Design notes

`sleep` is a thin wrapper around Go's `time.Timer` that sends its resolve call through the runtime owner's dispatch queue. This means the JavaScript Promise resolves on the correct goroutine and does not race with concurrent runtime access.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| "timer module requires runtime services" panic | The runtime was built without owner-based services | Use a factory created with `engine.NewBuilder().Build()` rather than a bare `goja.New()` |
| Promise never resolves after a long delay | The owner call context was canceled or timed out | Increase the context deadline on the owner call, or move the sleep to a longer-lived context |
| Negative duration rejected | `ms < 0` | Pass a non-negative integer for the sleep duration |
