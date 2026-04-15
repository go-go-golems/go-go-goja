import type { Meta, StoryObj } from "@storybook/react";
import { evaluateResponseFixture } from "@/features/meet-session/storyFixtures";
import { ExecutionResultSummary } from "@/features/code-flow/ExecutionResultSummary";

const meta = {
  title: "Features/CodeFlow/ExecutionResultSummary",
  component: ExecutionResultSummary,
  args: {
    execution: evaluateResponseFixture.cell.execution
  }
} satisfies Meta<typeof ExecutionResultSummary>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
