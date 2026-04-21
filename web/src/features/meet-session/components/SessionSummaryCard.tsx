import { Card } from "@/components/primitives/Card";
import { Prose } from "@/components/essay/Prose";
import type { SessionSummary } from "@/features/meet-session/types";

type SessionSummaryCardProps = {
  session: SessionSummary | null;
};

type SummaryRow = {
  label: string;
  value: string | number;
};

export function SessionSummaryCard({ session }: SessionSummaryCardProps) {
  if (!session) {
    return (
      <Card
        title="□ Session Summary"
        subtitle="This card will show the smallest high-signal slice of SessionSummary once the backend creates a session."
      >
        <Prose>
          Start here after you press <code>Create Session</code>. The goal of this card is not to
          show everything. It is to show the few fields you will use immediately when reasoning
          about identity, lifecycle, and later API calls.
        </Prose>
      </Card>
    );
  }

  const rows: SummaryRow[] = [
    { label: "ID", value: session.id },
    { label: "Profile", value: session.profile },
    { label: "Created", value: new Date(session.createdAt).toLocaleTimeString() },
    { label: "Cells", value: session.cellCount },
    { label: "Bindings", value: session.bindingCount }
  ];

  return (
    <Card
      title="□ Session Summary"
      subtitle="A compact read-model of the live session. Use it as the human-readable twin of the raw JSON panel."
    >
      <table className="essay-table">
        <tbody>
          {rows.map((row) => (
            <tr key={row.label}>
              <td className="essay-table__label">{row.label}</td>
              <td className="essay-table__value">{row.value}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="essay-prose essay-prose--after-table">
        Notice that the session ID is the operational handle, while the profile and counters tell
        you what kind of session exists and how much work it has accumulated so far.
      </p>
    </Card>
  );
}
