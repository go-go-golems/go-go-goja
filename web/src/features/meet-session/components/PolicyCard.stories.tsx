import type { Meta, StoryObj } from "@storybook/react";
import { PolicyCard } from "@/features/meet-session/components/PolicyCard";
import { persistentPolicyFixture } from "@/features/meet-session/storyFixtures";

const meta = {
  title: "Meet Session/PolicyCard",
  component: PolicyCard,
  args: {
    policy: null
  }
} satisfies Meta<typeof PolicyCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {};

export const Persistent: Story = {
  args: {
    policy: persistentPolicyFixture
  }
};
