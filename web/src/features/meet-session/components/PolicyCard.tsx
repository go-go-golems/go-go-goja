import { Card } from "@/components/primitives/Card";
import { PolicyRow } from "@/components/essay/PolicyRow";
import { Prose } from "@/components/essay/Prose";
import type { SessionPolicy } from "@/features/meet-session/types";

type PolicyCardProps = {
  policy: SessionPolicy | null;
};

type PolicyGroup = {
  title: string;
  rows: Array<{
    label: string;
    value: string | number | boolean;
  }>;
};

export function PolicyCard({ policy }: PolicyCardProps) {
  if (!policy) {
    return (
      <Card
        title="□ Policy Card"
        subtitle="Policy is the machine-readable behavior contract for the session."
      >
        <p className="essay-empty-note">No policy data yet.</p>
      </Card>
    );
  }

  const groups: PolicyGroup[] = [
    {
      title: "Eval",
      rows: [
        { label: "mode", value: policy.eval.mode },
        { label: "timeout", value: `${policy.eval.timeoutMs}ms` },
        { label: "capture last expr", value: policy.eval.captureLastExpression },
        { label: "top-level await", value: policy.eval.supportTopLevelAwait }
      ]
    },
    {
      title: "Observe",
      rows: [
        { label: "static analysis", value: policy.observe.staticAnalysis },
        { label: "runtime snapshot", value: policy.observe.runtimeSnapshot },
        { label: "binding tracking", value: policy.observe.bindingTracking }
      ]
    },
    {
      title: "Persist",
      rows: [
        { label: "enabled", value: policy.persist.enabled }
      ]
    }
  ];

  return (
    <Card
      title="□ Policy Card"
      subtitle="Read this object as the executable answer to: how will this session evaluate code, observe results, and persist state?"
    >
      {groups.map((group) => (
        <section key={group.title} className="essay-policy-table-block">
          <h3>{group.title}</h3>
          <table className="essay-table">
            <tbody>
              {group.rows.map((row) => (
                <PolicyRow key={row.label} label={row.label} value={row.value} />
              ))}
            </tbody>
          </table>
        </section>
      ))}
      <Prose>
        For a new engineer, this card is often more important than the friendly prose. It is the
        precise backend contract that explains why later evaluation behavior looks the way it does.
      </Prose>
    </Card>
  );
}
