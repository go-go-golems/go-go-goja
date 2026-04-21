import { useState, useEffect, useCallback, useRef } from "react";

/* ═══════════════════════════════════════════
   MOCK DATA
   ═══════════════════════════════════════════ */

const MOCK_SESSION = {
  id: "sess_7f3a2b",
  profile: "interactive",
  createdAt: "2026-04-14T19:22:08Z",
  cellCount: 0,
  bindingCount: 0,
  policy: {
    eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true },
    observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true, jsdocExtraction: false },
    persist: { enabled: false, sessions: false, evaluations: false, bindingVersions: false, bindingDocs: false },
  },
};

const PROFILES = {
  raw: {
    label: "raw",
    eval: { mode: "direct", timeoutMs: 5000, captureLastExpression: false, supportTopLevelAwait: false },
    observe: { staticAnalysis: false, runtimeSnapshot: false, bindingTracking: false, consoleCapture: false },
    persist: { enabled: false },
  },
  interactive: {
    label: "interactive",
    eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true },
    observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true },
    persist: { enabled: false },
  },
  persistent: {
    label: "persistent",
    eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true },
    observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true },
    persist: { enabled: true, sessions: true, evaluations: true, bindingVersions: true, bindingDocs: true },
  },
};

const MOCK_EVAL_RESULT = {
  source: "const x = 1; x",
  rewrite: {
    mode: "instrumented",
    transformedSource: 'var x = __capture("x", 1); __lastExpr(x);',
    declaredNames: ["x"],
    helperNames: ["__capture", "__lastExpr"],
    operations: [
      { kind: "const-to-var", detail: "const x → var x (binding persistence)" },
      { kind: "capture-wrap", detail: "wrapped initializer with __capture()" },
      { kind: "last-expr", detail: "wrapped trailing expression with __lastExpr()" },
    ],
  },
  execution: { status: "ok", result: "1", durationMs: 2, awaited: false, console: [] },
  static: {
    topLevelBindings: [{ name: "x", kind: "const", line: 1 }],
    diagnostics: [],
    unresolvedCount: 0,
    astNodeCount: 12,
  },
  runtime: {
    newBindings: ["x"],
    updatedBindings: [],
    removedBindings: [],
    leakedGlobals: [],
    diffs: [{ name: "x", before: "undefined", after: "1" }],
  },
};

const MOCK_BINDINGS = [
  { name: "x", kind: "const", origin: "user", declaredInCell: 1, value: "1" },
  { name: "answer", kind: "const", origin: "user", declaredInCell: 2, value: "42" },
  { name: "greet", kind: "const", origin: "user", declaredInCell: 3, value: "fn()" },
];

const MOCK_HISTORY = [
  { cellId: 1, source: "const x = 1; x", result: "1", status: "ok" },
  { cellId: 2, source: "const answer = 41 + 1; answer", result: "42", status: "ok" },
  { cellId: 3, source: 'const greet = (n) => `hello ${n}`', result: "fn()", status: "ok" },
  { cellId: 4, source: "while(true){}", result: "", status: "timeout" },
  { cellId: 5, source: "greet('world')", result: '"hello world"', status: "ok" },
];

const TIMEOUT_SCENARIOS = [
  { id: "infinite-loop", label: "while (true) {}", source: "while (true) {}" },
  { id: "never-settle", label: "new Promise(() => {})", source: "new Promise(() => {})" },
  { id: "recovery", label: "1 + 1  (after timeout)", source: "1 + 1" },
];

/* ═══════════════════════════════════════════
   FONTS & COLORS
   ═══════════════════════════════════════════ */

const FONT = `"Geneva", "ChicagoFLF", "Chicago", ui-monospace, "SF Mono", Monaco, monospace`;
const FONT_MONO = `"Monaco", "Menlo", "Courier New", monospace`;

const C = {
  blue: "#1c50a0",
  green: "#1a6e30",
  red: "#b82020",
  purple: "#5828a0",
  orange: "#a85010",
  teal: "#0e6068",
  comment: "#707070",
};

/* ═══════════════════════════════════════════
   SYNTAX HIGHLIGHTING
   ═══════════════════════════════════════════ */

