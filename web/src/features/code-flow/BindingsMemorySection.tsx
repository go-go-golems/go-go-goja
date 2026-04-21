import { useEffect, useMemo, useState } from "react";
import { Callout, Heading, Prose } from "@/components/essay";
import { Button, Card } from "@/components/primitives";
import type { SessionSummary } from "@/features/meet-session/types";

type BindingsMemorySectionProps = {
  session: SessionSummary | null;
};

export function BindingsMemorySection({
  session
}: BindingsMemorySectionProps) {
  const history = session?.history ?? [];
  const bindings = session?.bindings ?? [];
  const [selectedCell, setSelectedCell] = useState<number | null>(null);

  useEffect(() => {
    if (history.length > 0) {
      setSelectedCell(history[history.length - 1]?.cellId ?? null);
      return;
    }
    setSelectedCell(null);
  }, [history]);

  const visibleBindings = useMemo(() => {
    if (!selectedCell) {
      return bindings;
    }
    return bindings.filter((binding) => binding.declaredInCell <= selectedCell);
  }, [bindings, selectedCell]);

  return (
    <section className="essay-section-block">
      <Heading n="5">Bindings Are the Memory</Heading>
      <Prose>
        A session becomes useful because it accumulates bindings, not because it keeps a pile of raw
        command strings. The history tells you what cells ran. The binding table tells you which
        names still matter now, where they came from, and which parts of the environment survived
        into the next step.
      </Prose>
      <Prose>
        This section uses the current live session summary. If you evaluate several cells in the same
        session, the history and binding list should grow together. That gives the reader a concrete
        picture of how the REPL turns a stream of cells into a stateful working environment.
      </Prose>
      {session ? (
        <>
          <Card
            title="Session Timeline"
            subtitle="Select a recorded cell to approximate what the environment had accumulated by that point."
          >
            <div className="essay-example-row">
              {history.map((entry) => (
                <Button
                  key={entry.cellId}
                  type="button"
                  size="sm"
                  variant="secondary"
                  className="essay-timeline-button"
                  data-selected={selectedCell === entry.cellId}
                  onClick={() => setSelectedCell(entry.cellId)}
                >
                  Cell {entry.cellId}: {entry.sourcePreview}
                </Button>
              ))}
            </div>
            <div className="essay-meta-inline">
              {selectedCell
                ? `Showing bindings declared by or before cell ${selectedCell}.`
                : "Run more than one cell to make the growth pattern obvious."}
            </div>
          </Card>
          <Card
            title="Current Bindings"
            subtitle="The table is derived from the live SessionSummary, not reconstructed in the browser."
          >
            <table className="essay-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Kind</th>
                  <th>Preview</th>
                  <th>Declared</th>
                </tr>
              </thead>
              <tbody>
                {visibleBindings.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="essay-empty-note">
                      No bindings yet.
                    </td>
                  </tr>
                ) : (
                  visibleBindings.map((binding) => (
                    <tr key={`${binding.name}-${binding.declaredInCell}`}>
                      <td className="essay-profile-table__value">{binding.name}</td>
                      <td>{binding.kind}</td>
                      <td className="essay-table__value">{binding.runtime.preview}</td>
                      <td>{binding.declaredInCell}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </Card>
        </>
      ) : (
        <Callout>
          Run at least one cell first. This section reads <code>session.history</code> and{" "}
          <code>session.bindings</code> from the live summary that comes back after evaluation.
        </Callout>
      )}
    </section>
  );
}
