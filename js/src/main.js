var dayjs = require("dayjs");
var lodash = require("lodash");
var fs = require("fs");
var exec = require("exec");

function run() {
  var items = [{ n: 2 }, { n: 3 }];
  var total = lodash.sumBy(items, "n");
  var stamp = dayjs().format("YYYY-MM-DD");
  var payload = stamp + ":" + total;

  fs.writeFileSync("/tmp/goja-bun.txt", payload);

  return exec.run("cat", ["/tmp/goja-bun.txt"]).trim();
}

module.exports = { run: run };
