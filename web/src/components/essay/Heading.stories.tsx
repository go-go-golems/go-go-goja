import type { Meta, StoryObj } from "@storybook/react";
import { Heading } from "@/components/essay/Heading";

const meta = {
  title: "Essay/Heading",
  component: Heading,
  args: {
    n: 1,
    children: "Meet a Session"
  }
} satisfies Meta<typeof Heading>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const HigherSection: Story = {
  args: {
    n: 4,
    children: "Evaluation And Rewrite"
  }
};