function tokenize(code) {
  const tokens = [];
  const re = /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)|(\/\/.*$|\/\*[\s\S]*?\*\/)|\b(const|let|var|function|return|if|else|while|for|new|await|true|false|null|undefined|typeof|class|import|export|from|of|in)\b|(\b\d+(?:\.\d+)?\b)|(\b[a-zA-Z_]\w*(?=\s*\())|([{}()\[\];,=+\-*/<>!&|?.:\\])|(\S+)/gm;
  let m, last = 0;
  while ((m = re.exec(code)) !== null) {
    if (m.index > last) tokens.push({ t: code.slice(last, m.index) });
    if (m[1]) tokens.push({ t: m[0], c: C.green });
    else if (m[2]) tokens.push({ t: m[0], c: C.comment });
    else if (m[3]) tokens.push({ t: m[0], c: C.blue });
    else if (m[4]) tokens.push({ t: m[0], c: C.orange });
    else if (m[5]) tokens.push({ t: m[0], c: C.purple });
    else tokens.push({ t: m[0] });
    last = m.index + m[0].length;
  }
  if (last < code.length) tokens.push({ t: code.slice(last) });
  return tokens;
}

function Code({ children: code, style: extra }) {
  const toks = tokenize(code);
  return (
    <pre style={{
      fontFamily: FONT_MONO, fontSize: 11.5, lineHeight: 1.55,
      background: "#f4f4f0", border: "1px solid #b0b0b0",
      padding: "7px 10px", margin: 0, whiteSpace: "pre-wrap",
      wordBreak: "break-all", ...extra,
    }}>
      {toks.map((tk, i) => tk.c
        ? <span key={i} style={{ color: tk.c }}>{tk.t}</span>
        : <span key={i}>{tk.t}</span>
      )}
    </pre>
  );
}

/* ═══════════════════════════════════════════
   JSON COLORIZER
   ═══════════════════════════════════════════ */

function colorJSON(obj) {
  const raw = JSON.stringify(obj, null, 2);
  const parts = [];
  const lines = raw.split("\n");
  for (let li = 0; li < lines.length; li++) {
    let line = lines[li];
    const segs = [];
    let rest = line;
    // key
    const km = rest.match(/^(\s*)"([^"]+)"(\s*:\s*)/);
    if (km) {
      segs.push(km[1]);
      segs.push(<span key={`k${li}`} style={{ color: C.blue }}>"{km[2]}"</span>);
      segs.push(km[3]);
      rest = rest.slice(km[0].length);
    }
    // value
    const sm = rest.match(/^"((?:[^"\\]|\\.)*)"(,?)$/);
    const nm = rest.match(/^(\d+(?:\.\d+)?)(,?)$/);
    const bm = rest.match(/^(true|false|null)(,?)$/);
    if (sm) { segs.push(<span key={`v${li}`} style={{ color: C.green }}>"{sm[1]}"</span>); segs.push(sm[2]); }
    else if (nm) { segs.push(<span key={`v${li}`} style={{ color: C.orange }}>{nm[1]}</span>); segs.push(nm[2]); }
    else if (bm) { segs.push(<span key={`v${li}`} style={{ color: C.purple }}>{bm[1]}</span>); segs.push(bm[2]); }
    else { segs.push(rest); }
    parts.push(<span key={li}>{segs}{"\n"}</span>);
  }
  return parts;
}

