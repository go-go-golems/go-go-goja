import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { MeetSessionPage } from "@/features/meet-session/MeetSessionPage";
import {
  bootstrapFixture,
  durableSessionRecordFixture,
  evaluateResponseFixture,
  evaluationBootstrapFixture,
  persistenceBootstrapFixture,
  profilesBootstrapFixture,
  persistentPolicyFixture,
  sessionExportFixture,
  sessionFixture,
  timeoutBootstrapFixture
} from "@/features/meet-session/storyFixtures";
import { withEssayProviders } from "@/storybook/withEssayProviders";

const interactiveProfilePolicy =
  profilesBootstrapFixture.profiles.find((profile) => profile.id === "interactive")?.policy ??
  persistentPolicyFixture;

const emptyHandlers = [
  http.get("/api/essay/sections/meet-a-session", () => HttpResponse.json(bootstrapFixture)),
  http.post("/api/essay/sections/meet-a-session/session", () =>
    HttpResponse.json({ session: sessionFixture }, { status: 201 })
  ),
  http.get("/api/essay/sections/meet-a-session/session/:sessionID", ({ params }) =>
    HttpResponse.json({
      session: {
        ...sessionFixture,
        id: String(params.sessionID)
      }
    })
  ),
  http.get("/api/essay/sections/profiles-change-behavior", () =>
    HttpResponse.json(profilesBootstrapFixture)
  ),
  http.post("/api/essay/sections/profiles-change-behavior/session", () =>
    HttpResponse.json(
      {
        session: {
          ...sessionFixture,
          id: "session-story-profile",
          profile: "interactive",
          policy: interactiveProfilePolicy
        }
      },
      { status: 201 }
    )
  ),
  http.get("/api/essay/sections/what-happened-to-my-code", () =>
    HttpResponse.json(evaluationBootstrapFixture)
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session", () =>
    HttpResponse.json(
      {
        session: {
          ...sessionFixture,
          id: "session-story-code",
          profile: "interactive",
          policy: interactiveProfilePolicy
        }
      },
      { status: 201 }
    )
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session/:sessionID/evaluate", () =>
    HttpResponse.json(evaluateResponseFixture)
  ),
  http.get("/api/essay/sections/persistence-history-and-restore", () =>
    HttpResponse.json(persistenceBootstrapFixture)
  ),
  http.get("/api/essay/sections/timeouts-are-part-of-the-contract", () =>
    HttpResponse.json(timeoutBootstrapFixture)
  ),
  http.get("/api/sessions", () =>
    HttpResponse.json({ sessions: [durableSessionRecordFixture] })
  ),
  http.post("/api/sessions", () =>
    HttpResponse.json(
      {
        session: {
          ...sessionFixture,
          id: "session-story-persistent",
          profile: "persistent",
          policy: persistentPolicyFixture
        }
      },
      { status: 201 }
    )
  ),
  http.post("/api/sessions/:sessionID/evaluate", ({ request }) =>
    request
      .json()
      .then((body) => {
        const source = String((body as { source?: string }).source || "");
        if (source.includes("while") || source.includes("Promise")) {
          return HttpResponse.json({
            ...evaluateResponseFixture,
            cell: {
              ...evaluateResponseFixture.cell,
              source,
              execution: {
                ...evaluateResponseFixture.cell.execution,
                status: "timeout",
                result: "",
                error: "evaluation timed out"
              }
            }
          });
        }
        return HttpResponse.json(evaluateResponseFixture);
      })
  ),
  http.get("/api/sessions/:sessionID/history", () =>
    HttpResponse.json({ history: sessionExportFixture.evaluations })
  ),
  http.get("/api/sessions/:sessionID/export", () =>
    HttpResponse.json(sessionExportFixture)
  ),
  http.delete("/api/sessions/:sessionID", () =>
    HttpResponse.json({ deleted: true })
  ),
  http.post("/api/sessions/:sessionID/restore", () =>
    HttpResponse.json({ session: sessionFixture })
  )
];

const meta = {
  title: "Pages/MeetSessionPage",
  component: MeetSessionPage,
  decorators: [withEssayProviders()]
} satisfies Meta<typeof MeetSessionPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  parameters: {
    msw: {
      handlers: emptyHandlers
    }
  }
};

export const Created: Story = {
  decorators: [withEssayProviders({ meetSession: { activeSessionId: "sess_mock_0001" } })],
  parameters: {
    msw: {
      handlers: emptyHandlers
    }
  }
};

export const CreateRouteFailure: Story = {
  parameters: {
    msw: {
      handlers: [
        ...emptyHandlers,
        http.post("/api/essay/sections/meet-a-session/session", () =>
          HttpResponse.json({ error: "cannot create session in this profile" }, { status: 500 })
        )
      ]
    }
  }
};

export const SnapshotFailure: Story = {
  decorators: [withEssayProviders({ meetSession: { activeSessionId: "sess_unknown" } })],
  parameters: {
    msw: {
      handlers: [
        http.get("/api/essay/sections/meet-a-session", () => HttpResponse.json(bootstrapFixture)),
        http.post("/api/essay/sections/meet-a-session/session", () =>
          HttpResponse.json(
            {
              session: {
                ...sessionFixture,
                policy: persistentPolicyFixture
              }
            },
            { status: 201 }
          )
        ),
        http.get("/api/essay/sections/meet-a-session/session/:sessionID", () =>
          HttpResponse.json({ error: "session not found" }, { status: 404 })
        )
      ]
    }
  }
};
