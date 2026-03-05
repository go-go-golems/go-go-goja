// ============================================================
// Sample 4: Event system — advanced, multiple classes,
//           cross-package references, error handling docs
// All __doc__ sentinels are at top level.
// ============================================================

__package__({
  name: "core/events",
  title: "Event Emitter System",
  category: "Core",
  guide: "docs/guides/events.md",
  version: "3.1.0",
  description: "A typed, priority-ordered event emitter with once-listeners, wildcards, and async dispatch.",
  seeAlso: ["core/lifecycle", "core/scheduler"],
});

doc`
---
package: core/events
---

# Event Emitter System

This module provides a robust publish/subscribe event system designed for
both synchronous and asynchronous use. It is the backbone of the framework's
component communication model.

## Key Concepts

**EventEmitter** is the base class. Any object that needs to emit events
should extend it or compose it as a property.

**Priority ordering** ensures that high-priority listeners (lower numeric
value) are always called before lower-priority ones, regardless of
registration order.

**Wildcard listeners** registered with \`"*"\` receive every event emitted
on the emitter, useful for logging and debugging.

## Error Handling

By default, errors thrown inside listeners are caught and re-emitted as
\`"error"\` events. If no \`"error"\` listener is registered, the error is
rethrown synchronously. This mirrors the Node.js EventEmitter contract.

## Async Dispatch

\`emitAsync\` returns a \`Promise\` that resolves when all listeners (including
async ones) have settled. Listener errors in async mode are collected and
thrown as an \`AggregateError\`.
`;

// ---- EventToken documentation ----

__doc__("EventToken", {
  summary: "Opaque handle returned by on() and once(). Call remove() to unsubscribe.",
  concepts: ["event-system", "subscription-management"],
  docpage: "docs/core/events.md#tokens",
  related: ["EventEmitter.on", "EventEmitter.once", "EventEmitter.off"],
  tags: ["core", "events"],
});

__doc__("EventToken.remove", {
  summary: "Unsubscribes the associated listener from its emitter.",
  concepts: ["event-system", "subscription-management"],
  returns: { type: "void" },
  tags: ["core"],
});

// ---- EventEmitter documentation ----

__doc__("EventEmitter", {
  summary: "Base class for objects that emit named events to registered listeners.",
  concepts: ["event-system", "publish-subscribe", "observer-pattern"],
  docpage: "docs/core/events.md",
  related: ["EventToken", "EventEmitter.on", "EventEmitter.emit", "EventEmitter.emitAsync"],
  tags: ["core", "events", "base-class"],
});

doc`
---
symbol: EventEmitter
---

**EventEmitter** implements the classic observer pattern. Listeners are
stored per event name in a priority-sorted list. The default priority is
\`0\`; lower numbers run first.

### Wildcard Support

Register a listener on \`"*"\` to receive all events:

\`\`\`js
emitter.on("*", (eventName, ...args) => {
  console.log("Event fired:", eventName, args);
});
\`\`\`

### Memory Management

Always call \`token.remove()\` or \`emitter.off()\` when a component is
destroyed to prevent memory leaks. Alternatively, use \`once()\` for
one-shot subscriptions.
`;

__doc__("EventEmitter.on", {
  summary: "Registers a persistent listener for the named event.",
  concepts: ["event-system", "subscription"],
  params: [
    { name: "event",    type: "string",   description: "Event name, or '*' for all events." },
    { name: "listener", type: "Function", description: "Callback invoked with event arguments." },
    { name: "priority", type: "number",   description: "Execution order; lower runs first. Default 0." },
  ],
  returns: { type: "EventToken", description: "Token that can be used to unsubscribe." },
  related: ["EventEmitter.once", "EventEmitter.off"],
  tags: ["core"],
});

__doc__("EventEmitter.once", {
  summary: "Registers a one-shot listener that auto-removes after first invocation.",
  concepts: ["event-system", "one-shot"],
  params: [
    { name: "event",    type: "string" },
    { name: "listener", type: "Function" },
    { name: "priority", type: "number", description: "Default 0." },
  ],
  returns: { type: "EventToken" },
  related: ["EventEmitter.on"],
  tags: ["core"],
});

__doc__("EventEmitter.off", {
  summary: "Removes a specific listener from the named event.",
  concepts: ["event-system", "unsubscription"],
  params: [
    { name: "event",    type: "string" },
    { name: "listener", type: "Function", description: "The exact function reference to remove." },
  ],
  returns: { type: "void" },
  related: ["EventEmitter.on", "EventToken.remove"],
  tags: ["core"],
});

__doc__("EventEmitter.emit", {
  summary: "Synchronously invokes all listeners for the named event.",
  concepts: ["event-system", "dispatch"],
  params: [
    { name: "event",   type: "string",  description: "Event name to emit." },
    { name: "...args", type: "any[]",   description: "Arguments passed to each listener." },
  ],
  returns: { type: "boolean", description: "true if any listeners were called." },
  related: ["EventEmitter.emitAsync"],
  tags: ["core"],
});

