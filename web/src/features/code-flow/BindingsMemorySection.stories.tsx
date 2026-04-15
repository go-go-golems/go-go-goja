import type { Meta, StoryObj } from "@storybook/react";
import {
  evaluateResponseFixture,
  interactiveSessionFixture
} from "@/features/meet-session/storyFixtures";
import { BindingsMemorySection } from "@/features/code-flow/BindingsMemorySection";

const meta = {
  title: "Features/CodeFlow/BindingsMemorySection",
  component: BindingsMemorySection,
  args: {
    session: evaluateResponseFixture.session
  }
} satisfies Meta<typeof BindingsMemorySection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Populated: Story = {};

export const Empty: Story = {
  args: {
    session: interactiveSessionFixture
  }
};
