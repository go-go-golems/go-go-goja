"use strict";
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = function(target, all) {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = function(to, from, except, desc) {
  if (from && typeof from === "object" || typeof from === "function")
    for (var keys = __getOwnPropNames(from), i = 0, n = keys.length, key; i < n; i++) {
      key = keys[i];
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: function(k) {
          return from[k];
        }.bind(null, key), enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
  return to;
};
var __toCommonJS = function(mod) {
  return __copyProps(__defProp({}, "__esModule", { value: true }), mod);
};

// src/tsx/entry.tsx
var entry_exports = {};
__export(entry_exports, {
  renderHtml: function() {
    return renderHtml;
  },
  run: function() {
    return run;
  }
});
module.exports = __toCommonJS(entry_exports);

// src/tsx/runtime.ts
function escapeHtml(value) {
  return value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/\"/g, "&quot;").replace(/'/g, "&#39;");
}
function isHtmlChunk(value) {
  return typeof value === "object" && value !== null && Object.prototype.hasOwnProperty.call(value, "__html");
}
function renderChild(child) {
  if (Array.isArray(child)) {
    var out = "";
    for (var i = 0; i < child.length; i++) {
      out += renderChild(child[i]);
    }
    return out;
  }
  if (child === null || child === void 0 || child === false) {
    return "";
  }
  if (isHtmlChunk(child)) {
    return child.__html;
  }
  return escapeHtml(String(child));
}
function renderChildren(children) {
  if (children === void 0) {
    return "";
  }
  return renderChild(children);
}
function renderAttrs(props) {
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
    if (value === false || value === null || value === void 0) {
      continue;
    }
    attrs += " " + key + '="' + escapeHtml(String(value)) + '"';
  }
  return attrs;
}
function htmlChunk(markup) {
  return { __html: markup };
}
function renderToString(child) {
  return renderChild(child);
}
function Fragment(props) {
  return htmlChunk(renderChildren(props ? props.children : void 0));
}
function jsx(type, props) {
  var safeProps = props || {};
  if (arguments.length > 2) {
    var childArgs = [];
    for (var i = 2; i < arguments.length; i++) {
      childArgs.push(arguments[i]);
    }
    safeProps.children = childArgs.length === 1 ? childArgs[0] : childArgs;
  }
  if (typeof type === "function") {
    return type(safeProps);
  }
  if (type === Fragment) {
    return Fragment(safeProps);
  }
  var attrs = renderAttrs(safeProps);
  var inner = renderChildren(safeProps.children);
  return htmlChunk("<" + type + attrs + ">" + inner + "</" + type + ">");
}

// src/tsx/App.tsx
function App(props) {
  return /* @__PURE__ */ jsx("main", null, /* @__PURE__ */ jsx("header", null, /* @__PURE__ */ jsx("h1", null, props.title), /* @__PURE__ */ jsx("p", null, "Rendered from TSX inside Goja.")), /* @__PURE__ */ jsx("section", null, /* @__PURE__ */ jsx("ul", null, props.items.map(function(item) {
    return /* @__PURE__ */ jsx("li", { key: item }, item);
  }))));
}

// src/tsx/render.tsx
function renderHtml() {
  var html = renderToString(
    /* @__PURE__ */ jsx(App, { title: "Goja TSX Demo", items: ["alpha", "beta", "gamma"] })
  );
  return "<!doctype html>" + html;
}

// src/tsx/entry.tsx
function run() {
  return renderHtml();
}
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  renderHtml,
  run
});
