import { http, HttpResponse } from "msw";
import type {
  EvaluationRecord,
  ProfileName,
  SessionExport,
  SessionSummary
} from "@/features/meet-session/types";
import {
  bootstrapFixture,
  durableSessionRecordFixture,
  evaluateResponseFixture,
  evaluationBootstrapFixture,
  interactivePolicyFixture,
  interactiveSessionFixture,
  persistenceBootstrapFixture,
  profilesBootstrapFixture,
  persistentPolicyFixture,
  sessionExportFixture,
  timeoutBootstrapFixture
} from "@/features/meet-session/storyFixtures";

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";
const profilesSnapshotPrefix = "/api/essay/sections/profiles-change-behavior/session/";
const codeFlowSessionPrefix = "/api/essay/sections/what-happened-to-my-code/session/";

const sessions = new Map<string, SessionSummary>();
const codeSessions = new Map<string, SessionSummary>();
const persistentHistory = new Map<string, EvaluationRecord[]>();
const persistentExports = new Map<string, SessionExport>();
let sessionCounter = 0;

function createMockSession(): SessionSummary {
  sessionCounter += 1;
  return {
    id: `sess_mock_${sessionCounter.toString().padStart(4, "0")}`,
    profile: "persistent",
    createdAt: new Date().toISOString(),
    cellCount: 0,
    bindingCount: 0,
    policy: persistentPolicyFixture
  };
}

