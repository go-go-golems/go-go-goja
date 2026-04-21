import type { Meta, StoryObj } from "@storybook/react";
import { profilesBootstrapFixture } from "@/features/meet-session/storyFixtures";
import { ProfileComparisonTable } from "@/features/profile-comparison/ProfileComparisonTable";

const meta = {
  title: "Features/ProfileComparison/ProfileComparisonTable",
  component: ProfileComparisonTable,
  args: {
    profiles: profilesBootstrapFixture.profiles,
    selectedProfile: "interactive"
  }
} satisfies Meta<typeof ProfileComparisonTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const InteractiveHighlighted: Story = {};

export const RawHighlighted: Story = {
  args: {
    selectedProfile: "raw"
  }
};
