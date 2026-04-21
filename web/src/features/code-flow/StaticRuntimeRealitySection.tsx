import { Callout, Heading, Prose, Row, Col } from "@/components/essay";
import { Card } from "@/components/primitives";
import type { EvaluateResponse } from "@/features/meet-session/types";

type StaticRuntimeRealitySectionProps = {
  response: EvaluateResponse | null;
};

export function StaticRuntimeRealitySection({
  response
}: StaticRuntimeRealitySectionProps) {
  const diagnostics = response?.cell.static.diagnostics ?? [];
  const topLevelBindings = response?.cell.static.topLevelBindings ?? [];
  const unresolved = response?.cell.static.unresolved ?? [];
  const newBindings = response?.cell.runtime.newBindings ?? [];
  const updatedBindings = response?.cell.runtime.updatedBindings ?? [];
  const removedBindings = response?.cell.runtime.removedBindings ?? [];
  const leakedGlobals = response?.cell.runtime.leakedGlobals ?? [];
  const diffs = response?.cell.runtime.diffs ?? [];

  return (
    <section className="essay-section-block">
      <Heading n="4">Static Analysis vs Runtime Reality</Heading>
      <Prose>
        The REPL learns some facts before execution and some only after the runtime has mutated the
        session. Static analysis can tell you which bindings appear in the source, how many AST
        nodes were involved, and whether identifiers are unresolved. Runtime inspection can tell you
        which bindings actually appeared, which globals changed, and what the cell value became.
      </Prose>
      <Prose>
        That distinction matters because it teaches the provenance model of the system. Parser facts
        and runtime facts are both useful, but they answer different questions and should not be
        conflated.
      </Prose>
      {response ? (
        <Row>
          <Col>
            <Card
              title="Before Execution (Static)"
              subtitle="Facts available from parsing and analysis before the transformed source is run."
            >
              <table className="essay-table">
                <tbody>
                  <tr>
                    <td className="essay-table__label">Bindings found</td>
                    <td>{topLevelBindings.length}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">Diagnostics</td>
                    <td>{diagnostics.length}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">Unresolved</td>
                    <td>{unresolved.length}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">AST nodes</td>
                    <td>{response.cell.static.astNodeCount}</td>
                  </tr>
                </tbody>
              </table>
              <div className="essay-fact-list">
                {topLevelBindings.map((binding) => (
                  <div key={`${binding.name}-${binding.line}`}>
                    <span className="essay-emph-blue">{binding.kind}</span>{" "}
                    <strong>{binding.name}</strong>{" "}
                    <span className="essay-meta-inline">line {binding.line}</span>
                  </div>
                ))}
              </div>
            </Card>
          </Col>
          <Col>
            <Card
              title="After Execution (Runtime)"
              subtitle="Facts that only exist after the VM has run the rewritten cell and the session has been re-observed."
            >
              <table className="essay-table">
                <tbody>
                  <tr>
                    <td className="essay-table__label">New bindings</td>
                    <td>{newBindings.join(", ") || "—"}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">Updated</td>
                    <td>{updatedBindings.join(", ") || "—"}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">Removed</td>
                    <td>{removedBindings.join(", ") || "—"}</td>
                  </tr>
                  <tr>
                    <td className="essay-table__label">Leaked globals</td>
                    <td>{leakedGlobals.join(", ") || "—"}</td>
                  </tr>
                </tbody>
              </table>
              <div className="essay-fact-list">
                <div className="essay-fact-list__label">Global diffs</div>
                {diffs.length === 0 ? (
                  <div className="essay-meta-inline">No global changes recorded.</div>
                ) : (
                  diffs.map((diff) => (
                    <div key={`${diff.name}-${diff.change}`}>
                      <strong>{diff.name}</strong>:{" "}
                      <span className="essay-meta-inline">{diff.before || "undefined"}</span> →{" "}
                      <span className="essay-result-grid__value">{diff.after || "—"}</span>
                    </div>
                  ))
                )}
              </div>
            </Card>
          </Col>
        </Row>
      ) : (
        <Callout>
          Run one live evaluation first. The section needs a real <code>EvaluateResponse</code> so
          it can compare the parser-derived report against the runtime-derived report.
        </Callout>
      )}
    </section>
  );
}
