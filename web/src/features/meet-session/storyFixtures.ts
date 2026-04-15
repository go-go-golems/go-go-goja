import type {
  BootstrapResponse,
  EvaluateResponse,
  EvaluationBootstrapResponse,
  ProfilesBootstrapResponse,
  SessionPolicy,
  SessionSummary
} from "@/features/meet-session/types";

export const rawPolicyFixture: SessionPolicy = {
  eval: {
    mode: "raw",
    timeoutMs: 5000,
    captureLastExpression: false,
    supportTopLevelAwait: false
  },
  observe: {
    staticAnalysis: false,
    runtimeSnapshot: false,
    bindingTracking: false,
    consoleCapture: false,
    jsdocExtraction: false
  },
  persist: {
    enabled: false,
    sessions: false,
    evaluations: false,
    bindingVersions: false,
    bindingDocs: false
  }
};

export const interactivePolicyFixture: SessionPolicy = {
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
    enabled: false,
    sessions: false,
    evaluations: false,
    bindingVersions: false,
    bindingDocs: false
  }
};

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
  policy: persistentPolicyFixture,
  bindings: [],
  history: []
};

export const rawSessionFixture: SessionSummary = {
  ...sessionFixture,
  id: "session-raw-1",
  profile: "raw",
  policy: rawPolicyFixture
};

export const interactiveSessionFixture: SessionSummary = {
  ...sessionFixture,
  id: "session-interactive-1",
  profile: "interactive",
  policy: interactivePolicyFixture
};

export const profilesBootstrapFixture: ProfilesBootstrapResponse = {
  section: {
    id: "profiles-change-behavior",
    title: "Profiles Change Behavior",
    summary:
      "Raw, interactive, and persistent sessions are different execution contracts, not cosmetic labels.",
    intro: [
      "A profile is a named bundle of session policy. When you change the profile, you are changing what the REPL is allowed to do with your code before, during, and after execution.",
      "This section compares the three built-in profiles, then lets the browser create a real session using the selected profile so you can confirm that the backend summary matches the documented contract."
    ],
    primaryAction: {
      label: "Create Selected Profile",
      method: "POST",
      path: "/api/essay/sections/profiles-change-behavior/session"
    },
    panels: [
      {
        id: "profile-comparison",
        title: "Profile Comparison",
        kind: "comparison-table",
        description: "Side-by-side contract view for raw, interactive, and persistent profiles."
      }
    ]
  },
  selectedProfile: "interactive",
  profiles: [
    {
      id: "raw",
      title: "Raw",
      summary: "The thinnest possible layer over goja.",
      policy: rawPolicyFixture,
      highlights: [
        "Runs without instrumented helper rewriting.",
        "Does not capture the last expression automatically.",
        "Turns off static analysis and persistence."
      ]
    },
    {
      id: "interactive",
      title: "Interactive",
      summary: "Optimized for conversational exploration with in-memory rich observation.",
      policy: interactivePolicyFixture,
      highlights: [
        "Uses instrumented execution and last-expression capture.",
        "Enables static analysis, runtime snapshots, and binding tracking.",
        "Keeps state in memory only."
      ]
    },
    {
      id: "persistent",
      title: "Persistent",
      summary: "Adds durable storage on top of the interactive profile.",
      policy: persistentPolicyFixture,
      highlights: [
        "Keeps interactive instrumentation.",
        "Persists sessions, evaluations, binding versions, and docs.",
        "Supports restore/history/export workflows."
      ]
    }
  ],
  rawRoutes: [
    {
      method: "POST",
      path: "/api/essay/sections/profiles-change-behavior/session",
      purpose: "Article-scoped route that creates one session using the requested profile override."
    }
  ]
};

