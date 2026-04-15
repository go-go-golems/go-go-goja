import { Card } from "@/components/primitives";
import type { ExecutionReport } from "@/features/meet-session/types";

type ExecutionResultSummaryProps = {
  execution: ExecutionReport;
};

export function ExecutionResultSummary({
  execution
}: ExecutionResultSummaryProps) {
  return (
    <Card
      title="Execution Result"
      subtitle="This is the runtime outcome after any rewrite and helper insertion already happened."
    >
      <div className="essay-result-grid">
        <span>
          <strong>Status:</strong>{" "}
          <span className="essay-result-grid__value">{execution.status}</span>
        </span>
        <span>
          <strong>Result:</strong>{" "}
          <span className="essay-result-grid__value essay-result-grid__value--mono">
            {execution.result || "—"}
          </span>
        </span>
        <span>
          <strong>Duration:</strong> {execution.durationMs}ms
        </span>
        <span>
          <strong>Awaited:</strong> {execution.awaited ? "yes" : "no"}
        </span>
      </div>
      {execution.error ? (
        <p className="essay-inline-error">{execution.error}</p>
      ) : null}
    </Card>
  );
}
