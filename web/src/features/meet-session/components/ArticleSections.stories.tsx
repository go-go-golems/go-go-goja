import type { Meta, StoryObj } from "@storybook/react";
import { ApiReferenceSection } from "@/features/meet-session/components/ApiReferenceSection";
import { FieldsMatterSection } from "@/features/meet-session/components/FieldsMatterSection";
import { HappyPathSection } from "@/features/meet-session/components/HappyPathSection";
import { HowToReadSection } from "@/features/meet-session/components/HowToReadSection";
import { MentalModelSection } from "@/features/meet-session/components/MentalModelSection";
import { PolicyGuideSection } from "@/features/meet-session/components/PolicyGuideSection";
import { SourceFileGuideSection } from "@/features/meet-session/components/SourceFileGuideSection";
import { ValidationExercisesSection } from "@/features/meet-session/components/ValidationExercisesSection";
import {
  bootstrapFixture,
  sessionFixture
} from "@/features/meet-session/storyFixtures";

const meta = {
  title: "Meet Session/Article Sections",
  component: HowToReadSection
} satisfies Meta<typeof HowToReadSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const HowToRead: Story = {
  render: () => <HowToReadSection />
};

export const MentalModel: Story = {
  render: () => <MentalModelSection sessionID={sessionFixture.id} />
};

export const FieldsMatter: Story = {
  render: () => <FieldsMatterSection />
};

export const HappyPath: Story = {
  render: () => <HappyPathSection sessionID={sessionFixture.id} />
};

export const PolicyGuide: Story = {
  render: () => <PolicyGuideSection />
};

export const ApiReference: Story = {
  render: () => <ApiReferenceSection bootstrap={bootstrapFixture} session={sessionFixture} />
};

export const SourceFileGuide: Story = {
  render: () => <SourceFileGuideSection />
};

export const ValidationExercises: Story = {
  render: () => <ValidationExercisesSection />
};
