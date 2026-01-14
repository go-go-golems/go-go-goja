import dayjs from "dayjs";
import { buildMetrics } from "./modules/metrics";

export function run(): string {
  var date = dayjs().format("YYYY-MM-DD");
  var metrics = buildMetrics();

  return ["mode=split", "date=" + date, metrics].join(" ");
}
