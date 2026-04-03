const state = {
  sessionId: null,
  lastResponse: null,
};

const editor = document.getElementById('editor');
const runBtn = document.getElementById('runBtn');
const newSessionBtn = document.getElementById('newSessionBtn');

newSessionBtn.addEventListener('click', () => createSession());
runBtn.addEventListener('click', () => evaluateCell());
editor.addEventListener('keydown', (event) => {
  if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
    event.preventDefault();
    evaluateCell();
  }
});

window.addEventListener('load', () => {
  createSession();
});

async function createSession() {
  try {
    const response = await fetch('/api/sessions', { method: 'POST' });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || 'failed to create session');
    }
    state.sessionId = payload.session.id;
    state.lastResponse = { session: payload.session, cell: null };
    render();
  } catch (error) {
    renderFailure(error);
  }
}

async function evaluateCell() {
  if (!state.sessionId) {
    await createSession();
    if (!state.sessionId) {
      return;
    }
  }
  runBtn.disabled = true;
  try {
    const response = await fetch(`/api/sessions/${encodeURIComponent(state.sessionId)}/evaluate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source: editor.value }),
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || 'evaluation failed');
    }
    state.lastResponse = payload;
    render();
  } catch (error) {
    renderFailure(error);
  } finally {
    runBtn.disabled = false;
  }
}

function render() {
  renderSessionMeta(state.lastResponse?.session || null);
  renderBindings(state.lastResponse?.session?.bindings || []);
  renderHistory(state.lastResponse?.session?.history || []);
  renderCell(state.lastResponse?.cell || null);
}

function renderSessionMeta(session) {
  const target = document.getElementById('sessionMeta');
  if (!session) {
    target.innerHTML = '';
    return;
  }
  target.innerHTML = [
    metaRow('Session ID', session.id),
    metaRow('Created', formatDate(session.createdAt)),
    metaRow('Cells', session.cellCount),
    metaRow('Bindings', session.bindingCount),
    metaRow('Globals', session.currentGlobals ? session.currentGlobals.length : 0),
  ].join('');
}

function renderBindings(bindings) {
  const tbody = document.querySelector('#bindingsTable tbody');
  tbody.innerHTML = '';
  if (!bindings.length) {
    tbody.innerHTML = '<tr><td colspan="6" class="text-body-secondary p-3">No bindings yet.</td></tr>';
    return;
  }
  bindings.forEach((binding) => {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td><code>${escapeHtml(binding.name)}</code></td>
      <td>${escapeHtml(binding.kind)}</td>
      <td class="binding-preview">${escapeHtml(binding.runtime?.preview || '')}</td>
      <td>${escapeHtml(binding.origin || '')}</td>
      <td>${binding.declaredInCell || ''}</td>
      <td>${binding.lastUpdatedCell || ''}</td>
    `;
    tbody.appendChild(tr);
  });
}

function renderHistory(history) {
  const target = document.getElementById('history');
  target.innerHTML = '';
  if (!history.length) {
    target.innerHTML = '<div class="list-group-item text-body-secondary">No cells executed yet.</div>';
    return;
  }
  [...history].reverse().forEach((entry) => {
    const item = document.createElement('div');
    item.className = 'list-group-item';
    item.innerHTML = `
      <div class="d-flex justify-content-between align-items-center gap-3">
        <div>
          <div><strong>Cell ${entry.cellId}</strong> <span class="badge text-bg-secondary">${escapeHtml(entry.status)}</span></div>
          <div class="small text-body-secondary">${escapeHtml(entry.sourcePreview || '')}</div>
        </div>
        <div class="small text-end text-body-secondary">
          <div>${formatDate(entry.createdAt)}</div>
          <div>${escapeHtml(entry.resultPreview || '')}</div>
        </div>
      </div>
    `;
    target.appendChild(item);
  });
}

function renderCell(cell) {
  const executionSummary = document.getElementById('executionSummary');
  const consoleOutput = document.getElementById('consoleOutput');
  const provenanceList = document.getElementById('provenanceList');
  const staticJson = document.getElementById('staticJson');
  const rewriteSource = document.getElementById('rewriteSource');
  const runtimeJson = document.getElementById('runtimeJson');
  const rawJson = document.getElementById('rawJson');

  if (!cell) {
    executionSummary.innerHTML = '<p class="text-body-secondary mb-0">Run a cell to populate this panel.</p>';
    consoleOutput.textContent = '';
    provenanceList.innerHTML = '';
    staticJson.textContent = '';
    rewriteSource.textContent = '';
    runtimeJson.textContent = '';
    rawJson.textContent = '';
    return;
  }

  const statusClass = statusBadgeClass(cell.execution?.status);
  executionSummary.innerHTML = `
    <div class="d-flex flex-wrap gap-2 align-items-center mb-2">
      <span class="badge ${statusClass}">${escapeHtml(cell.execution?.status || '')}</span>
      <span class="text-body-secondary small">Cell ${cell.id}</span>
      <span class="text-body-secondary small">${cell.execution?.durationMs || 0} ms</span>
      <span class="text-body-secondary small">awaited: ${cell.execution?.awaited ? 'yes' : 'no'}</span>
    </div>
    <div class="mb-2"><strong>Result:</strong> <code>${escapeHtml(cell.execution?.result || '')}</code></div>
    ${cell.execution?.error ? `<div class="alert alert-danger py-2 mb-0"><strong>Error:</strong> ${escapeHtml(cell.execution.error)}</div>` : ''}
  `;

  if (!cell.execution?.console?.length) {
    consoleOutput.innerHTML = '<span class="text-body-secondary">No console output.</span>';
  } else {
    consoleOutput.innerHTML = cell.execution.console.map((item) => {
      const kind = escapeHtml(item.kind || 'log');
      return `<div class="${kind}">[${kind}] ${escapeHtml(item.message || '')}</div>`;
    }).join('');
  }

  provenanceList.innerHTML = (cell.provenance || []).map((item) => {
    const notes = (item.notes || []).map((note) => `<li>${escapeHtml(note)}</li>`).join('');
    return `<li><strong>${escapeHtml(item.section || '')}</strong>: ${escapeHtml(item.source || '')}${notes ? `<ul>${notes}</ul>` : ''}</li>`;
  }).join('');

  staticJson.textContent = formatJson(cell.static || {});
  rewriteSource.textContent = cell.rewrite?.transformedSource || formatJson(cell.rewrite || {});
  runtimeJson.textContent = formatJson(cell.runtime || {});
  rawJson.textContent = formatJson(cell);
}

function renderFailure(error) {
  const executionSummary = document.getElementById('executionSummary');
  executionSummary.innerHTML = `<div class="alert alert-danger py-2 mb-0">${escapeHtml(error.message || String(error))}</div>`;
}

function metaRow(label, value) {
  return `<dt class="col-4">${escapeHtml(String(label))}</dt><dd class="col-8">${escapeHtml(String(value ?? ''))}</dd>`;
}

function formatJson(value) {
  return JSON.stringify(value, null, 2);
}

function statusBadgeClass(status) {
  switch (status) {
    case 'ok':
      return 'text-bg-success';
    case 'parse-error':
    case 'runtime-error':
    case 'helper-error':
      return 'text-bg-danger';
    default:
      return 'text-bg-secondary';
  }
}

function formatDate(value) {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
