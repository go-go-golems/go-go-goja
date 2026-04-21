import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { withEssayProviders } from "@/storybook/withEssayProviders";
import {
  evaluateResponseFixture,
  evaluationBootstrapFixture,
  interactiveSessionFixture
} from "@/features/meet-session/storyFixtures";
import { CodeEvaluationSection } from "@/features/code-flow/CodeEvaluationSection";

const handlers = [
  http.get("/api/essay/sections/what-happened-to-my-code", () =>
    HttpResponse.json(evaluationBootstrapFixture)
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session", () =>
    HttpResponse.json({ session: interactiveSessionFixture }, { status: 201 })
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session/:sessionID/evaluate", () =>
    HttpResponse.json(evaluateResponseFixture)
  )
];

const meta = {
  title: "Features/CodeFlow/CodeEvaluationSection",
  component: CodeEvaluationSection,
  decorators: [withEssayProviders()],
  parameters: {
    msw: {
      handlers
    }
  }
} satisfies Meta<typeof CodeEvaluationSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
