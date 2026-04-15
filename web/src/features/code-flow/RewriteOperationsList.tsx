import { Card, Tag } from "@/components/primitives";
import type { RewriteStep } from "@/features/meet-session/types";

type RewriteOperationsListProps = {
  operations: RewriteStep[];
};

export function RewriteOperationsList({
  operations
}: RewriteOperationsListProps) {
  return (
    <Card
      title="Rewrite Operations"
      subtitle="These are the explicit transformation steps reported by the backend for this one cell."
    >
      <div className="essay-operation-list">
        {operations.map((operation) => (
          <div key={`${operation.kind}-${operation.detail}`} className="essay-operation-list__row">
            <Tag variant="accent">{operation.kind}</Tag>
            <span>{operation.detail}</span>
          </div>
        ))}
      </div>
    </Card>
  );
}
