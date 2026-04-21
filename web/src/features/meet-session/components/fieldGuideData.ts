import type { BootstrapResponse, SessionSummary } from "@/features/meet-session/types";

export const fallbackRoutes = [
  {
    method: "POST",
    path: "/api/essay/sections/meet-a-session/session",
    purpose: "Create one new session through the article wrapper."
  },
  {
    method: "GET",
    path: "/api/essay/sections/meet-a-session/session/{sessionID}",
    purpose: "Fetch a fresh snapshot for an existing session ID."
  }
];

export const fileGuide = [
  {
    path: "pkg/replessay/handler.go",
    note: "Owns the article page route, the bootstrap payload, and the article-scoped API wrapper."
  },
  {
    path: "pkg/replapi/app.go",
    note: "Defines the high-level application surface for creating sessions and reading snapshots."
  },
  {
    path: "pkg/replsession/service.go",
    note: "Owns session lifecycle and orchestration around the Goja runtime."
  },
  {
    path: "pkg/replsession/evaluate.go",
    note: "Runs code evaluation, timeout control, and top-level await handling."
  },
  {
    path: "pkg/repldb/store.go",
    note: "Initializes SQLite persistence and connection-level safety settings."
  },
  {
    path: "web/src/app/api/essayApi.ts",
    note: "Defines the browser-side HTTP contract used by this page."
  },
  {
    path: "web/src/features/meet-session/MeetSessionPage.tsx",
    note: "Composes the section UI and connects the live data sources."
  }
];

export function buildRequestFlowDiagram(sessionID: string | null) {
  const snapshotPath = sessionID
    ? `/api/essay/sections/meet-a-session/session/${sessionID}`
    : "/api/essay/sections/meet-a-session/session/{sessionID}";

  return [
    "Browser: MeetSessionPage",
    "  |",
    "  | 1. POST /api/essay/sections/meet-a-session/session",
    "  v",
    "pkg/replessay/handler.go",
    "  |",
    "  | 2. app.CreateSession(ctx)",
    "  v",
    "pkg/replapi/app.go",
    "  |",
    "  | 3. create session with persistent defaults",
    "  v",
    "pkg/replsession/*  +  pkg/repldb/*",
    "  |",
    "  | 4. return SessionSummary JSON",
    "  v",
    "Browser stores session.id",
    "  |",
    `  | 5. GET ${snapshotPath}`,
    "  v",
    "Browser rehydrates the same session view"
  ].join("\n");
}

export function buildCreateSessionPseudocode(sessionID: string | null) {
  const snapshotPath = sessionID
    ? `/api/essay/sections/meet-a-session/session/${sessionID}`
    : "/api/essay/sections/meet-a-session/session/{sessionID}";

  return [
    "// frontend",
    "onCreateSessionButtonClick():",
    "  response = POST /api/essay/sections/meet-a-session/session",
    "  session  = response.session",
    "  activeSessionId = session.id",
    `  snapshot = GET ${snapshotPath}`,
    "  render(summaryCard, policyCard, jsonPanel)",
    "",
    "// backend",
    "HandleCreateSession(ctx):",
    "  app = replapi.App",
    "  summary = app.CreateSession(ctx)",
    "  return { session: summary }",
    "",
    "CreateSession(ctx):",
    "  options = PersistentSessionOptions()",
    "  service = new session runtime + policy + persistence wiring",
    "  maybe persist session metadata",
    "  return SessionSummary(service)"
  ].join("\n");
}

export function resolveRoutes(
  bootstrap: BootstrapResponse | undefined,
  session: SessionSummary | null
) {
  const routes = bootstrap?.rawRoutes?.length ? bootstrap.rawRoutes : fallbackRoutes;
  const sessionID = session?.id ?? null;

  return routes.map((route) => ({
    ...route,
    resolvedPath:
      sessionID && route.path.includes("{sessionID}")
        ? route.path.replace("{sessionID}", sessionID)
        : route.path
  }));
}
