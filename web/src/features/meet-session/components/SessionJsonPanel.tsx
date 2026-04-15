import { Card } from "@/components/primitives/Card";
import { JsonViewer } from "@/components/primitives/JsonViewer";
import type { SessionSummary } from "@/features/meet-session/types";

type SessionJsonPanelProps = {
  session: SessionSummary | null;
};

export function SessionJsonPanel({ session }: SessionJsonPanelProps) {
  return (
    <Card
      title="Raw Session JSON"
      subtitle="This is the literal payload shape the section is interpreting. Use it to verify every friendly explanation on the page."
    >
      <JsonViewer
        data={session ? { session } : { session: null }}
        label="SessionSummary payload"
        collapsedByDefault={false}
      />
    </Card>
  );
}
