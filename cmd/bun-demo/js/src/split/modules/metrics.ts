import _ from "lodash";
import logoSvg from "../assets/logo.svg";

function countTags(svg: string): number {
  return (svg.match(/</g) || []).length;
}

export function buildMetrics(): string {
  var codepoints = _.map(logoSvg.split(""), function (ch) {
    return ch.charCodeAt(0);
  });
  var svgCsum = _.sum(codepoints);
  var svgTags = countTags(logoSvg);

  return [
    "svgLen=" + logoSvg.length,
    "svgTags=" + svgTags,
    "svgCsum=" + svgCsum,
  ].join(" ");
}
