import { useEffect, useMemo, useState } from "react";
import {
  useCreateCodeFlowSessionMutation,
  useEvaluateCodeFlowMutation,
  useGetCodeFlowBootstrapQuery
} from "@/app/api/essayApi";
import { Callout, Heading, Prose } from "@/components/essay";
import { Button, Card, JsonViewer, Typography } from "@/components/primitives";
import { SessionSummaryCard } from "@/features/meet-session/components/SessionSummaryCard";
import type { EvaluateResponse } from "@/features/meet-session/types";
import { BindingsMemorySection } from "@/features/code-flow/BindingsMemorySection";
import { ExecutionResultSummary } from "@/features/code-flow/ExecutionResultSummary";
import { RewriteOperationsList } from "@/features/code-flow/RewriteOperationsList";
import { SourceTransformPanels } from "@/features/code-flow/SourceTransformPanels";
import { StaticRuntimeRealitySection } from "@/features/code-flow/StaticRuntimeRealitySection";

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

export function CodeEvaluationSection() {
  const { data: bootstrap, isLoading, error } = useGetCodeFlowBootstrapQuery();
  const [createSession, createState] = useCreateCodeFlowSessionMutation();
  const [evaluate, evaluateState] = useEvaluateCodeFlowMutation();
  const [source, setSource] = useState("");
  const [sessionID, setSessionID] = useState<string | null>(null);
  const [response, setResponse] = useState<EvaluateResponse | null>(null);

  useEffect(() => {
    if (bootstrap && !source) {
      setSource(bootstrap.starterSource);
    }
  }, [bootstrap, source]);

  const statusText = useMemo(() => {
    if (createState.isLoading) {
      return "Preparing a live interactive evaluation session...";
    }
    if (evaluateState.isLoading) {
      return "Submitting source to the live evaluation pipeline...";
    }
    if (createState.isError) {
      return readErrorMessage(createState.error);
    }
    if (evaluateState.isError) {
      return readErrorMessage(evaluateState.error);
    }
    if (response) {
      return `Cell ${response.cell.id} finished with status ${response.cell.execution.status}. The reports below are real backend output.`;
    }
    if (sessionID) {
      return `Evaluation session ${sessionID} is ready. Edit the source, then run one live evaluation.`;
    }
    return "No live evaluation session yet. Preparing a session first keeps the request flow explicit for the reader.";
  }, [
    createState.error,
    createState.isError,
    createState.isLoading,
    evaluateState.error,
    evaluateState.isError,
    evaluateState.isLoading,
    response,
    sessionID
  ]);

  async function ensureSession(): Promise<string | null> {
    if (sessionID) {
      return sessionID;
    }
    const session = await createSession(bootstrap?.defaultProfile).unwrap();
    setSessionID(session.id);
    return session.id;
  }

  return (
    <section className="essay-section-block">
      <Heading n="3">What Happened To My Code?</Heading>
      <Prose>
        Evaluation in the new REPL is a pipeline. The system can inspect the source before
        execution, decide whether helper insertion is needed, rewrite declarations, capture the last
        expression, run the transformed code, and then compare the runtime state before and after the
        cell. The purpose of this section is to make those hidden steps visible.
      </Prose>
      <Prose>
        The source editor below talks to a real session. When you press Evaluate, the browser sends
        the source through the article-scoped route, which in turn calls the live session evaluation
        API. The transformed source, rewrite operations, execution status, and runtime diffs you see
        underneath all come from the backend response rather than from frontend guesswork.
      </Prose>
      <Card
        title="Source Editor"
        subtitle="Choose one of the suggested examples or edit the source directly. The section keeps one live evaluation session so later submissions build on earlier state."
        actions={
          <div className="essay-card__action-row">
            <Button
              type="button"
              disabled={isLoading || createState.isLoading || evaluateState.isLoading}
              onClick={async () => {
                try {
                  const session = await createSession(bootstrap?.defaultProfile).unwrap();
                  setSessionID(session.id);
                } catch {
                  // surfaced through RTK Query state
                }
              }}
            >
              {createState.isLoading ? "Preparing..." : "Prepare Session"}
            </Button>
            <Button
              type="button"
              disabled={!source.trim() || isLoading || createState.isLoading || evaluateState.isLoading}
              onClick={async () => {
                try {
                  const ensuredID = await ensureSession();
                  if (!ensuredID) {
                    return;
                  }
                  const next = await evaluate({ sessionId: ensuredID, source }).unwrap();
                  setResponse(next);
                  setSessionID(next.session.id);
                } catch {
                  // surfaced through RTK Query state
                }
              }}
            >
              {evaluateState.isLoading ? "Evaluating..." : "Evaluate Source"}
            </Button>
          </div>
        }
      >
        <div className="essay-example-row">
          {bootstrap?.examples.map((example) => (
            <Button
              key={example.id}
              type="button"
              size="sm"
              variant="secondary"
              onClick={() => setSource(example.source)}
            >
              {example.label}
            </Button>
          ))}
        </div>
        <textarea
          className="essay-source-editor"
          value={source}
          onChange={(event) => setSource(event.target.value)}
          rows={4}
          spellCheck={false}
        />
        <Typography as="p" variant="caption" tone={createState.isError || evaluateState.isError ? "danger" : "muted"}>
          {error ? readErrorMessage(error) : statusText}
        </Typography>
      </Card>
      <Callout>
        <strong>API note:</strong> the article still leans on the real session model. It prepares a
        live session first, then posts source to <code>/api/sessions/:id/evaluate</code> through a
        thin article route so the UI can stay explicit and readable.
      </Callout>
      {response ? (
        <>
          <SessionSummaryCard session={response.session} />
          <SourceTransformPanels
            source={response.cell.source}
            transformedSource={response.cell.rewrite.transformedSource}
          />
          <RewriteOperationsList operations={response.cell.rewrite.operations} />
          <ExecutionResultSummary execution={response.cell.execution} />
          <Card
            title="Static Summary"
            subtitle="These facts come from analysis before runtime effects are applied."
          >
            <ul className="essay-bullet-list">
              {response.cell.static.summary.map((fact) => (
                <li key={`${fact.label}-${fact.value}`}>
                  <strong>{fact.label}:</strong> {fact.value}
                </li>
              ))}
            </ul>
          </Card>
          <JsonViewer label="Raw EvaluateResponse" data={response} />
        </>
      ) : null}
      <StaticRuntimeRealitySection response={response} />
      <BindingsMemorySection session={response?.session ?? null} />
    </section>
  );
}
