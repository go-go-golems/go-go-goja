import { useMemo, useState } from "react";
import {
  useCreateProfileSessionMutation,
  useGetProfilesBootstrapQuery
} from "@/app/api/essayApi";
import { Callout, Heading, Prose, Row, Col } from "@/components/essay";
import { Button, Card, Typography } from "@/components/primitives";
import { PolicyCard } from "@/features/meet-session/components/PolicyCard";
import { SessionJsonPanel } from "@/features/meet-session/components/SessionJsonPanel";
import { SessionSummaryCard } from "@/features/meet-session/components/SessionSummaryCard";
import type { ProfileName } from "@/features/meet-session/types";
import { ProfileComparisonTable } from "@/features/profile-comparison/ProfileComparisonTable";
import { ProfileSelector } from "@/features/profile-comparison/ProfileSelector";

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

export function ProfileComparisonSection() {
  const { data: bootstrap, isLoading, error } = useGetProfilesBootstrapQuery();
  const [selectedProfile, setSelectedProfile] = useState<ProfileName>("interactive");
  const [createSession, createState] = useCreateProfileSessionMutation();

  const profiles = bootstrap?.profiles ?? [];
  const activeProfile =
    profiles.find((profile) => profile.id === selectedProfile) ?? profiles[0] ?? null;
  const liveSession = createState.data ?? null;

  const statusText = useMemo(() => {
    if (createState.isLoading) {
      return `Creating a live ${selectedProfile} session...`;
    }
    if (createState.isError) {
      return readErrorMessage(createState.error);
    }
    if (liveSession) {
      return `Created live ${liveSession.profile} session ${liveSession.id}. The backend summary below should match the selected contract.`;
    }
    return "Choose a profile, then create one real session to verify the contract against the backend.";
  }, [createState.error, createState.isError, createState.isLoading, liveSession, selectedProfile]);

  return (
    <section className="essay-section-block">
      <Heading n="2">Profiles Change Behavior</Heading>
      <Prose>
        The three built-in profiles are different behavioral contracts. A raw session aims to stay
        close to plain goja execution. An interactive session adds the extra machinery that makes a
        REPL feel helpful instead of brittle. A persistent session keeps the same interactive
        behavior but adds durable storage so the session can survive process restarts.
      </Prose>
      <Prose>
        That is why this section is worth validating with the real API. If the selected profile says
        it supports top-level await, captures the last expression, or persists history, the session
        summary returned by the backend should make that policy visible immediately.
      </Prose>
      <ProfileSelector
        profiles={profiles}
        selectedProfile={selectedProfile}
        onSelect={setSelectedProfile}
      />
      <div className="essay-inline-status">
        <Typography as="p" variant="caption" tone={createState.isError ? "danger" : "muted"}>
          {error ? readErrorMessage(error) : statusText}
        </Typography>
      </div>
      {profiles.length > 0 && (
        <Card
          title="Profile Comparison"
          subtitle="The highlighted column is the currently selected profile. The purpose of the table is to show that the profile selector changes policy, not just labels."
        >
          <ProfileComparisonTable profiles={profiles} selectedProfile={selectedProfile} />
        </Card>
      )}
      {activeProfile && (
        <Row>
          <Col>
            <Card
              title={`Profile: ${activeProfile.title}`}
              subtitle={activeProfile.summary}
              actions={
                <Button
                  type="button"
                  disabled={isLoading || createState.isLoading}
                  onClick={async () => {
                    try {
                      await createSession(selectedProfile).unwrap();
                    } catch {
                      // surfaced through RTK Query state
                    }
                  }}
                >
                  {createState.isLoading ? "Creating..." : "Create Selected Profile"}
                </Button>
              }
            >
              <ul className="essay-bullet-list">
                {activeProfile.highlights.map((highlight) => (
                  <li key={highlight}>{highlight}</li>
                ))}
              </ul>
              <Prose className="essay-prose--after-table">
                If the page creates a live session using this profile, the policy card next to the
                summary should echo the same machine-readable decisions shown in the table above.
              </Prose>
            </Card>
          </Col>
          <Col>
            <PolicyCard policy={activeProfile.policy} />
          </Col>
        </Row>
      )}
      <Callout>
        <strong>Backend note:</strong> this section uses an article-only profile override route. The
        general <code>/api/sessions</code> endpoint still defaults to the app profile when called
        directly.
      </Callout>
      {liveSession && (
        <>
          <Row>
            <Col>
              <SessionSummaryCard session={liveSession} />
            </Col>
            <Col>
              <PolicyCard policy={liveSession.policy} />
            </Col>
          </Row>
          <SessionJsonPanel session={liveSession} />
        </>
      )}
    </section>
  );
}
