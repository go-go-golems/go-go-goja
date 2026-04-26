---
Title: Imported event-emitter source brief
Ticket: EVT-001
Status: reference
Topics:
    - goja
    - javascript
    - event-emitter
    - module
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/event-emitter.md
Summary: Imported source brief provided by the user for the event-emitter design.
LastUpdated: 2026-04-26T09:33:00-04:00
WhatFor: Preserve the user-provided implementation sketch and references for EVT-001.
WhenToUse: Use as source material alongside the design guide.
---

Implement it as a **Go-owned event bus with a JS EventEmitter façade**. The key rule: **never call `goja.Runtime` directly from Watermill/filewatcher goroutines**. `goja.Runtime` is not goroutine-safe, so every JS call should be scheduled onto the goja event loop with `RunOnLoop()`. `goja_nodejs/eventloop` is built for this: `RunOnLoop()` preserves order and is safe to call from inside or outside the loop. ([GitHub][1])

Node’s `EventEmitter.emit()` calls listeners synchronously and ignores return values, so treat it as a dispatch mechanism, not as a reliable async job API. For Watermill, pass an object with explicit `ack()` / `nack()` methods. Watermill subscribers require `Ack()` to receive the next message, or `Nack()` for redelivery. ([Node.js][2])

## 1. Minimal `events` module for goja

```go
package jsevents

import (
	"errors"
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
)

const eventEmitterSource = `
(function () {
  function EventEmitter() {
    this._events = Object.create(null);
  }

  EventEmitter.prototype.on =
  EventEmitter.prototype.addListener = function (name, fn) {
    if (typeof fn !== "function") throw new TypeError("listener must be a function");
    this.emit("newListener", name, fn);
    var list = this._events[name];
    if (!list) list = this._events[name] = [];
    list.push(fn);
    return this;
  };

  EventEmitter.prototype.once = function (name, fn) {
    var self = this;
    function wrapped() {
      self.removeListener(name, wrapped);
      return fn.apply(this, arguments);
    }
    wrapped.listener = fn;
    return this.on(name, wrapped);
  };

  EventEmitter.prototype.removeListener =
  EventEmitter.prototype.off = function (name, fn) {
    var list = this._events[name];
    if (!list) return this;

    for (var i = 0; i < list.length; i++) {
      if (list[i] === fn || list[i].listener === fn) {
        list.splice(i, 1);
        this.emit("removeListener", name, fn);
        break;
      }
    }

    if (list.length === 0) delete this._events[name];
    return this;
  };

  EventEmitter.prototype.removeAllListeners = function (name) {
    if (name === undefined) this._events = Object.create(null);
    else delete this._events[name];
    return this;
  };

  EventEmitter.prototype.listeners = function (name) {
    var list = this._events[name];
    return list ? list.slice() : [];
  };

  EventEmitter.prototype.listenerCount = function (name) {
    var list = this._events[name];
    return list ? list.length : 0;
  };

  EventEmitter.prototype.emit = function (name) {
    var list = this._events[name];

    if (!list || list.length === 0) {
      if (name === "error") {
        var err = arguments[1];
        throw err instanceof Error ? err : new Error(err ? String(err) : "Unhandled error event");
      }
      return false;
    }

    var args = Array.prototype.slice.call(arguments, 1);
    list = list.slice();

    for (var i = 0; i < list.length; i++) {
      list[i].apply(this, args);
    }

    return true;
  };

  EventEmitter.EventEmitter = EventEmitter;
  return EventEmitter;
})()
`

func eventsLoader(vm *goja.Runtime, module *goja.Object) {
	v, err := vm.RunString(eventEmitterSource)
	if err != nil {
		panic(err)
	}

	// Supports both:
	//   const EventEmitter = require("events")
	//   const { EventEmitter } = require("events")
	v.ToObject(vm).Set("EventEmitter", v)

	module.Set("exports", v)
}

type Engine struct {
	loop    *eventloop.EventLoop
	onError func(error)
}

func NewEngine(onError func(error)) (*Engine, error) {
	registry := new(require.Registry)

	registry.RegisterNativeModule("events", eventsLoader)
	registry.RegisterNativeModule("node:events", eventsLoader)

	loop := eventloop.NewEventLoop(eventloop.WithRegistry(registry))

	e := &Engine{
		loop:    loop,
		onError: onError,
	}

	loop.Start()

	err := e.Eval(`
		globalThis.goEvents = new (require("events"))();
	`)
	if err != nil {
		loop.Terminate()
		return nil, err
	}

	return e, nil
}

func (e *Engine) Close() {
	e.loop.Terminate()
}

func (e *Engine) Eval(src string) error {
	done := make(chan error, 1)

	ok := e.loop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunString(src)
		done <- err
	})

	if !ok {
		return errors.New("goja event loop is terminated")
	}

	return <-done
}

func (e *Engine) Emit(name string, args ...any) bool {
	return e.loop.RunOnLoop(func(vm *goja.Runtime) {
		jsArgs := make([]goja.Value, 0, len(args))

		for _, arg := range args {
			jsArgs = append(jsArgs, vm.ToValue(arg))
		}

		_, err := e.emitOnLoop(vm, name, jsArgs...)
		if err != nil && e.onError != nil {
			e.onError(err)
		}
	})
}

func (e *Engine) emitOnLoop(vm *goja.Runtime, name string, args ...goja.Value) (bool, error) {
	emitter := vm.Get("goEvents").ToObject(vm)

	emit, ok := goja.AssertFunction(emitter.Get("emit"))
	if !ok {
		return false, errors.New("goEvents.emit is not a function")
	}

	argv := make([]goja.Value, 0, len(args)+1)
	argv = append(argv, vm.ToValue(name))
	argv = append(argv, args...)

	ret, err := emit(emitter, argv...)
	if err != nil {
		return false, err
	}

	return ret.ToBoolean(), nil
}

func (e *Engine) fail(err error) {
	if err != nil && e.onError != nil {
		e.onError(err)
	}
}

func formatJSError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
```

