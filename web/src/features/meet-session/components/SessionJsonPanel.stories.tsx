import type { Meta, StoryObj } from "@storybook/react";
import { SessionJsonPanel } from "@/features/meet-session/components/SessionJsonPanel";
import { sessionFixture } from "@/features/meet-session/storyFixtures";

const meta = {
  title: "Meet Session/SessionJsonPanel",
  component: SessionJsonPanel,
  args: {
    session: null
  }
} satisfies Meta<typeof SessionJsonPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {};

export const Created: Story = {
  args: {
    session: sessionFixture
  }
};
