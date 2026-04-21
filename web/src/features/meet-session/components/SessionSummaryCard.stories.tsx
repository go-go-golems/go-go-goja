import type { Meta, StoryObj } from "@storybook/react";
import { SessionSummaryCard } from "@/features/meet-session/components/SessionSummaryCard";
import { sessionFixture } from "@/features/meet-session/storyFixtures";

const meta = {
  title: "Meet Session/SessionSummaryCard",
  component: SessionSummaryCard,
  args: {
    session: null
  }
} satisfies Meta<typeof SessionSummaryCard>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {};

export const Created: Story = {
  args: {
    session: sessionFixture
  }
};
