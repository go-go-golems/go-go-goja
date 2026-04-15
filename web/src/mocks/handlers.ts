import { http, HttpResponse } from "msw";
import type { SessionSummary } from "@/features/meet-session/types";
import {
  bootstrapFixture,
  persistentPolicyFixture
} from "@/features/meet-session/storyFixtures";

const meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/";

const sessions = new Map<string, SessionSummary>();
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
  })
];
