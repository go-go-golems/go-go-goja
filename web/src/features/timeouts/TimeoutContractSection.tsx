import { useMemo, useState } from "react";
import {
  useCreateCodeFlowSessionMutation,
  useEvaluateCodeFlowMutation,
  useGetTimeoutBootstrapQuery
} from "@/app/api/essayApi";
import { Callout, Heading, Prose } from "@/components/essay";
import { Button, Card, Typography } from "@/components/primitives";
import type { EvaluateResponse } from "@/features/meet-session/types";

type ErrorWithMessage = {
  data?: {
    error?: string;
  };
  error?: string;
};

type TimelineEvent = {
  label: string;
  source: string;
  status: string;
  result: string;
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

export function TimeoutContractSection() {
  const { data: bootstrap, error } = useGetTimeoutBootstrapQuery();
  const [createSession, createState] = useCreateCodeFlowSessionMutation();
  const [evaluate, evaluateState] = useEvaluateCodeFlowMutation();
  const [sessionID, setSessionID] = useState<string | null>(null);
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [latest, setLatest] = useState<EvaluateResponse | null>(null);

  async function ensureSession(): Promise<string | null> {
    if (sessionID) {
      return sessionID;
    }
    const session = await createSession("interactive").unwrap();
    setSessionID(session.id);
    return session.id;
  }

  const statusText = useMemo(() => {
    if (createState.isLoading) {
      return "Preparing a live interactive session for timeout scenarios...";
    }
    if (evaluateState.isLoading) {
      return "Running a live timeout scenario...";
    }
    if (createState.isError) {
      return readErrorMessage(createState.error);
    }
    if (evaluateState.isError) {
      return readErrorMessage(evaluateState.error);
    }
    if (latest) {
      return `Latest scenario finished with status ${latest.cell.execution.status}. The same session id is reused across scenarios.`;
    }
    return "No timeout scenario has run yet. The same session should survive both timeout scenarios and the later recovery proof.";
  }, [
    createState.error,
    createState.isError,
    createState.isLoading,
    evaluateState.error,
    evaluateState.isError,
    evaluateState.isLoading,
    latest
  ]);

  return (
    <section className="essay-section-block">
      <Heading n="7">Timeouts Are Part of the Contract</Heading>
      <Prose>
        A timeout is not only an error condition. In this REPL it is part of the session contract.
        Runaway code should be interrupted, the cell should report timeout status, and the next cell
        should still evaluate successfully in the same session.
      </Prose>
      <Prose>
        This section reuses the real evaluation path. The scenario buttons all target one live
        interactive session so the recovery proof is meaningful instead of theatrical.
      </Prose>
      <Card
        title="Canned Scenarios"
        subtitle="Run the timeout scenarios first, then use the recovery scenario to prove that the session still works."
      >
        <div className="essay-example-row">
          {bootstrap?.scenarios.map((scenario) => (
            <Button
              key={scenario.id}
              type="button"
              size="sm"
              variant="secondary"
              disabled={createState.isLoading || evaluateState.isLoading}
              onClick={async () => {
                try {
                  const ensuredID = await ensureSession();
                  if (!ensuredID) {
                    return;
                  }
                  const response = await evaluate({
                    sessionId: ensuredID,
                    source: scenario.source
                  }).unwrap();
                  setLatest(response);
                  setEvents((previous) => [
                    ...previous,
                    {
                      label: scenario.label,
                      source: scenario.source,
                      status: response.cell.execution.status,
                      result: response.cell.execution.result || response.cell.execution.error || "—"
                    }
                  ]);
                } catch {
                  // surfaced through RTK Query state
                }
              }}
            >
              {scenario.label}
            </Button>
          ))}
        </div>
        <Typography
          as="p"
          variant="caption"
          tone={createState.isError || evaluateState.isError || error ? "danger" : "muted"}
        >
          {error ? readErrorMessage(error) : statusText}
        </Typography>
      </Card>
      {events.length > 0 ? (
        <Card
          title="Execution Timeline"
          subtitle="This is the observed sequence of scenario outcomes on one live session."
        >
          <div className="essay-operation-list">
            {events.map((event, index) => (
              <div key={`${event.label}-${index}`} className="essay-operation-list__row">
                <strong>{event.label}</strong>
                <span className="essay-meta-inline">{event.source}</span>
                <span className="essay-result-grid__value">{event.status}</span>
                <span className="essay-table__value">{event.result}</span>
              </div>
            ))}
          </div>
          {latest?.cell.execution.status === "timeout" ? (
            <Callout>
              <strong>Next step:</strong> run <code>1 + 1</code> now. The important proof is that
              the session remains usable after the timeout.
            </Callout>
          ) : null}
        </Card>
      ) : null}
    </section>
  );
}