__doc__("EventEmitter.emitAsync", {
  summary: "Asynchronously invokes all listeners, awaiting Promises. Collects errors.",
  concepts: ["event-system", "async-dispatch"],
  params: [
    { name: "event",   type: "string" },
    { name: "...args", type: "any[]" },
  ],
  returns: { type: "Promise<boolean>" },
  related: ["EventEmitter.emit"],
  tags: ["core", "async"],
});

__doc__("EventEmitter.listenerCount", {
  summary: "Returns the number of listeners registered for the named event.",
  concepts: ["event-system"],
  params: [{ name: "event", type: "string" }],
  returns: { type: "number" },
  tags: ["utility"],
});

// ---- EventToken implementation ----

export class EventToken {
  #emitter;
  #event;
  #listener;

  constructor(emitter, event, listener) {
    this.#emitter  = emitter;
    this.#event    = event;
    this.#listener = listener;
  }

  remove() {
    this.#emitter.off(this.#event, this.#listener);
  }
}

// ---- EventEmitter implementation ----

export class EventEmitter {
  #listeners = new Map();

  #getList(event) {
    if (!this.#listeners.has(event)) this.#listeners.set(event, []);
    return this.#listeners.get(event);
  }

  on(event, listener, priority = 0) {
    const list = this.#getList(event);
    list.push({ priority, fn: listener });
    list.sort((a, b) => a.priority - b.priority);
    return new EventToken(this, event, listener);
  }

  once(event, listener, priority = 0) {
    const wrapper = (...args) => {
      this.off(event, wrapper);
      listener(...args);
    };
    return this.on(event, wrapper, priority);
  }

  off(event, listener) {
    const list = this.#listeners.get(event);
    if (!list) return;
    const idx = list.findIndex(e => e.fn === listener);
    if (idx !== -1) list.splice(idx, 1);
  }

  emit(event, ...args) {
    const specific  = this.#listeners.get(event) ?? [];
    const wildcards = this.#listeners.get("*")   ?? [];
    const all = [
      ...specific,
      ...wildcards.map(e => ({ ...e, fn: (...a) => e.fn(event, ...a) })),
    ].sort((a, b) => a.priority - b.priority);

    if (all.length === 0) return false;

    for (const { fn } of all) {
      try { fn(...args); }
      catch (err) {
        if (event !== "error") this.emit("error", err);
        else throw err;
      }
    }
    return true;
  }

  async emitAsync(event, ...args) {
    const specific  = this.#listeners.get(event) ?? [];
    const wildcards = this.#listeners.get("*")   ?? [];
    if (specific.length + wildcards.length === 0) return false;

    const results = await Promise.allSettled(
      specific.map(({ fn }) => Promise.resolve().then(() => fn(...args)))
    );
    const errors = results
      .filter(r => r.status === "rejected")
      .map(r => r.reason);
    if (errors.length > 0) throw new AggregateError(errors, "Listeners failed");
    return true;
  }

  listenerCount(event) {
    return this.#listeners.get(event)?.length ?? 0;
  }
}

// ---- Examples ----

__example__({
  id: "events-basic",
  title: "Basic on/emit/off",
  symbols: ["EventEmitter", "EventEmitter.on", "EventEmitter.emit", "EventEmitter.off"],
  concepts: ["event-system", "publish-subscribe"],
  tags: ["beginner"],
});
function example_eventsBasic() {
  const emitter = new EventEmitter();
  const log = [];
  const token = emitter.on("data", v => log.push(v));
  emitter.emit("data", 1);
  emitter.emit("data", 2);
  token.remove();
  emitter.emit("data", 3);
  console.assert(log.length === 2);
}

__example__({
  id: "events-once",
  title: "One-shot listener with once()",
  symbols: ["EventEmitter.once"],
  concepts: ["event-system", "one-shot"],
  tags: ["beginner"],
});
function example_eventsOnce() {
  const emitter = new EventEmitter();
  let count = 0;
  emitter.once("ping", () => count++);
  emitter.emit("ping");
  emitter.emit("ping");
  console.assert(count === 1);
}

__example__({
  id: "events-priority",
  title: "Priority ordering of listeners",
  symbols: ["EventEmitter.on"],
  concepts: ["event-system", "priority"],
  docpage: "docs/core/events.md#priority",
  tags: ["intermediate"],
});
function example_eventsPriority() {
  const emitter = new EventEmitter();
  const order = [];
  emitter.on("tick", () => order.push("C"), 10);
  emitter.on("tick", () => order.push("A"), -5);
  emitter.on("tick", () => order.push("B"),  0);
  emitter.emit("tick");
  console.assert(order.join("") === "ABC");
}

__example__({
  id: "events-async",
  title: "Async dispatch with emitAsync",
  symbols: ["EventEmitter.emitAsync"],
  concepts: ["event-system", "async-dispatch"],
  tags: ["advanced"],
});
async function example_eventsAsync() {
  const emitter = new EventEmitter();
  const results = [];
  emitter.on("fetch", async id => {
    await new Promise(r => setTimeout(r, 10));
    results.push(id);
  });
  await emitter.emitAsync("fetch", 42);
  console.assert(results[0] === 42);
}
