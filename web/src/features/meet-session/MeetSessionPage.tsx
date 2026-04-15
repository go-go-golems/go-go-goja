import { skipToken } from "@reduxjs/toolkit/query";
import {
  useCreateMeetSessionMutation,
  useGetMeetSessionBootstrapQuery,
  useGetMeetSessionSnapshotQuery
} from "@/app/api/essayApi";
import { useAppDispatch, useAppSelector } from "@/app/hooks";
import { Button, Divider, Typography } from "@/components/primitives";
import { PolicyCard } from "@/features/meet-session/components/PolicyCard";
import { EssayMasthead } from "@/features/meet-session/components/EssayMasthead";
import { AboutEssayCallout } from "@/features/meet-session/components/AboutEssayCallout";
import { MeetSessionFieldGuide } from "@/features/meet-session/components/MeetSessionFieldGuide";
import { SectionIntro } from "@/features/meet-session/components/SectionIntro";
import { SectionShell } from "@/features/meet-session/components/SectionShell";
import { SessionJsonPanel } from "@/features/meet-session/components/SessionJsonPanel";
import { SessionSummaryCard } from "@/features/meet-session/components/SessionSummaryCard";
import { setActiveSessionId } from "@/features/meet-session/meetSessionSlice";
import type { SessionSummary } from "@/features/meet-session/types";
import { ProfileComparisonSection } from "@/features/profile-comparison/ProfileComparisonSection";
import { CodeEvaluationSection } from "@/features/code-flow/CodeEvaluationSection";

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

function resolveSession(
  createdSession: SessionSummary | undefined,
  snapshotSession: SessionSummary | undefined
) {
  return snapshotSession || createdSession || null;
}

export function MeetSessionPage() {
  const dispatch = useAppDispatch();
  const activeSessionId = useAppSelector((state) => state.meetSession.activeSessionId);

  const {
    data: bootstrap,
    isLoading: isBootstrapLoading,
    error: bootstrapError
  } = useGetMeetSessionBootstrapQuery();

  const [createSession, createState] = useCreateMeetSessionMutation();
  const snapshotArg = activeSessionId ?? skipToken;
  const {
    data: snapshotSession,
    isFetching: isSnapshotFetching,
    error: snapshotError
  } = useGetMeetSessionSnapshotQuery(snapshotArg, {
    refetchOnFocus: true
  });

  const session = resolveSession(createState.data, snapshotSession);

  const section = bootstrap?.section;
  const title = section?.title || "Meet a Session";
  const summary =
    section?.summary ||
    "Create one real REPL session and inspect how identity, policy, and backend state fit together.";
  const intro = section?.intro || [];

  let statusTone: "muted" | "success" | "danger" = "muted";
  let statusText = activeSessionId
    ? `Tracking live session ${activeSessionId}. The browser will use this ID to request later snapshots.`
    : "No session exists yet. Press Create Session to ask the backend for one new persistent session.";

  if (createState.isLoading) {
    statusText = "Creating live session...";
  } else if (isSnapshotFetching && activeSessionId) {
    statusText = `Refreshing session ${activeSessionId}...`;
  } else if (createState.isError) {
    statusTone = "danger";
    statusText = readErrorMessage(createState.error);
  } else if (snapshotError) {
    statusTone = "danger";
    statusText = readErrorMessage(snapshotError);
  } else if (bootstrapError) {
    statusTone = "danger";
    statusText = readErrorMessage(bootstrapError);
  } else if (session) {
    statusTone = "success";
    statusText = `Created live session ${session.id}. The cards below are now rendering real backend data.`;
  }

  const createButtonLabel = section?.primaryAction.label || "Create Session";

  return (
    <SectionShell
      header={
        <>
          <EssayMasthead />
          <AboutEssayCallout />
          <Divider />
          <SectionIntro title={title} summary={summary} introParagraphs={intro} />
          <Divider />
        </>
      }
      actions={
        <div className="essay-action-row">
          <Button
            type="button"
            disabled={createState.isLoading || isBootstrapLoading}
            onClick={async () => {
              try {
                const created = await createSession().unwrap();
                dispatch(setActiveSessionId(created.id));
              } catch {
                // Errors are surfaced from mutation state.
              }
            }}
          >
            {createState.isLoading ? "Creating..." : createButtonLabel}
          </Button>
          <Typography as="p" variant="caption" tone={statusTone}>
            {statusText}
          </Typography>
        </div>
      }
      left={<SessionSummaryCard session={session} />}
      right={<PolicyCard policy={session?.policy || bootstrap?.defaultView.policy || null} />}
      footer={
        <>
          <SessionJsonPanel session={session} />
          <MeetSessionFieldGuide bootstrap={bootstrap} session={session} />
          <Divider />
          <ProfileComparisonSection />
          <Divider />
          <CodeEvaluationSection />
        </>
      }
    />
  );
}
