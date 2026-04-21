import type { Meta, StoryObj } from "@storybook/react";
import { Card } from "@/components/primitives/Card";
import { JsonViewer } from "@/components/primitives/JsonViewer";

const meta = {
  title: "Primitives/Card",
  component: Card,
  args: {
    children: "card content"
  },
  render: () => (
    <div style={{ maxWidth: "42rem", padding: "2rem" }}>
      <Card
        title="Session Summary"
        subtitle="Compact identity and counters from SessionSummary."
      >
        <JsonViewer
          collapsedByDefault={false}
          data={{
            session: {
              id: "sess_mock_0001",
              profile: "persistent",
              createdAt: "2026-04-15T10:21:00Z",
              cellCount: 0,
              bindingCount: 0
            }
          }}
          label="Session payload"
        />
      </Card>
    </div>
  )
} satisfies Meta<typeof Card>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
