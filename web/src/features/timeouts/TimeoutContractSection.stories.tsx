import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { withEssayProviders } from "@/storybook/withEssayProviders";
import {
  evaluateResponseFixture,
  interactiveSessionFixture,
  timeoutBootstrapFixture
} from "@/features/meet-session/storyFixtures";
import { TimeoutContractSection } from "@/features/timeouts/TimeoutContractSection";

const handlers = [
  http.get("/api/essay/sections/timeouts-are-part-of-the-contract", () =>
    HttpResponse.json(timeoutBootstrapFixture)
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session", () =>
    HttpResponse.json({ session: interactiveSessionFixture }, { status: 201 })
  ),
  http.post("/api/essay/sections/what-happened-to-my-code/session/:sessionID/evaluate", ({ request }) =>
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
  )
];

const meta = {
  title: "Features/Timeouts/TimeoutContractSection",
  component: TimeoutContractSection,
  decorators: [withEssayProviders()],
  parameters: {
    msw: {
      handlers
    }
  }
} satisfies Meta<typeof TimeoutContractSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
