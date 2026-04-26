const EventEmitter = require("events");
const fs = require("fs");
const path = require("path");
const timer = require("timer");

async function watchAndWrite(dir, fileName) {
  const emitter = new EventEmitter();
  const events = [];
  const errors = [];

  emitter.on("event", (ev) => {
    if (String(ev.name).indexOf(fileName) >= 0) {
      events.push({
        source: ev.source,
        watchPath: ev.watchPath,
        name: ev.name,
        op: ev.op,
        create: ev.create,
        write: ev.write,
        remove: ev.remove,
        rename: ev.rename,
        chmod: ev.chmod
      });
    }
  });
  emitter.on("error", (err) => {
    errors.push(err && err.message ? err.message : String(err));
  });

  const conn = fswatch.watch(dir, emitter);
  const file = path.join(dir, fileName);
  fs.writeFileSync(file, "created by fswatch jsverb");

  for (let i = 0; i < 100 && events.length === 0 && errors.length === 0; i++) {
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
    op: first.op,
    create: first.create,
    write: first.write,
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
    }
  }
});
