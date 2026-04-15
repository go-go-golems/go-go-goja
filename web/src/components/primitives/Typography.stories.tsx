import type { Meta, StoryObj } from "@storybook/react";
import { Typography } from "@/components/primitives/Typography";

const meta = {
  title: "Primitives/Typography",
  component: Typography,
  args: {
    children: "Sample text"
  },
  render: () => (
    <div style={{ maxWidth: "46rem", padding: "2rem", display: "grid", gap: "0.8rem" }}>
      <Typography as="h1" variant="display">
        The REPL Essay
      </Typography>
      <Typography as="p" variant="title" tone="muted">
        An interactive guide to the session model and evaluation pipeline.
      </Typography>
      <Typography as="p" variant="body">
        Create one live session and inspect identity, policy, and backend payloads.
      </Typography>
      <Typography as="p" variant="caption" tone="accent">
        GOJA-043 · Component foundation
      </Typography>
      <Typography as="code" variant="mono">
        POST /api/essay/sections/meet-a-session/session
      </Typography>
    </div>
  )
} satisfies Meta<typeof Typography>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
