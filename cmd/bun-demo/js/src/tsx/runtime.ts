export type HtmlChunk = {
  __html: string;
};

export type Child =
  | string
  | number
  | boolean
  | null
  | undefined
  | HtmlChunk
  | Child[];

type Props = {
  [key: string]: unknown;
  children?: Child | Child[];
};

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function isHtmlChunk(value: Child): value is HtmlChunk {
  return (
    typeof value === "object" &&
    value !== null &&
    Object.prototype.hasOwnProperty.call(value, "__html")
  );
}

function renderChild(child: Child): string {
  if (Array.isArray(child)) {
    var out = "";
    for (var i = 0; i < child.length; i++) {
      out += renderChild(child[i]);
    }
    return out;
  }
  if (child === null || child === undefined || child === false) {
    return "";
  }
  if (isHtmlChunk(child)) {
    return child.__html;
  }
  return escapeHtml(String(child));
}

function renderChildren(children?: Child | Child[]): string {
  if (children === undefined) {
    return "";
  }
  return renderChild(children as Child);
}

function renderAttrs(props: Props): string {
  var attrs = "";
  for (var key in props) {
    if (!Object.prototype.hasOwnProperty.call(props, key)) {
      continue;
    }
    if (key === "children" || key === "key") {
      continue;
    }
    var value = props[key];
    if (value === true) {
      attrs += " " + key;
      continue;
    }
    if (value === false || value === null || value === undefined) {
      continue;
    }
    attrs += " " + key + '="' + escapeHtml(String(value)) + '"';
  }
  return attrs;
}

function htmlChunk(markup: string): HtmlChunk {
  return { __html: markup };
}

export function renderToString(child: Child): string {
  return renderChild(child);
}

export function Fragment(props: Props): HtmlChunk {
  return htmlChunk(renderChildren(props ? props.children : undefined));
}

export function jsx(type: unknown, props: Props | null): HtmlChunk {
  var safeProps = props || {};
  if (arguments.length > 2) {
    var childArgs: Child[] = [];
    for (var i = 2; i < arguments.length; i++) {
      childArgs.push(arguments[i] as Child);
    }
    safeProps.children = childArgs.length === 1 ? childArgs[0] : childArgs;
  }
  if (typeof type === "function") {
    return (type as (p: Props) => HtmlChunk)(safeProps);
  }
  if (type === Fragment) {
    return Fragment(safeProps);
  }
  var attrs = renderAttrs(safeProps);
  var inner = renderChildren(safeProps.children);
  return htmlChunk("<" + type + attrs + ">" + inner + "</" + type + ">");
}
