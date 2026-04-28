const EventEmitter = require("events");
const fs = require("fs");
const path = require("path");
const timer = require("timer");

async function watchAndWrite(dir, fileName, recursive, debounceMs, include, exclude) {
  const emitter = new EventEmitter();
  const events = [];
  const errors = [];
  const options = {
    recursive: !!recursive,
    debounceMs: debounceMs || 0,
    include: include || [],
    exclude: exclude || []
  };

  emitter.on("event", (ev) => {
    if (String(ev.name).indexOf(fileName) >= 0 || String(ev.relativeName).indexOf(fileName) >= 0) {
      events.push({
        source: ev.source,
        watchPath: ev.watchPath,
        name: ev.name,
        relativeName: ev.relativeName,
        op: ev.op,
        create: ev.create,
        write: ev.write,
        remove: ev.remove,
        rename: ev.rename,
        chmod: ev.chmod,
        recursive: ev.recursive,
        debounced: ev.debounced,
        count: ev.count
      });
    }
  });
  emitter.on("error", (err) => {
    errors.push(err && err.message ? err.message : String(err));
  });

  const conn = fswatch.watch(dir, emitter, options);
  const file = path.join(dir, fileName);
  const parent = path.dirname(file);
  fs.mkdirSync(parent, { recursive: true });
  if (options.recursive && parent !== dir) {
    await timer.sleep(50);
  }
  fs.writeFileSync(file, "created by fswatch jsverb");
  if (options.debounceMs > 0) {
    fs.writeFileSync(file, "updated by fswatch jsverb");
  }

  for (let i = 0; i < 150 && events.length === 0 && errors.length === 0; i++) {
    await timer.sleep(10);
  }

  const closeResult = conn.close();

  if (errors.length > 0) {
    throw new Error(errors[0]);
  }
  if (events.length === 0) {
    throw new Error("no fswatch event received for " + fileName);
  }

  const first = events[0];
  return {
    count: events.length,
    source: first.source,
    watchPath: first.watchPath,
    name: first.name,
    relativeName: first.relativeName,
    op: first.op,
    create: first.create,
    write: first.write,
    recursive: first.recursive,
    debounced: first.debounced,
    rawCount: first.count,
    connectionRecursive: conn.recursive,
    connectionDebounceMs: conn.debounceMs,
    closeResult
  };
}

__verb__("watchAndWrite", {
  short: "Watch a directory with fswatch and write a file to trigger an event",
  fields: {
    dir: {
      argument: true,
      help: "Directory to watch"
    },
    fileName: {
      argument: true,
      default: "fswatch-demo.txt",
      help: "File name to create inside the watched directory"
    },
    recursive: {
      type: "bool",
      default: false,
      help: "Watch nested directories recursively"
    },
    debounceMs: {
      type: "int",
      default: 0,
      help: "Trailing debounce window in milliseconds"
    },
    include: {
      type: "stringList",
      default: [],
      help: "Glob patterns to include, for example **/*.js"
    },
    exclude: {
      type: "stringList",
      default: [],
      help: "Glob patterns to exclude, for example **/node_modules/**"
    }
  }
});
