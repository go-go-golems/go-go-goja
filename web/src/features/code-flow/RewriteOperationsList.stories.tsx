import type { Meta, StoryObj } from "@storybook/react";
import { evaluateResponseFixture } from "@/features/meet-session/storyFixtures";
import { RewriteOperationsList } from "@/features/code-flow/RewriteOperationsList";

const meta = {
  title: "Features/CodeFlow/RewriteOperationsList",
  component: RewriteOperationsList,
  args: {
    operations: evaluateResponseFixture.cell.rewrite.operations
  }
} satisfies Meta<typeof RewriteOperationsList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
