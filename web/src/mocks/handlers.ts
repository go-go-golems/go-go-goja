import { http, HttpResponse } from "msw";
import type { ProfileName, SessionSummary } from "@/features/meet-session/types";
import {
  bootstrapFixture,
  evaluateResponseFixture,
  evaluationBootstrapFixture,
  interactivePolicyFixture,
  interactiveSessionFixture,
  profilesBootstrapFixture,
  persistentPolicyFixture
} from "@/features/meet-session/storyFixtures";

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";
const profilesSnapshotPrefix = "/api/essay/sections/profiles-change-behavior/session/";
const codeFlowSessionPrefix = "/api/essay/sections/what-happened-to-my-code/session/";

const sessions = new Map<string, SessionSummary>();
const codeSessions = new Map<string, SessionSummary>();
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
  })
];
