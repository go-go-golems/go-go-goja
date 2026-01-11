import dayjs from "dayjs";
import { sumBy } from "lodash";
import logo from "./assets/logo.svg";
import * as exec from "exec";
import * as fs from "fs";

export function run(): string {
  var items: Array<{ n: number }> = [{ n: 2 }, { n: 3 }];
  var total = sumBy(items, "n");
  var stamp = dayjs().format("YYYY-MM-DD");
  var payload = stamp + ":" + total + ":" + logo.length;

  fs.writeFileSync("/tmp/goja-bun.txt", payload);

  return exec.run("cat", ["/tmp/goja-bun.txt"]).trim();
}
