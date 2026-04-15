import type { Meta, StoryObj } from "@storybook/react";
import { Callout } from "@/components/essay/Callout";

const meta = {
  title: "Essay/Callout",
  component: Callout,
  args: {
    children: "Callout"
  }
} satisfies Meta<typeof Callout>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <Callout>
      <strong>About this essay.</strong> Every section teaches through real feedback and ties the
      explanation back to the actual backend payload.
    </Callout>
  )
};