export const evaluationBootstrapFixture: EvaluationBootstrapResponse = {
  section: {
    id: "what-happened-to-my-code",
    title: "What Happened To My Code?",
    summary:
      "Instrumented sessions do not just execute source. They analyze it, rewrite it, execute it, and then report what changed.",
    intro: [
      "This section focuses on the evaluation pipeline. The important question is not only what the code returned, but what the system learned and what transformations it applied along the way.",
      "The browser prepares one real evaluation session, submits source to the live API, and then renders the backend's rewrite, execution, and runtime reports in synchronized views."
    ],
    primaryAction: {
      label: "Evaluate Source",
      method: "POST",
      path: "/api/essay/sections/what-happened-to-my-code/session/{sessionID}/evaluate"
    },
    panels: [
      {
        id: "source-transform",
        title: "Source Before and After",
        kind: "code-diff",
        description: "Original user source compared with the transformed source the runtime actually executes."
      }
    ]
  },
  defaultProfile: "interactive",
  starterSource: "const x = 1; x",
  examples: [
    {
      id: "capture-last-expression",
      label: "Capture last expression",
      source: "const x = 1; x",
      rationale: "Smallest useful example of declaration rewriting plus last-expression capture."
    },
    {
      id: "top-level-await",
      label: "Top-level await",
      source: "await Promise.resolve(41 + 1)",
      rationale: "Shows how instrumented sessions can support awaited values directly."
    },
    {
      id: "global-side-effect",
      label: "Global side effect",
      source: "globalThis.answer = 42; answer",
      rationale: "Makes runtime diffs and session-bound state changes visible."
    }
  ],
  rawRoutes: [
    {
      method: "POST",
      path: "/api/essay/sections/what-happened-to-my-code/session/{sessionID}/evaluate",
      purpose: "Article-scoped route that runs one live evaluation."
    }
  ]
};

export const evaluateResponseFixture: EvaluateResponse = {
  session: {
    ...interactiveSessionFixture,
    cellCount: 1,
    bindingCount: 1,
    bindings: [
      {
        name: "x",
        kind: "const",
        origin: "user",
        declaredInCell: 1,
        lastUpdatedCell: 1,
        runtime: {
          valueKind: "number",
          preview: "1"
        }
      }
    ],
    history: [
      {
        cellId: 1,
        createdAt: "2026-04-15T03:15:22.000000000Z",
        sourcePreview: "const x = 1; x",
        resultPreview: "1",
        status: "ok"
      }
    ]
  },
  cell: {
    id: 1,
    createdAt: "2026-04-15T03:15:22.000000000Z",
    source: "const x = 1; x",
    static: {
      diagnostics: [],
      topLevelBindings: [
        {
          name: "x",
          kind: "const",
          line: 1,
          snippet: "const x = 1",
          referenceCount: 1
        }
      ],
      unresolved: [],
      astNodeCount: 12,
      summary: [
        { label: "Bindings found", value: "1" },
        { label: "Unresolved identifiers", value: "0" },
        { label: "AST nodes", value: "12" }
      ]
    },
    rewrite: {
      mode: "instrumented",
      declaredNames: ["x"],
      helperNames: ["__capture", "__lastExpr"],
      lastHelperName: "__lastExpr",
      bindingHelperName: "__capture",
      capturedLastExpr: true,
      transformedSource: "var x = __capture(\"x\", 1); __lastExpr(x);",
      operations: [
        {
          kind: "const-to-var",
          detail: "const x became var x so the session can track and update the binding."
        },
        {
          kind: "capture-wrap",
          detail: "The initializer was wrapped with __capture so the runtime can persist the binding value."
        },
        {
          kind: "last-expression",
          detail: "The trailing expression was wrapped with __lastExpr so the REPL can surface a final value."
        }
      ],
      warnings: [],
      finalExpressionSource: "x"
    },
    execution: {
      status: "ok",
      result: "1",
      durationMs: 2,
      awaited: false,
      console: [],
      hadSideEffects: false,
      helperError: false
    },
    runtime: {
      diffs: [
        {
          name: "x",
          change: "created",
          after: "1",
          afterKind: "number",
          sessionBound: true
        }
      ],
      newBindings: ["x"],
      updatedBindings: [],
      removedBindings: [],
      leakedGlobals: [],
      persistedByWrap: ["x"],
      currentCellValue: "1"
    }
  }
};
