import type { Meta, StoryObj } from "@storybook/react";
import { Button } from "@/components/primitives/Button";

const meta = {
  title: "Primitives/Button",
  component: Button,
  args: {
    children: "Create Session"
  },
  render: (args) => (
    <div style={{ padding: "2rem", display: "flex", gap: "0.75rem", flexWrap: "wrap" }}>
      <Button {...args} variant="primary" />
      <Button {...args} variant="secondary" />
      <Button {...args} variant="ghost" />
    </div>
  )
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Sizes: Story = {
  render: (args) => (
    <div style={{ padding: "2rem", display: "flex", gap: "0.75rem", alignItems: "center" }}>
      <Button {...args} size="sm">
        Small
      </Button>
      <Button {...args} size="md">
        Medium
      </Button>
      <Button {...args} size="lg">
        Large
      </Button>
    </div>
  )
};