JS usage:

```js
goEvents.on("fs:event", (ev) => {
  console.log("file event", ev.name, ev.op);
});

goEvents.on("watermill:orders", (msg) => {
  try {
    const order = JSON.parse(msg.payload);

    console.log("order", order);

    msg.ack();
  } catch (e) {
    console.log("order failed", e);
    msg.nack();
  }
});

goEvents.on("error", (err) => {
  console.log("go-side error", err.message || err);
});
```

## 2. Watermill adapter

Watermill gives you a channel from `Subscribe()`. The adapter should read that channel in Go, schedule delivery onto the JS loop, and expose `ack()` / `nack()` to JS. Do not auto-ack unless “delivered to JS” is good enough for your use case. ([Watermill][3])

```go
package jsevents

import (
	"context"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
)

func (e *Engine) SubscribeWatermill(
	ctx context.Context,
	sub message.Subscriber,
	topic string,
	eventName string,
) error {
	messages, err := sub.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-messages:
				if !ok {
					return
				}

				e.dispatchWatermillMessage(msg, eventName)
			}
		}
	}()

	return nil
}

func (e *Engine) dispatchWatermillMessage(msg *message.Message, eventName string) {
	ok := e.loop.RunOnLoop(func(vm *goja.Runtime) {
		jsMsg := vm.NewObject()

		metadata := map[string]string{}
		for k, v := range msg.Metadata {
			metadata[k] = v
		}

		var settleOnce sync.Once

		jsMsg.Set("uuid", msg.UUID)
		jsMsg.Set("payload", string(msg.Payload))
		jsMsg.Set("metadata", metadata)

		jsMsg.Set("ack", func() bool {
			called := false
			settleOnce.Do(func() {
				called = msg.Ack()
			})
			return called
		})

		jsMsg.Set("nack", func() bool {
			called := false
			settleOnce.Do(func() {
				called = msg.Nack()
			})
			return called
		})

		delivered, err := e.emitOnLoop(vm, eventName, jsMsg)
		if err != nil || !delivered {
			settleOnce.Do(func() {
				msg.Nack()
			})
			e.fail(err)
		}
	})

	if !ok {
		msg.Nack()
	}
}
```

