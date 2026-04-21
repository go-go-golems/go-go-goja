import type { Meta, StoryObj } from "@storybook/react";
import { evaluateResponseFixture } from "@/features/meet-session/storyFixtures";
import { SourceTransformPanels } from "@/features/code-flow/SourceTransformPanels";

const meta = {
  title: "Features/CodeFlow/SourceTransformPanels",
  component: SourceTransformPanels,
  args: {
    source: evaluateResponseFixture.cell.source,
    transformedSource: evaluateResponseFixture.cell.rewrite.transformedSource
  }
} satisfies Meta<typeof SourceTransformPanels>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
