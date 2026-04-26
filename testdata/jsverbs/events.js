const EventEmitter = require("events");

function eventTimeline(prefix, count) {
  const emitter = new EventEmitter();
  const rows = [];

  emitter.once("tick", (index) => {
    rows.push({ kind: "once", value: `${prefix}:${index}` });
  });

  emitter.on("tick", (index) => {
    rows.push({ kind: "tick", value: `${prefix}:${index}` });
  });

  for (let i = 0; i < count; i++) {
    emitter.emit("tick", i);
  }

  return rows;
}

__verb__("eventTimeline", {
  short: "Demonstrate EventEmitter listener ordering and once semantics",
  fields: {
    prefix: {
      argument: true,
      help: "Prefix for emitted values"
    },
    count: {
      type: "int",
      default: 3,
      help: "Number of tick events to emit"
    }
  }
});

function listenerSummary(name) {
  const emitter = new EventEmitter();
  function persistent() {}
  function oneShot() {}

  emitter.on("message", persistent);
  emitter.once("message", oneShot);

  const before = emitter.listenerCount("message");
  const names = emitter.eventNames();
  emitter.removeListener("message", oneShot);
  const afterRemoveOnce = emitter.listenerCount("message");
  emitter.removeAllListeners("message");

  return {
    name,
    before,
    afterRemoveOnce,
    afterRemoveAll: emitter.listenerCount("message"),
    eventNames: names.join(",")
  };
}

__verb__("listenerSummary", {
  short: "Inspect EventEmitter listener counts from jsverbs",
  fields: {
    name: {
      argument: true,
      help: "Label included in the returned summary"
    }
  }
});

function handledError(message) {
  const emitter = new EventEmitter();
  let handled = "";

  emitter.on("error", (err) => {
    handled = err && err.message ? err.message : String(err);
  });

  const delivered = emitter.emit("error", new Error(message));

  return {
    delivered,
    handled
  };
}

__verb__("handledError", {
  short: "Show that EventEmitter error events can be handled",
  fields: {
    message: {
      argument: true,
      help: "Error message to emit"
    }
  }
});
