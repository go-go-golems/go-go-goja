let csrfToken = "";
let lastInviteToken = "";

async function fetchText(url, options) {
  const res = await fetch(url, options);
  const text = await res.text();
  let body = text;
  try { body = JSON.parse(text); } catch (_) {}
  return { status: res.status, ok: res.ok, body };
}

function show(id, value) {
  document.getElementById(id).textContent = typeof value === "string" ? value : JSON.stringify(value, null, 2);
}

async function loadSession() {
  const out = await fetchText("/auth/session");
  if (out.ok) {
    csrfToken = out.body.csrfToken || "";
    document.getElementById("session-status").innerHTML = '<span class="ok">Logged in</span>';
  } else {
    document.getElementById("session-status").innerHTML = '<span class="bad">Not logged in</span>';
  }
  show("session-output", out);
}

async function loadMe() { show("session-output", await fetchText("/me")); }

async function postLogout() {
  show("session-output", await fetchText("/auth/logout", { method:"POST" }));
  csrfToken = "";
  await loadSession();
}

async function updateProject() {
  if (!csrfToken) await loadSession();
  show("project-output", await fetchText("/orgs/o1/projects/p1", { method:"PATCH", headers:{ "X-CSRF-Token": csrfToken } }));
}

async function updateMissingProject() {
  if (!csrfToken) await loadSession();
  show("project-output", await fetchText("/orgs/o1/projects/missing", { method:"PATCH", headers:{ "X-CSRF-Token": csrfToken } }));
}

async function issueInvite() {
  if (!csrfToken) await loadSession();
  const out = await fetchText("/orgs/o1/invites", {
    method:"POST",
    headers:{ "Content-Type":"application/json", "X-CSRF-Token": csrfToken },
    body: JSON.stringify({ email:"invitee@example.test", role:"viewer" })
  });
  if (out.ok) lastInviteToken = out.body.token || "";
  show("invite-output", out);
}

async function acceptInvite() {
  show("invite-output", await fetchText("/org-invites/accept", {
    method:"POST",
    headers:{ "Content-Type":"application/json" },
    body: JSON.stringify({ token:lastInviteToken })
  }));
}

async function loadAudit() { show("audit-output", await fetchText("/auth/audit")); }

loadSession();
