const form = document.querySelector("#capture-form");
const statusEl = document.querySelector("#status");
const sessionStatusEl = document.querySelector("#session-status");
const itemsEl = document.querySelector("#items");
const refreshEl = document.querySelector("#refresh");
const logoutEl = document.querySelector("#logout");
let csrfToken = "";

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(form).entries());
  await captureItem(data);
});

refreshEl.addEventListener("click", () => loadItems());
logoutEl.addEventListener("click", () => logout());

loadSession();
loadItems();

async function loadSession() {
  try {
    const res = await fetch("/auth/session");
    const body = await res.json();
    if (!res.ok || !body.authenticated) {
      sessionStatusEl.textContent = "Not logged in.";
      sessionStatusEl.classList.add("error");
      return;
    }
    csrfToken = body.csrfToken || "";
    const actor = body.actor || {};
    sessionStatusEl.textContent = `Logged in as ${actor.claims?.email || actor.id || "session user"}.`;
    sessionStatusEl.classList.remove("error");
  } catch (err) {
    sessionStatusEl.textContent = String(err.message || err);
    sessionStatusEl.classList.add("error");
  }
}

async function logout() {
  try {
    await fetch("/auth/logout", { method: "POST", headers: csrfHeaders() });
  } finally {
    window.location.href = "/";
  }
}

async function captureItem(data) {
  setStatus("Capturing…");
  try {
    const res = await fetch("/api/capture", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...csrfHeaders() },
      body: JSON.stringify({ ...data, source: "browser-ui" })
    });
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || `HTTP ${res.status}`);
    form.reset();
    setStatus("Captured.");
    await loadItems();
  } catch (err) {
    setStatus(String(err.message || err), true);
  }
}

async function loadItems() {
  try {
    const res = await fetch("/api/inbox");
    const body = await res.json();
    if (!res.ok) throw new Error(body.error || `HTTP ${res.status}`);
    renderItems(body.items || []);
  } catch (err) {
    itemsEl.innerHTML = `<li class="empty">${escapeHtml(String(err.message || err))}</li>`;
  }
}

function renderItems(items) {
  if (!items.length) {
    itemsEl.innerHTML = '<li class="empty">No captures yet.</li>';
    return;
  }
  itemsEl.innerHTML = items.map((item) => `
    <li class="item">
      <div>
        <div class="item-title">${escapeHtml(item.title)}</div>
        <div class="item-url">${escapeHtml(item.url)}</div>
        ${item.note ? `<div class="item-note">${escapeHtml(item.note)}</div>` : ""}
      </div>
      <div class="item-meta">${escapeHtml(item.source || "api")}</div>
    </li>
  `).join("");
}

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function csrfHeaders() {
  return csrfToken ? { "X-CSRF-Token": csrfToken } : {};
}

function escapeHtml(value) {
  return String(value || "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}
