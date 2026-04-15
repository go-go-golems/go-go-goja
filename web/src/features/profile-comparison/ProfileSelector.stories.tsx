import type { Meta, StoryObj } from "@storybook/react";
import { ProfileSelector } from "@/features/profile-comparison/ProfileSelector";
import { profilesBootstrapFixture } from "@/features/meet-session/storyFixtures";

const meta = {
  title: "Features/ProfileComparison/ProfileSelector",
  component: ProfileSelector,
  args: {
    profiles: profilesBootstrapFixture.profiles,
    selectedProfile: "interactive",
    onSelect: () => undefined
  }
} satisfies Meta<typeof ProfileSelector>;

export default meta;
type Story = StoryObj<typeof meta>;

export const InteractiveSelected: Story = {};

export const PersistentSelected: Story = {
  args: {
    selectedProfile: "persistent",
    onSelect: () => undefined
  }
};
