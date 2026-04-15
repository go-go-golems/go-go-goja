import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { MeetSessionPage } from "@/features/meet-session/MeetSessionPage";
import {
  bootstrapFixture,
  persistentPolicyFixture,
  sessionFixture
} from "@/features/meet-session/storyFixtures";
import { withEssayProviders } from "@/storybook/withEssayProviders";

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
