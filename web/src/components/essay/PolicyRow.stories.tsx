import type { Meta, StoryObj } from "@storybook/react";
import { PolicyRow } from "@/components/essay/PolicyRow";

const meta = {
  title: "Essay/PolicyRow",
  component: PolicyRow,
  args: {
    label: "top-level await",
    value: true
  }
} satisfies Meta<typeof PolicyRow>;

export default meta;
type Story = StoryObj<typeof meta>;

function renderInTable(args: { label: string; value: string | number | boolean }) {
  return (
    <table className="essay-table" style={{ width: 320 }}>
      <tbody>
        <PolicyRow {...args} />
      </tbody>
    </table>
  );
}

export const BooleanTrue: Story = {
  render: renderInTable
};

export const BooleanFalse: Story = {
  args: {
    label: "binding tracking",
    value: false
  },
  render: renderInTable
};

export const StringValue: Story = {
  args: {
    label: "mode",
    value: "instrumented"
  },
  render: renderInTable
};
