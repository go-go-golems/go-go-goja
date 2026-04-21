import type { Meta, StoryObj } from "@storybook/react";
import { Prose } from "@/components/essay/Prose";

const meta = {
  title: "Essay/Prose",
  component: Prose,
  args: {
    children:
      "A session is the durable unit of state in the new REPL. It carries an id, a profile, a policy, and a growing body of runtime and persistence data."
  }
} satisfies Meta<typeof Prose>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithInlineCode: Story = {
  render: () => (
    <Prose>
      The browser talks to <code>goja-repl essay</code> over HTTP and treats the returned{" "}
      <code>SessionSummary</code> as the source of truth for this section.
    </Prose>
  )
};
