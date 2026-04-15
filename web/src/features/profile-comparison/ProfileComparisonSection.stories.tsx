import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { withEssayProviders } from "@/storybook/withEssayProviders";
import {
  profilesBootstrapFixture,
  rawSessionFixture
} from "@/features/meet-session/storyFixtures";
import { ProfileComparisonSection } from "@/features/profile-comparison/ProfileComparisonSection";

const sectionHandlers = [
  http.get("/api/essay/sections/profiles-change-behavior", () =>
    HttpResponse.json(profilesBootstrapFixture)
  ),
  http.post("/api/essay/sections/profiles-change-behavior/session", () =>
    HttpResponse.json({ session: rawSessionFixture }, { status: 201 })
  )
];

const meta = {
  title: "Features/ProfileComparison/ProfileComparisonSection",
  component: ProfileComparisonSection,
  decorators: [withEssayProviders()],
  parameters: {
    msw: {
      handlers: sectionHandlers
    }
  }
} satisfies Meta<typeof ProfileComparisonSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
