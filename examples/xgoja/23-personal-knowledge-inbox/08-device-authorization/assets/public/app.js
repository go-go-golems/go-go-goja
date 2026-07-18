const form = document.querySelector("#capture-form");
const statusEl = document.querySelector("#status");
const sessionStatusEl = document.querySelector("#session-status");
const deviceForm = document.querySelector("#device-form");
const deviceStatusEl = document.querySelector("#device-status");
const itemsEl = document.querySelector("#items");
const refreshEl = document.querySelector("#refresh");
const loginEl = document.querySelector("#login");
const logoutEl = document.querySelector("#logout");
const inboxOwnerEl = document.querySelector("#inbox-owner");
let csrfToken = "";
let authenticated = false;

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!authenticated) {
    setStatus("Log in before capturing from the browser UI.", true);
    return;
  }
  const data = Object.fromEntries(new FormData(form).entries());
  await captureItem(data);
});

refreshEl.addEventListener("click", () => loadItems());
logoutEl.addEventListener("click", () => logout());
deviceForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!authenticated) {
    setDeviceStatus("Log in before approving a device.", true);
    return;
  }
  const data = Object.fromEntries(new FormData(deviceForm).entries());
  await approveDevice(data.userCode);
});

loadSession();

async function loadSession() {
  try {
    const res = await fetch("/auth/session");
    if (res.status === 401) {
      setLoggedOut();
      return;
    }
    const body = await readResponse(res);
    if (!res.ok || !body.userId) {
      setLoggedOut();
      return;
    }
    csrfToken = body.csrfToken || "";
    authenticated = true;
    sessionStatusEl.textContent = `Logged in as ${displayName(body)}.`;
    inboxOwnerEl.textContent = `${displayName(body)}'s inbox`;
    sessionStatusEl.classList.remove("error");
    loginEl.classList.add("hidden");
    logoutEl.classList.remove("hidden");
    setFormEnabled(true);
    setDeviceFormEnabled(true);
    await loadItems();
  } catch (err) {
    sessionStatusEl.textContent = String(err.message || err);
    sessionStatusEl.classList.add("error");
    setFormEnabled(false);
  }
}

function setLoggedOut() {
  authenticated = false;
  csrfToken = "";
  sessionStatusEl.textContent = "Not logged in.";
  sessionStatusEl.classList.add("error");
  loginEl.classList.remove("hidden");
  logoutEl.classList.add("hidden");
  setFormEnabled(false);
  setDeviceFormEnabled(false);
  inboxOwnerEl.textContent = "User-scoped";
  itemsEl.innerHTML = '<li class="empty">Log in to view the inbox.</li>';
}

function setFormEnabled(enabled) {
  for (const control of form.elements) {
    control.disabled = !enabled;
  }
}

function setDeviceFormEnabled(enabled) {
  for (const control of deviceForm.elements) {
    control.disabled = !enabled;
  }
}

async function logout() {
  try {
    const res = await fetch("/auth/logout", {
      method: "POST",
      headers: csrfHeaders()
    });
    const body = await readResponse(res);
    if (!res.ok) throw new Error(body.error || body.message || `HTTP ${res.status}`);
    setLoggedOut();
    window.location.href = "/";
  } catch (err) {
    setStatus(`Logout failed: ${String(err.message || err)}`, true);
  }
}

async function approveDevice(userCode) {
  setDeviceStatus("Approving…");
  try {
    const res = await fetch("/auth/device/approve", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...csrfHeaders() },
      body: JSON.stringify({ user_code: userCode, actions: ["user.self.read"] })
    });
    const body = await readResponse(res);
    if (!res.ok) throw new Error(body.error_description || body.error || body.message || `HTTP ${res.status}`);
    deviceForm.reset();
    setDeviceStatus("Device approved. Return to the CLI and poll for tokens.");
  } catch (err) {
    setDeviceStatus(String(err.message || err), true);
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
    const body = await readResponse(res);
    if (!res.ok) throw new Error(body.error || body.message || `HTTP ${res.status}`);
    form.reset();
    setStatus("Captured.");
    await loadItems();
  } catch (err) {
    setStatus(String(err.message || err), true);
  }
}

async function loadItems() {
  if (!authenticated) {
    itemsEl.innerHTML = '<li class="empty">Log in to view the inbox.</li>';
    return;
  }
  try {
    const res = await fetch("/api/inbox");
    const body = await readResponse(res);
    if (!res.ok) throw new Error(body.error || body.message || `HTTP ${res.status}`);
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

async function readResponse(res) {
  const contentType = res.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return await res.json();
  }
  const text = await res.text();
  return { message: text || res.statusText };
}

function setStatus(message, isError = false) {
  statusEl.textContent = message;
  statusEl.classList.toggle("error", isError);
}

function setDeviceStatus(message, isError = false) {
  deviceStatusEl.textContent = message;
  deviceStatusEl.classList.toggle("error", isError);
}

function displayName(session) {
  const claims = session.claims || {};
  return claims.preferredUsername || session.email || claims.name || session.userId || "session user";
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
