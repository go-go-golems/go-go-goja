package server

// uiHTML is the embedded single-page doc browser application.
// It uses vanilla JS + marked.js for Markdown rendering, with a
// Mathematica-inspired minimal Swiss design.
const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>JSDocEx — Documentation Browser</title>
<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/lib/core.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/languages/javascript.min.js"></script>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/highlight.js@11.9.0/styles/github.min.css">
<style>
  /* ---- Reset & Base ---- */
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root {
    --bg:        #fafafa;
    --surface:   #ffffff;
    --border:    #e0e0e0;
    --accent:    #c41230;   /* Mathematica red */
    --accent2:   #1a1a8c;   /* deep blue */
    --text:      #1a1a1a;
    --muted:     #666666;
    --tag-bg:    #f0f0f0;
    --tag-text:  #444444;
    --code-bg:   #f6f8fa;
    --sidebar-w: 260px;
    --header-h:  52px;
    --font-sans: 'Helvetica Neue', Helvetica, Arial, sans-serif;
    --font-mono: 'SF Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace;
  }
  html, body { height: 100%; font-family: var(--font-sans); font-size: 14px; color: var(--text); background: var(--bg); }

  /* ---- Layout ---- */
  #app { display: flex; flex-direction: column; height: 100vh; }

  header {
    height: var(--header-h);
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    padding: 0 16px;
    gap: 16px;
    position: sticky;
    top: 0;
    z-index: 100;
    flex-shrink: 0;
  }
  .logo {
    font-size: 16px;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: var(--accent);
    white-space: nowrap;
    cursor: pointer;
  }
  .logo span { color: var(--text); }
  #search-input {
    flex: 1;
    max-width: 400px;
    height: 32px;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 10px;
    font-size: 13px;
    font-family: var(--font-sans);
    outline: none;
    background: var(--bg);
  }
  #search-input:focus { border-color: var(--accent2); }
  #reload-badge {
    display: none;
    background: var(--accent);
    color: white;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 10px;
    cursor: pointer;
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.6} }

  .main-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  /* ---- Sidebar ---- */
  #sidebar {
    width: var(--sidebar-w);
    background: var(--surface);
    border-right: 1px solid var(--border);
    overflow-y: auto;
    flex-shrink: 0;
  }
  .sidebar-section { padding: 8px 0; border-bottom: 1px solid var(--border); }
  .sidebar-section-title {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--muted);
    padding: 6px 14px 4px;
  }
  .sidebar-item {
    display: block;
    padding: 5px 14px;
    font-size: 13px;
    cursor: pointer;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    border-left: 2px solid transparent;
    color: var(--text);
    text-decoration: none;
  }
  .sidebar-item:hover { background: var(--bg); }
  .sidebar-item.active { border-left-color: var(--accent); color: var(--accent); font-weight: 600; }
  .sidebar-item .item-type {
    font-size: 10px;
    color: var(--muted);
    margin-left: 4px;
  }

  /* ---- Content ---- */
  #content {
    flex: 1;
    overflow-y: auto;
    padding: 32px 40px;
    max-width: 900px;
  }

  /* ---- Cards ---- */
  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    margin-bottom: 20px;
    overflow: hidden;
  }
  .card-header {
    padding: 14px 20px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: baseline;
    gap: 10px;
    background: #fcfcfc;
  }
  .card-title {
    font-size: 16px;
    font-weight: 700;
    font-family: var(--font-mono);
    color: var(--accent2);
  }
  .card-kind {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--muted);
  }
  .card-body { padding: 16px 20px; }

  /* ---- Symbol page ---- */
  .symbol-summary {
    font-size: 15px;
    line-height: 1.5;
    margin-bottom: 16px;
    color: var(--text);
  }
  .params-table { width: 100%; border-collapse: collapse; margin: 12px 0; font-size: 13px; }
  .params-table th {
    text-align: left;
    padding: 6px 10px;
    background: var(--code-bg);
    border: 1px solid var(--border);
    font-weight: 600;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--muted);
  }
  .params-table td {
    padding: 6px 10px;
    border: 1px solid var(--border);
    vertical-align: top;
  }
  .param-name { font-family: var(--font-mono); color: var(--accent2); font-size: 12px; }
  .param-type { font-family: var(--font-mono); color: #888; font-size: 12px; }

  /* ---- Tags ---- */
  .tag-list { display: flex; flex-wrap: wrap; gap: 5px; margin: 8px 0; }
  .tag {
    background: var(--tag-bg);
    color: var(--tag-text);
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 3px;
    font-family: var(--font-mono);
    cursor: pointer;
  }
  .tag:hover { background: var(--border); }
  .tag.concept { background: #e8f0fe; color: var(--accent2); }
  .tag.related { background: #fce8e6; color: var(--accent); }

  /* ---- Prose (Markdown) ---- */
  .prose { font-size: 14px; line-height: 1.7; color: var(--text); }
  .prose h1 { font-size: 20px; margin: 0 0 12px; }
  .prose h2 { font-size: 16px; margin: 20px 0 8px; border-bottom: 1px solid var(--border); padding-bottom: 4px; }
  .prose h3 { font-size: 14px; margin: 16px 0 6px; }
  .prose p  { margin: 0 0 10px; }
  .prose ul, .prose ol { margin: 0 0 10px 20px; }
  .prose li { margin: 3px 0; }
  .prose code { font-family: var(--font-mono); font-size: 12px; background: var(--code-bg); padding: 1px 4px; border-radius: 3px; }
  .prose pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px; padding: 12px 14px; overflow-x: auto; margin: 10px 0; }
  .prose pre code { background: none; padding: 0; font-size: 12px; }
  .prose strong { font-weight: 700; }
  .prose em { font-style: italic; }
  .prose blockquote { border-left: 3px solid var(--border); padding-left: 12px; color: var(--muted); margin: 10px 0; }

  /* ---- Example block ---- */
  .example-card { border-left: 3px solid var(--accent); }
  .example-card .card-header { background: #fff8f8; }
  .example-meta { font-size: 12px; color: var(--muted); margin-bottom: 8px; }

  /* ---- Package page ---- */
  .package-header { margin-bottom: 24px; }
  .package-title { font-size: 24px; font-weight: 700; margin-bottom: 4px; }
  .package-name { font-family: var(--font-mono); font-size: 13px; color: var(--muted); }
  .package-description { margin: 12px 0; font-size: 15px; line-height: 1.5; }

  .symbol-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 10px;
    margin: 12px 0;
  }
  .symbol-chip {
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 8px 12px;
    cursor: pointer;
    background: var(--surface);
    transition: border-color 0.15s;
  }
  .symbol-chip:hover { border-color: var(--accent2); }
  .symbol-chip-name { font-family: var(--font-mono); font-size: 13px; font-weight: 600; color: var(--accent2); }
  .symbol-chip-summary { font-size: 11px; color: var(--muted); margin-top: 3px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

  /* ---- Home / overview ---- */
  .home-title { font-size: 22px; font-weight: 700; margin-bottom: 6px; }
  .home-subtitle { color: var(--muted); margin-bottom: 28px; }
  .stats-row { display: flex; gap: 20px; margin-bottom: 28px; }
  .stat-box { border: 1px solid var(--border); border-radius: 6px; padding: 14px 20px; background: var(--surface); text-align: center; }
  .stat-num { font-size: 28px; font-weight: 700; color: var(--accent); }
  .stat-label { font-size: 11px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }

  /* ---- Search results ---- */
  .search-section-title { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.07em; color: var(--muted); margin: 20px 0 8px; }
  .search-result-item { padding: 10px 14px; border: 1px solid var(--border); border-radius: 4px; margin-bottom: 6px; cursor: pointer; background: var(--surface); }
  .search-result-item:hover { border-color: var(--accent2); }
  .search-result-name { font-family: var(--font-mono); font-size: 13px; font-weight: 600; color: var(--accent2); }
  .search-result-summary { font-size: 12px; color: var(--muted); margin-top: 2px; }

  /* ---- Mobile ---- */
  #sidebar-toggle { display: none; background: none; border: none; cursor: pointer; font-size: 20px; padding: 4px; }
  @media (max-width: 700px) {
    #sidebar-toggle { display: block; }
    #sidebar { position: fixed; top: var(--header-h); left: 0; bottom: 0; z-index: 200; transform: translateX(-100%); transition: transform 0.2s; box-shadow: 2px 0 8px rgba(0,0,0,0.1); }
    #sidebar.open { transform: translateX(0); }
    #content { padding: 20px 16px; }
  }

  /* ---- Live reload indicator ---- */
  #live-dot {
    width: 8px; height: 8px; border-radius: 50%;
    background: #ccc;
    flex-shrink: 0;
    title: "Live reload";
  }
  #live-dot.connected { background: #22c55e; }
</style>
</head>
<body>
<div id="app">
  <header>
    <button id="sidebar-toggle" onclick="toggleSidebar()">☰</button>
    <div class="logo" onclick="navigate('home')">JSDoc<span>Ex</span></div>
    <input id="search-input" type="search" placeholder="Search symbols, concepts, examples…" oninput="onSearch(this.value)" autocomplete="off">
    <div id="reload-badge" onclick="location.reload()">● Reload</div>
    <div id="live-dot" title="Live reload disconnected"></div>
  </header>

  <div class="main-layout">
    <nav id="sidebar">
      <div id="sidebar-content"></div>
    </nav>
    <main id="content"></main>
  </div>
</div>

<script>
// ---- State ----
let store = null;
let currentView = null;
let searchTimeout = null;

// ---- Init ----
async function init() {
  await loadStore();
  renderSidebar();
  navigate('home');
  connectSSE();
}

async function loadStore() {
  const res = await fetch('/api/store');
  store = await res.json();
}

// ---- SSE ----
function connectSSE() {
  const es = new EventSource('/events');
  const dot = document.getElementById('live-dot');
  es.onopen = () => { dot.className = 'connected'; dot.title = 'Live reload connected'; };
  es.onmessage = (e) => {
    if (e.data === 'reload') {
      document.getElementById('reload-badge').style.display = 'inline-block';
      loadStore().then(() => {
        renderSidebar();
        document.getElementById('reload-badge').style.display = 'none';
        // Re-render current view
        if (currentView) navigate(currentView.type, currentView.id);
      });
    }
  };
  es.onerror = () => { dot.className = ''; dot.title = 'Live reload disconnected'; };
}

// ---- Sidebar ----
function renderSidebar() {
  if (!store) return;
  const el = document.getElementById('sidebar-content');
  let html = '';

  // Packages
  const pkgs = Object.values(store.by_package || {});
  if (pkgs.length) {
    html += '<div class="sidebar-section"><div class="sidebar-section-title">Packages</div>';
    pkgs.sort((a,b) => a.name.localeCompare(b.name)).forEach(pkg => {
      const active = currentView?.type === 'package' && currentView?.id === pkg.name ? ' active' : '';
      html += '<a class="sidebar-item' + active + '" onclick="navigate(\'package\',\''+esc(pkg.name)+'\')">' + esc(pkg.title || pkg.name) + '</a>';
    });
    html += '</div>';
  }

  // Symbols grouped by package
  const files = store.files || [];
  files.forEach(fd => {
    if (!fd.symbols || !fd.symbols.length) return;
    const pkgName = fd.package?.name || fd.file_path;
    html += '<div class="sidebar-section"><div class="sidebar-section-title">' + esc(pkgName) + '</div>';
    fd.symbols.forEach(sym => {
      const active = currentView?.type === 'symbol' && currentView?.id === sym.name ? ' active' : '';
      html += '<a class="sidebar-item' + active + '" onclick="navigate(\'symbol\',\''+esc(sym.name)+'\')">' + esc(sym.name) + '</a>';
    });
    html += '</div>';
  });

  el.innerHTML = html;
}

// ---- Navigation ----
function navigate(type, id) {
  currentView = { type, id };
  closeSidebar();
  switch(type) {
    case 'home':    renderHome(); break;
    case 'package': renderPackage(id); break;
    case 'symbol':  renderSymbol(id); break;
    case 'example': renderExample(id); break;
    case 'concept': renderConcept(id); break;
  }
  renderSidebar(); // update active states
}

// ---- Home ----
function renderHome() {
  const files = store.files || [];
  const symCount = Object.keys(store.by_symbol || {}).length;
  const exCount  = Object.keys(store.by_example || {}).length;
  const pkgCount = Object.keys(store.by_package || {}).length;
  const conceptCount = Object.keys(store.by_concept || {}).length;

  let html = '<div class="home-title">Documentation Browser</div>';
  html += '<div class="home-subtitle">Mathematica-style JS documentation extracted via tree-sitter</div>';

  html += '<div class="stats-row">';
  html += stat(pkgCount, 'Packages');
  html += stat(symCount, 'Symbols');
  html += stat(exCount,  'Examples');
  html += stat(conceptCount, 'Concepts');
  html += '</div>';

  // Package cards
  const pkgs = Object.values(store.by_package || {});
  pkgs.sort((a,b) => (a.category||'').localeCompare(b.category||'')).forEach(pkg => {
    html += '<div class="card" style="cursor:pointer" onclick="navigate(\'package\',\''+esc(pkg.name)+'\')">';
    html += '<div class="card-header"><span class="card-title">' + esc(pkg.title || pkg.name) + '</span>';
    html += '<span class="card-kind">' + esc(pkg.category || '') + '</span></div>';
    html += '<div class="card-body">';
    if (pkg.description) html += '<div style="font-size:13px;color:var(--muted);margin-bottom:8px">' + esc(pkg.description) + '</div>';
    html += '<div style="font-family:var(--font-mono);font-size:11px;color:var(--muted)">' + esc(pkg.name) + ' v' + esc(pkg.version||'') + '</div>';
    html += '</div></div>';
  });

  setContent(html);
}

function stat(n, label) {
  return '<div class="stat-box"><div class="stat-num">'+n+'</div><div class="stat-label">'+label+'</div></div>';
}

// ---- Package page ----
async function renderPackage(name) {
  const res = await fetch('/api/package/' + encodeURIComponent(name));
  if (!res.ok) { setContent('<p>Package not found: ' + esc(name) + '</p>'); return; }
  const pkg = await res.json();

  let html = '<div class="package-header">';
  html += '<div class="package-title">' + esc(pkg.title || pkg.name) + '</div>';
  html += '<div class="package-name">' + esc(pkg.name) + (pkg.version ? ' &nbsp;v' + esc(pkg.version) : '') + '</div>';
  if (pkg.category) html += '<div style="font-size:12px;color:var(--muted);margin-top:4px">' + esc(pkg.category) + '</div>';
  if (pkg.description) html += '<div class="package-description">' + esc(pkg.description) + '</div>';
  if (pkg.see_also?.length) {
    html += '<div style="font-size:12px;color:var(--muted);margin-top:4px">See also: ';
    html += pkg.see_also.map(s => '<a style="color:var(--accent2);cursor:pointer" onclick="navigate(\'package\',\''+esc(s)+'\')">' + esc(s) + '</a>').join(', ');
    html += '</div>';
  }
  html += '</div>';

  // Long-form prose
  if (pkg.prose) {
    html += '<div class="card"><div class="card-body"><div class="prose">' + renderMarkdown(pkg.prose) + '</div></div></div>';
  }

  // Symbols in this package
  const fileDoc = (store.files || []).find(f => f.package?.name === name);
  if (fileDoc?.symbols?.length) {
    html += '<div style="font-size:13px;font-weight:700;margin:20px 0 10px">Symbols</div>';
    html += '<div class="symbol-grid">';
    fileDoc.symbols.forEach(sym => {
      html += '<div class="symbol-chip" onclick="navigate(\'symbol\',\''+esc(sym.name)+'\')">';
      html += '<div class="symbol-chip-name">' + esc(sym.name) + '</div>';
      if (sym.summary) html += '<div class="symbol-chip-summary">' + esc(sym.summary) + '</div>';
      html += '</div>';
    });
    html += '</div>';
  }

  // Examples in this package
  if (fileDoc?.examples?.length) {
    html += '<div style="font-size:13px;font-weight:700;margin:20px 0 10px">Examples</div>';
    fileDoc.examples.forEach(ex => {
      html += renderExampleCard(ex);
    });
  }

  setContent(html);
}

// ---- Symbol page ----
async function renderSymbol(name) {
  const res = await fetch('/api/symbol/' + encodeURIComponent(name));
  if (!res.ok) { setContent('<p>Symbol not found: ' + esc(name) + '</p>'); return; }
  const data = await res.json();
  const sym = data;

  let html = '<div class="card">';
  html += '<div class="card-header"><span class="card-title">' + esc(sym.name) + '</span>';
  html += '<span class="card-kind">Symbol</span>';
  if (sym.source_file) html += '<span style="font-size:11px;color:var(--muted);margin-left:auto">' + esc(sym.source_file.split('/').pop()) + ':' + (sym.line||'') + '</span>';
  html += '</div>';
  html += '<div class="card-body">';

  if (sym.summary) html += '<div class="symbol-summary">' + esc(sym.summary) + '</div>';

  // Params table
  if (sym.params?.length) {
    html += '<table class="params-table"><thead><tr><th>Parameter</th><th>Type</th><th>Description</th></tr></thead><tbody>';
    sym.params.forEach(p => {
      html += '<tr><td><span class="param-name">' + esc(p.name) + '</span></td>';
      html += '<td><span class="param-type">' + esc(p.type||'') + '</span></td>';
      html += '<td>' + esc(p.description||'') + '</td></tr>';
    });
    html += '</tbody></table>';
  }

  // Returns
  if (sym.returns?.type) {
    html += '<div style="font-size:12px;margin:8px 0"><strong>Returns:</strong> <span class="param-type">' + esc(sym.returns.type) + '</span>';
    if (sym.returns.description) html += ' — ' + esc(sym.returns.description);
    html += '</div>';
  }

  // Tags & concepts
  if (sym.tags?.length || sym.concepts?.length) {
    html += '<div class="tag-list">';
    (sym.concepts||[]).forEach(c => html += '<span class="tag concept" onclick="navigate(\'concept\',\''+esc(c)+'\')">' + esc(c) + '</span>');
    (sym.tags||[]).forEach(t => html += '<span class="tag">' + esc(t) + '</span>');
    html += '</div>';
  }

  // Related
  if (sym.related?.length) {
    html += '<div style="font-size:12px;margin-top:10px"><strong>Related:</strong> ';
    html += sym.related.map(r => '<a style="color:var(--accent2);cursor:pointer;font-family:var(--font-mono);font-size:12px" onclick="navigate(\'symbol\',\''+esc(r)+'\')">' + esc(r) + '</a>').join(', ');
    html += '</div>';
  }

  if (sym.docpage) {
    html += '<div style="font-size:12px;margin-top:8px;color:var(--muted)">📄 ' + esc(sym.docpage) + '</div>';
  }

  html += '</div></div>';

  // Long-form prose
  if (sym.prose) {
    html += '<div class="card"><div class="card-header"><span class="card-kind">Documentation</span></div>';
    html += '<div class="card-body"><div class="prose">' + renderMarkdown(sym.prose) + '</div></div></div>';
  }

  // Examples
  if (data.examples?.length) {
    html += '<div style="font-size:13px;font-weight:700;margin:20px 0 10px">Examples</div>';
    data.examples.forEach(ex => { html += renderExampleCard(ex); });
  }

  setContent(html);
}

// ---- Example page ----
async function renderExample(id) {
  const res = await fetch('/api/example/' + encodeURIComponent(id));
  if (!res.ok) { setContent('<p>Example not found: ' + esc(id) + '</p>'); return; }
  const ex = await res.json();
  setContent(renderExampleCard(ex, true));
}

function renderExampleCard(ex, standalone) {
  let html = '<div class="card example-card">';
  html += '<div class="card-header"><span class="card-title">' + esc(ex.title || ex.id) + '</span>';
  html += '<span class="card-kind">Example</span></div>';
  html += '<div class="card-body">';
  html += '<div class="example-meta">ID: <code>' + esc(ex.id) + '</code>';
  if (ex.source_file) html += ' &nbsp;·&nbsp; ' + esc(ex.source_file.split('/').pop()) + ':' + (ex.line||'');
  html += '</div>';

  if (ex.symbols?.length) {
    html += '<div style="font-size:12px;margin-bottom:8px"><strong>Symbols:</strong> ';
    html += ex.symbols.map(s => '<a style="color:var(--accent2);cursor:pointer;font-family:var(--font-mono);font-size:12px" onclick="navigate(\'symbol\',\''+esc(s)+'\')">' + esc(s) + '</a>').join(', ');
    html += '</div>';
  }

  if (ex.tags?.length || ex.concepts?.length) {
    html += '<div class="tag-list">';
    (ex.concepts||[]).forEach(c => html += '<span class="tag concept" onclick="navigate(\'concept\',\''+esc(c)+'\')">' + esc(c) + '</span>');
    (ex.tags||[]).forEach(t => html += '<span class="tag">' + esc(t) + '</span>');
    html += '</div>';
  }

  if (ex.docpage) html += '<div style="font-size:12px;margin-top:8px;color:var(--muted)">📄 ' + esc(ex.docpage) + '</div>';

  html += '</div></div>';
  return html;
}

// ---- Concept page ----
function renderConcept(concept) {
  const symbolNames = (store.by_concept || {})[concept] || [];
  let html = '<div class="home-title">Concept: ' + esc(concept) + '</div>';
  html += '<div class="home-subtitle" style="margin-bottom:20px">All symbols and examples related to this concept</div>';

  if (symbolNames.length) {
    html += '<div style="font-size:13px;font-weight:700;margin-bottom:10px">Symbols</div>';
    html += '<div class="symbol-grid">';
    symbolNames.forEach(name => {
      const sym = (store.by_symbol || {})[name];
      if (!sym) return;
      html += '<div class="symbol-chip" onclick="navigate(\'symbol\',\''+esc(name)+'\')">';
      html += '<div class="symbol-chip-name">' + esc(name) + '</div>';
      if (sym.summary) html += '<div class="symbol-chip-summary">' + esc(sym.summary) + '</div>';
      html += '</div>';
    });
    html += '</div>';
  }

  // Examples for this concept
  const exs = Object.values(store.by_example || {}).filter(e => (e.concepts||[]).includes(concept));
  if (exs.length) {
    html += '<div style="font-size:13px;font-weight:700;margin:20px 0 10px">Examples</div>';
    exs.forEach(ex => { html += renderExampleCard(ex); });
  }

  setContent(html);
}

// ---- Search ----
function onSearch(q) {
  clearTimeout(searchTimeout);
  if (!q.trim()) { navigate('home'); return; }
  searchTimeout = setTimeout(() => doSearch(q), 200);
}

async function doSearch(q) {
  const res = await fetch('/api/search?q=' + encodeURIComponent(q));
  const data = await res.json();
  currentView = { type: 'search', id: q };

  let html = '<div class="home-title">Search: <em>' + esc(q) + '</em></div>';

  if (data.packages?.length) {
    html += '<div class="search-section-title">Packages</div>';
    data.packages.forEach(pkg => {
      html += '<div class="search-result-item" onclick="navigate(\'package\',\''+esc(pkg.name)+'\')">';
      html += '<div class="search-result-name">' + esc(pkg.title || pkg.name) + '</div>';
      if (pkg.description) html += '<div class="search-result-summary">' + esc(pkg.description) + '</div>';
      html += '</div>';
    });
  }

  if (data.symbols?.length) {
    html += '<div class="search-section-title">Symbols</div>';
    data.symbols.forEach(sym => {
      html += '<div class="search-result-item" onclick="navigate(\'symbol\',\''+esc(sym.name)+'\')">';
      html += '<div class="search-result-name">' + esc(sym.name) + '</div>';
      if (sym.summary) html += '<div class="search-result-summary">' + esc(sym.summary) + '</div>';
      html += '</div>';
    });
  }

  if (data.examples?.length) {
    html += '<div class="search-section-title">Examples</div>';
    data.examples.forEach(ex => {
      html += '<div class="search-result-item" onclick="navigate(\'example\',\''+esc(ex.id)+'\')">';
      html += '<div class="search-result-name">' + esc(ex.title || ex.id) + '</div>';
      html += '<div class="search-result-summary">Example · ' + (ex.symbols||[]).join(', ') + '</div>';
      html += '</div>';
    });
  }

  if (!data.packages?.length && !data.symbols?.length && !data.examples?.length) {
    html += '<p style="color:var(--muted);margin-top:20px">No results found.</p>';
  }

  setContent(html);
}

// ---- Utilities ----
function setContent(html) {
  document.getElementById('content').innerHTML = html;
  // Apply syntax highlighting to any code blocks
  document.querySelectorAll('pre code').forEach(el => {
    hljs.highlightElement(el);
  });
}

function renderMarkdown(text) {
  if (!text) return '';
  return marked.parse(text);
}

function esc(s) {
  if (s == null) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function toggleSidebar() {
  document.getElementById('sidebar').classList.toggle('open');
}
function closeSidebar() {
  document.getElementById('sidebar').classList.remove('open');
}

// ---- Bootstrap ----
init();
</script>
</body>
</html>`
