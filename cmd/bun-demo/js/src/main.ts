import dayjs from "dayjs";
import * as lodash from "lodash";
import logo from "./assets/logo.svg";
import * as exec from "exec";
import * as fs from "fs";

export function run(): string {
  var items: Array<{ n: number }> = [{ n: 2 }, { n: 3 }];
  var total = lodash.sumBy(items, "n");
  var stamp = dayjs().format("YYYY-MM-DD");
  var logoChars = logo.split("");
  var counts = lodash.countBy(logoChars, function (ch: string) {
    return ch;
  });
  var tagCount = counts["<"] || 0;
  var checksum = lodash.reduce(
    logoChars,
    function (acc: number, ch: string) {
      return (acc + ch.charCodeAt(0)) % 100000;
    },
    0
  );
  var payload =
    "date=" +
    stamp +
    " sum=" +
    total +
    " svgLen=" +
    logo.length +
    " svgTags=" +
    tagCount +
    " svgCsum=" +
    checksum;

  fs.writeFileSync("/tmp/goja-bun.txt", payload);

  return exec.run("cat", ["/tmp/goja-bun.txt"]).trim();
}