export const handlers = [
  http.get("/api/essay/sections/meet-a-session", () => HttpResponse.json(bootstrapFixture)),
  http.post("/api/essay/sections/meet-a-session/session", () => {
    const session = createMockSession();
    sessions.set(session.id, session);
    return HttpResponse.json({ session }, { status: 201 });
  }),
  http.get(`${meetSessionSnapshotPrefix}:sessionID`, ({ params }) => {
    const sessionID = String(params.sessionID || "");
    const session = sessions.get(sessionID);
    if (!session) {
      return HttpResponse.json({ error: `session ${sessionID} not found` }, { status: 404 });
    }
    return HttpResponse.json({ session });
  }),
  http.get("/api/essay/sections/profiles-change-behavior", () =>
    HttpResponse.json(profilesBootstrapFixture)
  ),
  http.post("/api/essay/sections/profiles-change-behavior/session", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { profile?: ProfileName };
    const profile = body.profile ?? "interactive";
    const policy =
      profile === "persistent"
        ? persistentPolicyFixture
        : profile === "interactive"
          ? interactivePolicyFixture
          : profilesBootstrapFixture.profiles.find((item) => item.id === "raw")?.policy ??
            interactivePolicyFixture;
    const session: SessionSummary = {
      id: `sess_profile_${profile}_${String(Date.now()).slice(-6)}`,
      profile,
      createdAt: new Date().toISOString(),
      cellCount: 0,
      bindingCount: 0,
      policy
    };
    sessions.set(session.id, session);
    return HttpResponse.json({ session }, { status: 201 });
  }),
  http.get(`${profilesSnapshotPrefix}:sessionID`, ({ params }) => {
    const sessionID = String(params.sessionID || "");
    const session = sessions.get(sessionID);
    if (!session) {
      return HttpResponse.json({ error: `session ${sessionID} not found` }, { status: 404 });
    }
    return HttpResponse.json({ session });
  }),
  http.get("/api/essay/sections/what-happened-to-my-code", () =>
    HttpResponse.json(evaluationBootstrapFixture)
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session", () => {
    const session: SessionSummary = {
      ...interactiveSessionFixture,
      id: `sess_code_${String(Date.now()).slice(-6)}`,
      createdAt: new Date().toISOString(),
      cellCount: 0,
      bindingCount: 0
    };
    codeSessions.set(session.id, session);
    return HttpResponse.json({ session }, { status: 201 });
  }),
  http.post(`${codeFlowSessionPrefix}:sessionID/evaluate`, ({ params }) => {
    const sessionID = String(params.sessionID || "");
    const baseSession = codeSessions.get(sessionID);
    if (!baseSession) {
      return HttpResponse.json({ error: `session ${sessionID} not found` }, { status: 404 });
    }
    const response = {
      ...evaluateResponseFixture,
      session: {
        ...baseSession,
        cellCount: baseSession.cellCount + 1,
        bindingCount: 1
      }
    };
    codeSessions.set(sessionID, response.session);
    return HttpResponse.json(response);
  }),
  http.get("/api/essay/sections/persistence-history-and-restore", () =>
    HttpResponse.json(persistenceBootstrapFixture)
  ),
  http.get("/api/essay/sections/timeouts-are-part-of-the-contract", () =>
    HttpResponse.json(timeoutBootstrapFixture)
  ),
  http.get("/api/sessions", () =>
    HttpResponse.json({
      sessions: [
        ...Array.from(sessions.values())
          .filter((session) => session.profile === "persistent")
          .map((session) => ({
            sessionId: session.id,
            createdAt: session.createdAt,
            updatedAt: session.createdAt,
            engineKind: "goja"
          })),
        ...Array.from(persistentExports.values()).map((exported) => exported.session)
      ]
    })
  ),
  http.post("/api/sessions", () => {
    const session = createMockSession();
    sessions.set(session.id, session);
    persistentHistory.set(session.id, []);
    persistentExports.set(session.id, {
      session: {
        ...durableSessionRecordFixture,
        sessionId: session.id,
        createdAt: session.createdAt,
        updatedAt: session.createdAt
      },
      evaluations: []
    });
    return HttpResponse.json({ session }, { status: 201 });
  }),
  http.post("/api/sessions/:sessionID/evaluate", async ({ params, request }) => {
    const sessionID = String(params.sessionID || "");
    const source = String(((await request.json().catch(() => ({}))) as { source?: string }).source || "");
    const session = sessions.get(sessionID);
    if (!session) {
      return HttpResponse.json({ error: `session ${sessionID} not found` }, { status: 404 });
    }
    const history = persistentHistory.get(sessionID) ?? [];
    const nextRecord: EvaluationRecord = {
      evaluationId: history.length + 1,
      sessionId: sessionID,
      cellId: history.length + 1,
      createdAt: new Date().toISOString(),
      rawSource: source,
      rewrittenSource: source,
      ok: !source.includes("while") && !source.includes("Promise"),
      resultJson: source.includes("1 + 1") ? 2 : 1,
      errorText: source.includes("while") || source.includes("Promise") ? "evaluation timed out" : ""
    };
    persistentHistory.set(sessionID, [...history, nextRecord]);
    const exported = persistentExports.get(sessionID);
    if (exported) {
      exported.evaluations = [...persistentHistory.get(sessionID)!];
    }
    const timeout = source.includes("while") || source.includes("Promise");
    const response = {
      ...evaluateResponseFixture,
      session: {
        ...session,
        cellCount: history.length + 1,
        bindingCount: timeout ? session.bindingCount : Math.max(session.bindingCount, 1)
      },
      cell: {
        ...evaluateResponseFixture.cell,
        id: history.length + 1,
        source,
        execution: {
          ...evaluateResponseFixture.cell.execution,
          status: timeout ? "timeout" : "ok",
          result: timeout ? "" : source.includes("1 + 1") ? "2" : "1",
          error: timeout ? "evaluation timed out" : undefined
        }
      }
    };
    sessions.set(sessionID, response.session);
    return HttpResponse.json(response);
  }),
  http.get("/api/sessions/:sessionID/history", ({ params }) => {
    const sessionID = String(params.sessionID || "");
    return HttpResponse.json({ history: persistentHistory.get(sessionID) ?? sessionExportFixture.evaluations });
  }),
  http.get("/api/sessions/:sessionID/export", ({ params }) => {
    const sessionID = String(params.sessionID || "");
    return HttpResponse.json(
      persistentExports.get(sessionID) ?? {
        ...sessionExportFixture,
        session: {
          ...sessionExportFixture.session,
          sessionId: sessionID
        }
      }
    );
  }),
  http.delete("/api/sessions/:sessionID", () =>
    HttpResponse.json({ deleted: true })
  ),
  http.post("/api/sessions/:sessionID/restore", ({ params }) => {
    const sessionID = String(params.sessionID || "");
    const existing = sessions.get(sessionID);
    return HttpResponse.json({
      session:
        existing ?? {
          ...createMockSession(),
          id: sessionID
        }
    });
  })
];
