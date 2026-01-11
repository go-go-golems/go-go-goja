import { Fragment, jsx, renderToString } from "./runtime";
import { App } from "./App";

export function renderHtml(): string {
  var html = renderToString(
    <App title="Goja TSX Demo" items={["alpha", "beta", "gamma"]} />
  );

  return "<!doctype html>" + html;
}