function JSONViewer({ data, label }) {
  const [open, setOpen] = useState(false);
  return (
    <div style={{ margin: "6px 0" }}>
      <button onClick={() => setOpen(!open)} style={{
        fontFamily: FONT, fontSize: 10.5, fontWeight: 700,
        background: "none", border: "1px solid #999", padding: "2px 8px",
        cursor: "pointer", color: "#555",
      }}>
        {open ? "▾" : "▸"} {label || "Inspect JSON"}
      </button>
      {open && (
        <pre style={{
          fontFamily: FONT_MONO, fontSize: 10.5, lineHeight: 1.45,
          background: "#f8f8f4", border: "1px solid #b0b0b0",
          padding: "8px 10px", marginTop: 4,
          maxHeight: 280, overflow: "auto", whiteSpace: "pre-wrap",
        }}>
          {colorJSON(data)}
        </pre>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════
   UI PRIMITIVES — System 1 flavor
   ═══════════════════════════════════════════ */

function Win({ title, children, style: extra }) {
  return (
    <div style={{
      border: "2px solid #000", margin: "10px 0",
      background: "#fff", ...extra,
    }}>
      <div style={{
        background: "#000", color: "#fff", padding: "3px 8px",
        fontSize: 11.5, fontWeight: 700, fontFamily: FONT,
        display: "flex", alignItems: "center", gap: 6,
        userSelect: "none",
      }}>
        <div style={{
          width: 11, height: 11, border: "1.5px solid #fff",
          flexShrink: 0, boxSizing: "border-box",
        }} />
        <span style={{ letterSpacing: 0.3 }}>{title}</span>
      </div>
      <div style={{ padding: "10px 12px" }}>{children}</div>
    </div>
  );
}

const btnBase = {
  fontFamily: FONT, fontSize: 11.5, fontWeight: 700,
  padding: "4px 14px", border: "2px solid #000",
  background: "#fff", color: "#000", cursor: "pointer",
  boxShadow: "2px 2px 0 #000", userSelect: "none",
};
const btnOn = { background: "#000", color: "#fff" };
const btnSm = { padding: "2px 9px", fontSize: 10.5, boxShadow: "1px 1px 0 #000" };

function Btn({ children, on, sm, style: extra, ...rest }) {
  return (
    <button style={{ ...btnBase, ...(sm ? btnSm : {}), ...(on ? btnOn : {}), ...extra }} {...rest}>
      {children}
    </button>
  );
}

function Dither() {
  return <div style={{
    height: 4, margin: "28px 0",
    backgroundImage: `repeating-linear-gradient(90deg, #000 0px, #000 2px, transparent 2px, transparent 4px)`,
    backgroundSize: "4px 2px", backgroundRepeat: "repeat-x",
    backgroundPosition: "0 1px",
  }} />;
}

function Heading({ n, children }) {
  return (
    <h2 style={{
      fontSize: 15.5, fontWeight: 700, fontFamily: FONT,
      margin: "0 0 2px", display: "flex", alignItems: "center", gap: 8,
    }}>
      <span style={{
        display: "inline-flex", alignItems: "center", justifyContent: "center",
        width: 22, height: 22, border: "2px solid #000", fontSize: 12,
        fontWeight: 700, flexShrink: 0, lineHeight: 1,
      }}>{n}</span>
      {children}
    </h2>
  );
}

function Tag({ children, v }) {
  const colors = { ok: C.green, timeout: C.red, new: C.blue };
  const color = colors[v] || "#000";
  return (
    <span style={{
      display: "inline-block", fontSize: 9.5, fontWeight: 700,
      padding: "1px 5px", border: `1px solid ${color}`,
      color, textTransform: "uppercase", letterSpacing: 0.5,
      marginRight: 4, lineHeight: 1.4,
    }}>{children}</span>
  );
}

function Callout({ children }) {
  return (
    <div style={{
      border: "2px solid #000", borderLeft: "6px solid #000",
      padding: "8px 12px", margin: "10px 0", fontSize: 12,
      background: "#fafaf6", lineHeight: 1.55,
    }}>{children}</div>
  );
}

function Row({ children, style: extra }) {
  return <div style={{ display: "flex", gap: 12, flexWrap: "wrap", ...extra }}>{children}</div>;
}

function Col({ children, style: extra }) {
  return <div style={{ flex: 1, minWidth: 220, ...extra }}>{children}</div>;
}

function Prose({ children }) {
  return <p style={{ margin: "6px 0 14px", fontSize: 13, lineHeight: 1.6, maxWidth: 640 }}>{children}</p>;
}

function PolicyRow({ label, value }) {
  const yes = value === true;
  const no = value === false;
  return (
    <tr>
      <td style={{ padding: "3px 8px", borderBottom: "1px solid #ddd", fontSize: 12 }}>{label}</td>
      <td style={{
        padding: "3px 8px", borderBottom: "1px solid #ddd", fontSize: 12,
        fontWeight: 600,
        color: yes ? C.green : no ? "#aaa" : C.blue,
      }}>
        {typeof value === "boolean" ? (value ? "✓ yes" : "— no") : String(value)}
      </td>
    </tr>
  );
}

const tbl = { width: "100%", borderCollapse: "collapse", fontFamily: FONT, fontSize: 12 };
const th = {
  textAlign: "left", borderBottom: "2px solid #000", padding: "4px 8px",
  fontWeight: 700, fontSize: 10.5, textTransform: "uppercase", letterSpacing: 0.5,
};
const td = { padding: "4px 8px", borderBottom: "1px solid #ddd", verticalAlign: "top", fontSize: 12 };

/* ═══════════════════════════════════════════
   SECTIONS
   ═══════════════════════════════════════════ */

function S1() {
  const [session, setSession] = useState(null);
  const [loading, setLoading] = useState(false);
  const create = () => { setLoading(true); setTimeout(() => { setSession({ ...MOCK_SESSION, createdAt: new Date().toISOString() }); setLoading(false); }, 600); };

  return (<div>
    <Heading n="1">Meet a Session</Heading>
    <Prose>
      A session is more than a prompt. It has an <span style={{ color: C.blue, fontWeight: 600 }}>identity</span>,
      a <span style={{ color: C.blue, fontWeight: 600 }}>profile</span>,
      a <span style={{ color: C.blue, fontWeight: 600 }}>policy</span>,
      and evolving state. Press the button to create one and see what the backend returns.
    </Prose>
    {!session ? (
      <div style={{ textAlign: "center", padding: "18px 0" }}>
        {loading
          ? <span style={{ fontSize: 12, color: "#888" }}>Creating session…</span>
          : <Btn onClick={create}>◆ Create Session</Btn>
        }
        {!loading && <div style={{ marginTop: 10, fontSize: 10.5, color: "#999", fontStyle: "italic", fontFamily: FONT_MONO }}>POST /api/sessions</div>}
      </div>
    ) : (
      <Row>
        <Col>
          <Win title="Session Summary">
            <table style={tbl}><tbody>
              <tr><td style={{ ...td, fontWeight: 700, width: 110 }}>ID</td><td style={{ ...td, fontFamily: FONT_MONO, color: C.blue }}>{session.id}</td></tr>
              <tr><td style={{ ...td, fontWeight: 700 }}>Profile</td><td style={td}>{session.profile}</td></tr>
              <tr><td style={{ ...td, fontWeight: 700 }}>Created</td><td style={{ ...td, fontSize: 10.5 }}>{new Date(session.createdAt).toLocaleTimeString()}</td></tr>
              <tr><td style={{ ...td, fontWeight: 700 }}>Cells</td><td style={td}>{session.cellCount}</td></tr>
              <tr><td style={{ ...td, fontWeight: 700 }}>Bindings</td><td style={td}>{session.bindingCount}</td></tr>
            </tbody></table>
          </Win>
        </Col>
        <Col>
          <Win title="Policy Card">
            <div style={{ fontSize: 10.5, fontWeight: 700, marginBottom: 3, textTransform: "uppercase", letterSpacing: 0.5 }}>Eval</div>
            <table style={tbl}><tbody>
              <PolicyRow label="mode" value={session.policy.eval.mode} />
              <PolicyRow label="timeout" value={`${session.policy.eval.timeoutMs}ms`} />
              <PolicyRow label="capture last expr" value={session.policy.eval.captureLastExpression} />
              <PolicyRow label="top-level await" value={session.policy.eval.supportTopLevelAwait} />
            </tbody></table>
            <div style={{ fontSize: 10.5, fontWeight: 700, margin: "8px 0 3px", textTransform: "uppercase", letterSpacing: 0.5 }}>Observe</div>
            <table style={tbl}><tbody>
              <PolicyRow label="static analysis" value={session.policy.observe.staticAnalysis} />
              <PolicyRow label="runtime snapshot" value={session.policy.observe.runtimeSnapshot} />
              <PolicyRow label="binding tracking" value={session.policy.observe.bindingTracking} />
            </tbody></table>
            <div style={{ fontSize: 10.5, fontWeight: 700, margin: "8px 0 3px", textTransform: "uppercase", letterSpacing: 0.5 }}>Persist</div>
            <table style={tbl}><tbody>
              <PolicyRow label="enabled" value={session.policy.persist.enabled} />
            </tbody></table>
          </Win>
        </Col>
      </Row>
    )}
    {session && <JSONViewer data={session} label="Raw SessionSummary" />}
  </div>);
}

function S2() {
  const [sel, setSel] = useState("interactive");
  const p = PROFILES[sel];
  return (<div>
    <Heading n="2">Profiles Change Behavior</Heading>
    <Prose>Three profiles, three different contracts. The difference is not cosmetic — it changes what the engine does with your code.</Prose>
    <div style={{ display: "inline-flex", border: "2px solid #000", marginBottom: 12 }}>
      {["raw", "interactive", "persistent"].map((pr, i, a) => (
        <div key={pr} onClick={() => setSel(pr)} style={{
          padding: "3px 14px", fontSize: 11, fontWeight: 700, fontFamily: FONT,
          cursor: "pointer", userSelect: "none",
          borderRight: i < a.length - 1 ? "1px solid #000" : "none",
          background: sel === pr ? "#000" : "#fff",
          color: sel === pr ? "#fff" : "#000",
        }}>{pr}</div>
      ))}
    </div>
    <Win title={`Profile: ${sel}`}>
      <table style={tbl}><thead><tr><th style={th}>Property</th><th style={th}>Value</th></tr></thead><tbody>
        <PolicyRow label="eval mode" value={p.eval.mode} />
        <PolicyRow label="timeout" value={`${p.eval.timeoutMs}ms`} />
        <PolicyRow label="capture last expression" value={p.eval.captureLastExpression || false} />
        <PolicyRow label="top-level await" value={p.eval.supportTopLevelAwait || false} />
        <PolicyRow label="static analysis" value={p.observe.staticAnalysis || false} />
        <PolicyRow label="binding tracking" value={p.observe.bindingTracking || false} />
        <PolicyRow label="persistence" value={p.persist.enabled || false} />
      </tbody></table>
    </Win>
    <Callout><strong>Design note:</strong> The HTTP create-session route does not yet accept a profile override body. This comparison is mocked from known profile configurations.</Callout>
  </div>);
}

function S3() {
  const [source, setSource] = useState("const x = 1; x");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const evaluate = () => { setLoading(true); setTimeout(() => { setResult(MOCK_EVAL_RESULT); setLoading(false); }, 500); };

  return (<div>
    <Heading n="3">What Happened to My Code?</Heading>
    <Prose>Evaluation is not just "run it." The system may <span style={{ color: C.purple, fontWeight: 600 }}>rewrite</span> your source before execution — wrapping declarations, capturing expressions, inserting helpers.</Prose>
    <Win title="Source Editor">
      <textarea value={source} onChange={e => setSource(e.target.value)} rows={2} spellCheck={false} style={{
        fontFamily: FONT_MONO, fontSize: 12, width: "100%", border: "none",
        outline: "none", resize: "vertical", minHeight: 40, background: "transparent",
        lineHeight: 1.5, padding: 0,
      }} />
      <div style={{ marginTop: 6 }}>
        <Btn onClick={evaluate} disabled={loading}>{loading ? "evaluating…" : "▶ Evaluate"}</Btn>
        <span style={{ marginLeft: 10, fontSize: 10.5, color: "#999", fontFamily: FONT_MONO }}>POST /api/sessions/:id/evaluate</span>
      </div>
    </Win>
    {result && (<>
      <Row>
        <Col><Win title="Original Source"><Code>{result.source}</Code></Win></Col>
        <div style={{ display: "flex", alignItems: "center", padding: "0 2px", fontSize: 20, fontWeight: 700 }}>→</div>
        <Col><Win title="Transformed Source"><Code>{result.rewrite.transformedSource}</Code></Win></Col>
      </Row>
      <Win title="Rewrite Operations">
        {result.rewrite.operations.map((op, i) => (
          <div key={i} style={{ display: "flex", gap: 8, padding: "3px 0", fontSize: 12, borderBottom: "1px solid #eee", alignItems: "center" }}>
            <Tag v="new">{op.kind}</Tag><span>{op.detail}</span>
          </div>
        ))}
      </Win>
      <Win title="Execution Result">
        <div style={{ display: "flex", gap: 16, fontSize: 12, flexWrap: "wrap" }}>
          <span><strong>Status:</strong> <span style={{ color: C.green }}>{result.execution.status}</span></span>
          <span><strong>Result:</strong> <span style={{ color: C.blue, fontFamily: FONT_MONO }}>{result.execution.result}</span></span>
          <span><strong>Duration:</strong> {result.execution.durationMs}ms</span>
        </div>
      </Win>
      <JSONViewer data={result} label="Raw EvaluateResponse" />
    </>)}
  </div>);
}

function S4() {
  const s = MOCK_EVAL_RESULT.static;
  const r = MOCK_EVAL_RESULT.runtime;
  return (<div>
    <Heading n="4">Static Analysis vs Runtime Reality</Heading>
    <Prose>Some facts come from <span style={{ color: C.teal, fontWeight: 600 }}>parsing</span> (before execution) and some from <span style={{ color: C.orange, fontWeight: 600 }}>runtime inspection</span> (after execution). Provenance tells you which is which.</Prose>
    <Row>
      <Col>
        <Win title="Before Execution (Static)">
          <table style={tbl}><tbody>
            <tr><td style={{ ...td, fontWeight: 700 }}>Bindings found</td><td style={td}>{s.topLevelBindings.length}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>Diagnostics</td><td style={td}>{s.diagnostics.length}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>Unresolved</td><td style={td}>{s.unresolvedCount}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>AST nodes</td><td style={td}>{s.astNodeCount}</td></tr>
          </tbody></table>
          <div style={{ marginTop: 8, fontSize: 11.5 }}>
            {s.topLevelBindings.map((b, i) => (
              <div key={i}><span style={{ color: C.blue }}>{b.kind}</span> <span style={{ fontWeight: 600 }}>{b.name}</span> <span style={{ color: "#999" }}>line {b.line}</span></div>
            ))}
          </div>
        </Win>
      </Col>
      <Col>
        <Win title="After Execution (Runtime)">
          <table style={tbl}><tbody>
            <tr><td style={{ ...td, fontWeight: 700 }}>New bindings</td><td style={{ ...td, color: C.green }}>{r.newBindings.join(", ") || "—"}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>Updated</td><td style={td}>{r.updatedBindings.join(", ") || "—"}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>Removed</td><td style={td}>{r.removedBindings.join(", ") || "—"}</td></tr>
            <tr><td style={{ ...td, fontWeight: 700 }}>Leaked globals</td><td style={{ ...td, color: r.leakedGlobals.length ? C.red : "#999" }}>{r.leakedGlobals.join(", ") || "—"}</td></tr>
          </tbody></table>
          <div style={{ marginTop: 8, fontSize: 11.5 }}>
            <div style={{ fontWeight: 700, marginBottom: 3 }}>Global diffs:</div>
            {r.diffs.map((d, i) => (
              <div key={i}><span style={{ fontWeight: 600 }}>{d.name}</span>: <span style={{ color: "#999" }}>{d.before}</span> → <span style={{ color: C.green }}>{d.after}</span></div>
            ))}
          </div>
        </Win>
      </Col>
    </Row>
  </div>);
}

function S5() {
  const [sel, setSel] = useState(0);
  const vis = MOCK_BINDINGS.filter(b => b.declaredInCell <= sel + 1);
  return (<div>
    <Heading n="5">Bindings Are the Memory</Heading>
    <Prose>The session accumulates meaning through <span style={{ color: C.blue, fontWeight: 600 }}>bindings</span>, not just a list of previous commands. Scrub the timeline to see the environment grow.</Prose>
    <Win title="Session Timeline">
      <div style={{ display: "flex", gap: 6, flexWrap: "wrap", marginBottom: 6 }}>
        {MOCK_HISTORY.slice(0, 3).map((h, i) => (
          <Btn key={i} sm on={sel === i} onClick={() => setSel(i)}
            style={{ fontFamily: FONT_MONO, fontSize: 10, maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            Cell {h.cellId}: {h.source}
          </Btn>
        ))}
      </div>
      <div style={{ fontSize: 11, color: "#777" }}>Showing environment after cell {sel + 1}</div>
    </Win>
    <Win title="Current Bindings">
      <table style={tbl}>
        <thead><tr><th style={th}>Name</th><th style={th}>Kind</th><th style={th}>Value</th><th style={th}>Cell</th></tr></thead>
        <tbody>
          {vis.length === 0
            ? <tr><td colSpan={4} style={{ ...td, color: "#999", fontStyle: "italic" }}>No bindings yet</td></tr>
            : vis.map((b, i) => (
              <tr key={i}>
                <td style={{ ...td, fontWeight: 600, color: C.blue }}>{b.name}</td>
                <td style={{ ...td, color: C.blue }}>{b.kind}</td>
                <td style={{ ...td, fontFamily: FONT_MONO }}>{b.value}</td>
                <td style={td}>{b.declaredInCell}</td>
              </tr>
            ))
          }
        </tbody>
      </table>
    </Win>
  </div>);
}

function S6() {
  const [restored, setRestored] = useState(false);
  const sessions = [
    { id: "sess_7f3a2b", cells: 5, age: "2m ago" },
    { id: "sess_a1c9e0", cells: 12, age: "1h ago" },
    { id: "sess_d4f7b2", cells: 3, age: "3h ago" },
  ];
  return (<div>
    <Heading n="6">Persistence, History, and Restore</Heading>
    <Prose>Persistent mode changes the mental model from "temporary REPL" to "recoverable session." History survives. Bindings survive. The session can be exported and restored.</Prose>
    <Row>
      <Col style={{ maxWidth: 230 }}>
        <Win title="Durable Sessions">
          {sessions.map((s, i) => (
            <div key={i} style={{
              padding: "4px 6px", borderBottom: "1px solid #ddd", fontSize: 11,
              cursor: "pointer",
              background: i === 0 ? "#000" : "transparent",
              color: i === 0 ? "#fff" : "#000",
            }}>
              <span style={{ fontFamily: FONT_MONO }}>{s.id}</span>
              <span style={{ float: "right", opacity: 0.6 }}>{s.cells} cells · {s.age}</span>
            </div>
          ))}
        </Win>
      </Col>
      <Col>
        <Win title="History — sess_7f3a2b">
          <table style={tbl}>
            <thead><tr><th style={th}>#</th><th style={th}>Source</th><th style={th}>Result</th><th style={th}>Status</th></tr></thead>
            <tbody>
              {MOCK_HISTORY.map((h, i) => (
                <tr key={i}>
                  <td style={td}>{h.cellId}</td>
                  <td style={{ ...td, fontFamily: FONT_MONO, fontSize: 11 }}>{h.source}</td>
                  <td style={{ ...td, fontFamily: FONT_MONO, fontSize: 11, color: h.status === "ok" ? C.green : C.red }}>{h.result || "—"}</td>
                  <td style={td}><Tag v={h.status}>{h.status}</Tag></td>
                </tr>
              ))}
            </tbody>
          </table>
        </Win>
        <div style={{ display: "flex", gap: 8, marginTop: 6, alignItems: "center", flexWrap: "wrap" }}>
          <Btn sm onClick={() => setRestored(false)}>✕ Kill</Btn>
          <Btn sm on={restored} onClick={() => setRestored(true)}>↻ Restore</Btn>
          {restored && <span style={{ fontSize: 11, color: C.green, fontWeight: 600 }}>✓ Session restored — 5 cells, 3 bindings</span>}
        </div>
      </Col>
    </Row>
  </div>);
}

function S7() {
  const [sc, setSc] = useState(null);
  const [phase, setPhase] = useState(null);
  const run = (s) => {
    setSc(s); setPhase("running");
    if (s.id === "recovery") setTimeout(() => setPhase("ok"), 500);
    else setTimeout(() => setPhase("timeout"), 1500);
  };
  return (<div>
    <Heading n="7">Timeouts Are Part of the Contract</Heading>
    <Prose>Timeouts are not just errors. They are <span style={{ color: C.green, fontWeight: 600 }}>managed recovery behavior</span>. The session survives. The next cell still works.</Prose>
    <Win title="Canned Scenarios">
      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
        {TIMEOUT_SCENARIOS.map(s => (
          <Btn key={s.id} sm on={sc?.id === s.id} onClick={() => run(s)} style={{ fontFamily: FONT_MONO }}>{s.label}</Btn>
        ))}
      </div>
    </Win>
    {sc && (
      <Win title="Execution Timeline">
        <div style={{ fontSize: 12, padding: "2px 0" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
            <span style={{ fontFamily: FONT_MONO, fontSize: 11 }}>{sc.source}</span>
            {phase === "running" && <span style={{ color: "#999" }}>⏳ running…</span>}
            {phase === "timeout" && (<>
              <Tag v="timeout">timeout</Tag>
              <span style={{ color: C.red, fontSize: 11 }}>Interrupted after 5000ms. Session intact.</span>
            </>)}
            {phase === "ok" && (<>
              <Tag v="ok">ok</Tag>
              <span style={{ color: C.green, fontWeight: 600 }}>→ 2</span>
              <span style={{ color: "#666", fontSize: 11 }}>Session recovered. Proof that timeouts don't corrupt state.</span>
            </>)}
          </div>
          {phase === "timeout" && (
            <Callout><strong>Try it:</strong> Now run "1 + 1 (after timeout)" to prove the session still works.</Callout>
          )}
        </div>
      </Win>
    )}
  </div>);
}

function S8() {
  const rows = [
    { section: "static.topLevelBindings", source: "parser", notes: "SWC AST visitor" },
    { section: "rewrite.operations", source: "rewriter", notes: "instrumented transform pipeline" },
    { section: "runtime.diffs", source: "runtime", notes: "global snapshot before/after" },
    { section: "bindings[].runtime", source: "runtime", notes: "captured from Goja VM" },
    { section: "history", source: "persistence", notes: "SQLite via repldb" },
  ];
  const sc = { parser: C.teal, rewriter: C.purple, runtime: C.orange, persistence: C.blue };
  return (<div>
    <Heading n="8">Docs and Provenance</Heading>
    <Prose>The REPL produces more than values. It can preserve documentation and explain where its insights came from: <span style={{ color: C.teal }}>parser-derived</span>, <span style={{ color: C.orange }}>runtime-derived</span>, or <span style={{ color: C.purple }}>persisted</span>.</Prose>
    <Win title="Provenance Inspector">
      <table style={tbl}>
        <thead><tr><th style={th}>Section</th><th style={th}>Source</th><th style={th}>Notes</th></tr></thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i}>
              <td style={{ ...td, fontFamily: FONT_MONO, fontSize: 11, color: C.blue }}>{r.section}</td>
              <td style={{ ...td, color: sc[r.source] || "#000", fontWeight: 600 }}>{r.source}</td>
              <td style={{ ...td, fontSize: 11, color: "#666" }}>{r.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Win>
    <div style={{ fontSize: 10.5, color: "#999", fontFamily: FONT_MONO }}>GET /api/sessions/:id/docs</div>
  </div>);
}

function S9() {
  const routes = [
    { m: "GET", p: "/api/sessions", d: "List all sessions" },
    { m: "POST", p: "/api/sessions", d: "Create a new session" },
    { m: "GET", p: "/api/sessions/:id", d: "Session snapshot" },
    { m: "DELETE", p: "/api/sessions/:id", d: "Delete session" },
    { m: "POST", p: "/api/sessions/:id/evaluate", d: "Evaluate code" },
    { m: "POST", p: "/api/sessions/:id/restore", d: "Restore from persistence" },
    { m: "GET", p: "/api/sessions/:id/history", d: "Evaluation history" },
    { m: "GET", p: "/api/sessions/:id/bindings", d: "Current bindings" },
    { m: "GET", p: "/api/sessions/:id/docs", d: "Binding documentation" },
    { m: "GET", p: "/api/sessions/:id/export", d: "Full session export" },
  ];
  const mc = { GET: C.blue, POST: C.green, DELETE: C.red };
  return (<div>
    <Heading n="9">API Appendix</Heading>
    <Prose>Everything above talks to a real HTTP API. Here is the complete route surface.</Prose>
    <Win title="Route Table">
      <table style={tbl}>
        <thead><tr><th style={th}>Method</th><th style={th}>Path</th><th style={th}>Description</th></tr></thead>
        <tbody>
          {routes.map((r, i) => (
            <tr key={i}>
              <td style={{ ...td, fontFamily: FONT_MONO, fontWeight: 700, fontSize: 11, color: mc[r.m] || "#000" }}>{r.m}</td>
              <td style={{ ...td, fontFamily: FONT_MONO, fontSize: 11 }}>{r.p}</td>
              <td style={{ ...td, fontSize: 11 }}>{r.d}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Win>
    <Win title="Example: Create + Evaluate">
      <Code>{`# Create session\ncurl -X POST http://localhost:8080/api/sessions\n\n# Evaluate code\ncurl -X POST http://localhost:8080/api/sessions/sess_7f3a2b/evaluate \\\n  -H "Content-Type: application/json" \\\n  -d '{"source": "const x = 1; x"}'`}</Code>
    </Win>
  </div>);
}

/* ═══════════════════════════════════════════
   APP
   ═══════════════════════════════════════════ */

export default function REPLEssay() {
  return (
    <div style={{
      background: "#fff", color: "#000", fontFamily: FONT, fontSize: 13,
      lineHeight: 1.55, maxWidth: 780, margin: "0 auto", padding: "24px 20px 80px",
    }}>
      <div style={{ textAlign: "center", marginBottom: 4 }}>
        <div style={{ fontSize: 26, fontWeight: 700, letterSpacing: -0.5 }}>▪ The REPL Essay ▪</div>
        <div style={{ fontSize: 11, color: "#777", fontStyle: "italic", marginTop: 2, marginBottom: 8 }}>
          An interactive guide to the session model, evaluation pipeline, and persistence layer
        </div>
        <div style={{ height: 3, background: "#000", margin: "0 auto", maxWidth: 400 }} />
        <div style={{ height: 1, background: "#000", margin: "3px auto 0", maxWidth: 400 }} />
      </div>
      <div style={{ fontSize: 10.5, color: "#999", textAlign: "center", margin: "10px 0 20px", fontFamily: FONT_MONO }}>
        GOJA-043 · go-go-goja · {new Date().toLocaleDateString("en-US", { year: "numeric", month: "long", day: "numeric" })}
      </div>
      <Callout>
        <strong>About this essay.</strong> Every section teaches through real feedback. When you do something,
        the page reveals multiple synchronized views of the same event: the friendly explanation, the compact summary,
        and the exact backend payload. This is mock data — the real version will talk to <span style={{ fontFamily: FONT_MONO, fontSize: 11 }}>goja-repl serve</span>.
      </Callout>
      <S1 /><Dither />
      <S2 /><Dither />
      <S3 /><Dither />
      <S4 /><Dither />
      <S5 /><Dither />
      <S6 /><Dither />
      <S7 /><Dither />
      <S8 /><Dither />
      <S9 />
      <Dither />
      <div style={{ textAlign: "center", fontSize: 10.5, color: "#aaa", marginTop: 8 }}>
        ◆ End of essay · Designed for GOJA-043 · Mock revision ◆
      </div>
    </div>
  );
}
