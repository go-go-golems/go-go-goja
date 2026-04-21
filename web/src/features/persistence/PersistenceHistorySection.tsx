import { skipToken } from "@reduxjs/toolkit/query";
import { useEffect, useMemo, useState } from "react";
import {
  useCreatePersistentSessionMutation,
  useDeletePersistentSessionMutation,
  useEvaluatePersistentSessionMutation,
  useGetPersistenceBootstrapQuery,
  useGetPersistentSessionExportQuery,
  useGetPersistentSessionHistoryQuery,
  useListPersistentSessionsQuery,
  useRestorePersistentSessionMutation
} from "@/app/api/essayApi";
import { Callout, Heading, Prose, Row, Col } from "@/components/essay";
import { Button, Card, JsonViewer, Typography } from "@/components/primitives";

type ErrorWithMessage = {
  data?: {
    error?: string;
  };
  error?: string;
};

function readErrorMessage(error: unknown): string {
  if (!error) {
    return "Unknown error";
  }
  if (typeof error === "string") {
    return error;
  }
  if (error instanceof Error) {
    return error.message;
  }
  const maybe = error as ErrorWithMessage;
  return maybe.data?.error || maybe.error || "Request failed";
}

export function PersistenceHistorySection() {
  const { data: bootstrap, error: bootstrapError } = useGetPersistenceBootstrapQuery();
  const { data: sessions = [] } = useListPersistentSessionsQuery();
  const [createSession, createState] = useCreatePersistentSessionMutation();
  const [evaluateSession, evaluateState] = useEvaluatePersistentSessionMutation();
  const [deleteSession, deleteState] = useDeletePersistentSessionMutation();
  const [restoreSession, restoreState] = useRestorePersistentSessionMutation();
  const [selectedSessionID, setSelectedSessionID] = useState<string | null>(null);
  const historyArg = selectedSessionID ?? skipToken;
  const { data: history = [] } = useGetPersistentSessionHistoryQuery(historyArg);
  const { data: exported } = useGetPersistentSessionExportQuery(historyArg);

  useEffect(() => {
    if (!selectedSessionID && sessions.length > 0) {
      setSelectedSessionID(sessions[0]?.sessionId ?? null);
    }
  }, [selectedSessionID, sessions]);

  const statusText = useMemo(() => {
    if (createState.isLoading) {
      return "Creating and seeding a durable session...";
    }
    if (evaluateState.isLoading) {
      return "Writing persistent cell history...";
    }
    if (deleteState.isLoading) {
      return "Dropping the live in-memory session...";
    }
    if (restoreState.isLoading) {
      return "Restoring the live session from durable history...";
    }
    const error =
      createState.error ??
      evaluateState.error ??
      deleteState.error ??
      restoreState.error ??
      bootstrapError;
    if (error) {
      return readErrorMessage(error);
    }
    if (selectedSessionID) {
      return `Selected durable session ${selectedSessionID}. The history and export views below come from real raw API calls.`;
    }
    return "No durable session selected yet. Seed one to create real stored history, then inspect and restore it.";
  }, [
    bootstrapError,
    createState.error,
    createState.isLoading,
    deleteState.error,
    deleteState.isLoading,
    evaluateState.error,
    evaluateState.isLoading,
    restoreState.error,
    restoreState.isLoading,
    selectedSessionID
  ]);

  async function seedPersistentSession() {
    const session = await createSession().unwrap();
    for (const source of bootstrap?.seedSources ?? []) {
      await evaluateSession({ sessionId: session.id, source: source.source }).unwrap();
    }
    setSelectedSessionID(session.id);
  }

  return (
    <section className="essay-section-block essay-section-block--persistence">
      <Heading n="6">Persistence, History, and Restore</Heading>
      <Prose>
        Persistent mode changes the mental model from “temporary REPL” to “recoverable session”.
        The point is not only that values can be remembered. The point is that the session can be
        reconstructed later from durable records, with a real history and a recoverable identity.
      </Prose>
      <Prose>
        This section deliberately leans on the raw session routes. It seeds a durable session, lists
        persistent records, shows cell history, and exercises restore against the same underlying
        API that a non-article client would use.
      </Prose>
      <div className="essay-inline-status">
        <Typography
          as="p"
          variant="caption"
          tone={
            createState.error || evaluateState.error || deleteState.error || restoreState.error
              ? "danger"
              : "muted"
          }
        >
          {statusText}
        </Typography>
      </div>
      <div className="essay-persistence-panels">
        <Row>
          <Col>
          <Card
            className="essay-persistence-card essay-persistence-card--sessions"
            title="Durable Sessions"
            subtitle="These rows come from GET /api/sessions against the real persistent store."
            actions={
              <Button
                type="button"
                disabled={createState.isLoading || evaluateState.isLoading}
                onClick={async () => {
                  try {
                    await seedPersistentSession();
                  } catch {
                    // surfaced through RTK Query state
                  }
                }}
              >
                {createState.isLoading || evaluateState.isLoading ? "Seeding..." : "Seed Durable Session"}
              </Button>
            }
          >
            <div className="essay-session-list">
              {sessions.map((session) => (
                <button
                  key={session.sessionId}
                  type="button"
                  className="essay-session-list__row"
                  data-selected={selectedSessionID === session.sessionId}
                  onClick={() => setSelectedSessionID(session.sessionId)}
                >
                  <span className="essay-table__value">{session.sessionId}</span>
                  <span className="essay-meta-inline">{session.updatedAt}</span>
                </button>
              ))}
            </div>
          </Card>
          </Col>
          <Col>
          <Card
            className="essay-persistence-card essay-persistence-card--restore"
            title="Restore Controls"
            subtitle="Delete removes the live in-memory session. Restore rebuilds it from the durable store."
          >
            <div className="essay-card__action-row">
              <Button
                type="button"
                size="sm"
                disabled={!selectedSessionID || deleteState.isLoading}
                onClick={async () => {
                  if (!selectedSessionID) {
                    return;
                  }
                  try {
                    await deleteSession(selectedSessionID).unwrap();
                  } catch {
                    // surfaced through RTK Query state
                  }
                }}
              >
                {deleteState.isLoading ? "Killing..." : "Kill Live Session"}
              </Button>
              <Button
                type="button"
                size="sm"
                disabled={!selectedSessionID || restoreState.isLoading}
                onClick={async () => {
                  if (!selectedSessionID) {
                    return;
                  }
                  try {
                    await restoreSession(selectedSessionID).unwrap();
                  } catch {
                    // surfaced through RTK Query state
                  }
                }}
              >
                {restoreState.isLoading ? "Restoring..." : "Restore Session"}
              </Button>
            </div>
            <Callout>
              <strong>Contract:</strong> deletion of the live session should not erase durable
              history. Restore should rebuild a live session from the stored cells and metadata.
            </Callout>
          </Card>
          </Col>
        </Row>
      </div>
      <div className="essay-persistence-history">
        <Card
          className="essay-persistence-card"
          title={selectedSessionID ? `History — ${selectedSessionID}` : "History"}
          subtitle="The table below is the durable evaluation history, not a frontend reconstruction."
        >
          <div className="essay-table-scroll">
            <table className="essay-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>Source</th>
                  <th>Rewritten</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {history.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="essay-empty-note">
                      No stored evaluations yet.
                    </td>
                  </tr>
                ) : (
                  history.map((entry) => (
                    <tr key={entry.evaluationId}>
                      <td>{entry.cellId}</td>
                      <td className="essay-table__value">{entry.rawSource}</td>
                      <td className="essay-table__value">{entry.rewrittenSource}</td>
                      <td>{entry.ok ? "ok" : entry.errorText || "error"}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
      {exported ? (
        <div className="essay-persistence-export">
          <JsonViewer label="Raw SessionExport" data={exported} />
        </div>
      ) : null}
    </section>
  );
}