Use it like:

```go
err := engine.SubscribeWatermill(ctx, redisSubscriber, "orders", "watermill:orders")
if err != nil {
	panic(err)
}
```

Important edge case: if a JS listener receives a Watermill message and never calls `ack()` or `nack()`, the subscriber can stall. That is usually desirable during development because it exposes broken handlers quickly.

## 3. File watcher adapter

`fsnotify.Watcher` exposes `Events` and `Errors` channels, so it maps cleanly into the same emitter. It does not recursively watch subdirectories by default, so add watches for each directory you care about. ([Go Packages][4])

```go
package jsevents

import (
	"context"

	"github.com/fsnotify/fsnotify"
)

func (e *Engine) WatchPath(ctx context.Context, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		_ = watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return

			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}

				e.Emit("fs:event", map[string]any{
					"name": ev.Name,
					"op":   ev.Op.String(),
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				e.Emit("error", map[string]any{
					"source":  "fsnotify",
					"message": err.Error(),
				})
			}
		}
	}()

	return nil
}
```

## 4. What about streams?

Start with `EventEmitter`. Add streams only when you need **backpressure**, **chunking**, or `pipe()`-style composition. Node streams are more than events: readable streams buffer data until consumed, writable streams signal pressure by returning `false` from `write()`, and `pipe()` exists partly to keep buffering bounded. ([Node.js][5])

A practical progression:

1. **EventEmitter** for file events, Watermill messages, Redis messages, cron ticks, process lifecycle.
2. **Readable-like wrapper** for high-volume event feeds.
3. **Full Node-compatible streams** only if you need npm packages that expect real `stream.Readable` / `stream.Writable`.

A minimal readable-ish JS class can be built on top of your `EventEmitter`:

```js
const EventEmitter = require("events");

class GoReadable extends EventEmitter {
  constructor() {
    super();
    this.paused = false;
    this.queue = [];
    this.closed = false;
  }

  push(chunk) {
    if (this.closed) return false;

    if (chunk === null) {
      this.closed = true;
      this.emit("end");
      this.emit("close");
      return false;
    }

    if (this.paused) {
      this.queue.push(chunk);
      return false;
    }

    this.emit("data", chunk);
    return true;
  }

  pause() {
    this.paused = true;
    return this;
  }

  resume() {
    this.paused = false;

    while (!this.paused && this.queue.length > 0) {
      this.emit("data", this.queue.shift());
    }

    return this;
  }

  destroy(err) {
    this.closed = true;
    if (err) this.emit("error", err);
    this.emit("close");
  }
}
```

For Watermill specifically, I would **not** model messages as a stream first. The ack/nack lifecycle is message-oriented, not byte-stream-oriented. Keep Watermill as:

```js
goEvents.on("watermill:topic", (msg) => {
  // process
  msg.ack();
});
```

Then add stream wrappers later for sources where pressure matters, such as logs, subprocess output, socket-like feeds, or large file reads.

[1]: https://github.com/dop251/goja "GitHub - dop251/goja: ECMAScript/JavaScript engine in pure Go · GitHub"
[2]: https://nodejs.org/api/events.html "Events | Node.js v25.9.0 Documentation"
[3]: https://watermill.io/docs/pub-sub/?utm_source=chatgpt.com "Publisher & Subscriber | Watermill | Event-Driven in Go"
[4]: https://pkg.go.dev/github.com/fsnotify/fsnotify "fsnotify package - github.com/fsnotify/fsnotify - Go Packages"
[5]: https://nodejs.org/api/stream.html "Stream | Node.js v25.9.0 Documentation"

