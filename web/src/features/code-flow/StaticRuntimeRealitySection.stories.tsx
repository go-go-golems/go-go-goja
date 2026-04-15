import type { Meta, StoryObj } from "@storybook/react";
import { evaluateResponseFixture } from "@/features/meet-session/storyFixtures";
import { StaticRuntimeRealitySection } from "@/features/code-flow/StaticRuntimeRealitySection";

const meta = {
  title: "Features/CodeFlow/StaticRuntimeRealitySection",
  component: StaticRuntimeRealitySection,
  args: {
    response: evaluateResponseFixture
  }
} satisfies Meta<typeof StaticRuntimeRealitySection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Populated: Story = {};

export const Empty: Story = {
  args: {
    response: null
  }
};
