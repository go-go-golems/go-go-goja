import type { BootstrapResponse, SessionPolicy, SessionSummary } from "@/features/meet-session/types";

export const persistentPolicyFixture: SessionPolicy = {
  eval: {
    mode: "instrumented",
    timeoutMs: 5000,
    captureLastExpression: true,
    supportTopLevelAwait: true
  },
  observe: {
    staticAnalysis: true,
    runtimeSnapshot: true,
    bindingTracking: true,
    consoleCapture: true,
    jsdocExtraction: true
  },
  persist: {
    enabled: true,
    sessions: true,
    evaluations: true,
    bindingVersions: true,
    bindingDocs: true
  }
};

export const bootstrapFixture: BootstrapResponse = {
  section: {
    id: "meet-a-session",
    title: "Meet a Session",
    summary:
      "Create one real REPL session, then use it to learn how identity, policy, and backend state fit together.",
    intro: [
      "A session is the durable unit of state in the new REPL. It is not only a prompt. It carries an id, a profile, a policy, and a growing body of runtime and persistence data.",
      "In this section, the browser will trigger one real session creation request, then render the resulting SessionSummary in several synchronized forms: prose, summary table, policy table, and raw JSON.",
      "The intended lesson is architectural. You should leave this section understanding which fields matter first, which backend routes produce them, and which source files own the behavior you are seeing."
    ],
    primaryAction: {
      label: "Create Session",
      method: "POST",
      path: "/api/essay/sections/meet-a-session/session"
    },
    panels: [
      {
        id: "session-summary",
        title: "Session Summary",
        kind: "summary-card",
        description: "Compact identity and count fields taken directly from SessionSummary."
      },
      {
        id: "policy-card",
        title: "Policy",
        kind: "policy-card",
        description: "Human-readable view of eval, observe, and persist policy fields."
      },
      {
        id: "session-json",
        title: "Raw Session JSON",
        kind: "json-inspector",
        description: "Exact JSON payload returned by the backend for trust and debugging."
      }
    ]
  },
  defaultView: {
    profile: "persistent",
    policy: persistentPolicyFixture
  },
  rawRoutes: [
    {
      method: "POST",
      path: "/api/essay/sections/meet-a-session/session",
      purpose: "Article-scoped route that creates one session using the essay's default persistent profile."
    },
    {
      method: "GET",
      path: "/api/essay/sections/meet-a-session/session/{sessionID}",
      purpose: "Article-scoped route that fetches a fresh read-model for one existing session id."
    },
    {
      method: "POST",
      path: "/api/sessions",
      purpose: "Underlying raw REPL create-session route exposed for trust and debugging."
    },
    {
      method: "GET",
      path: "/api/sessions/{sessionID}",
      purpose: "Underlying raw REPL snapshot route for direct inspection outside the essay wrapper."
    }
  ]
};

export const sessionFixture: SessionSummary = {
  id: "session-7a223b53-4875-4ad3-a1c3-d349a70b154a",
  profile: "persistent",
  createdAt: "2026-04-15T02:00:49.125710603Z",
  cellCount: 0,
  bindingCount: 0,
  policy: persistentPolicyFixture
};
