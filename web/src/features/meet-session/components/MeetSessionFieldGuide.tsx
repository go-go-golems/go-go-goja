import type { BootstrapResponse, SessionSummary } from "@/features/meet-session/types";
import { ApiReferenceSection } from "@/features/meet-session/components/ApiReferenceSection";
import { FieldsMatterSection } from "@/features/meet-session/components/FieldsMatterSection";
import { HappyPathSection } from "@/features/meet-session/components/HappyPathSection";
import { HowToReadSection } from "@/features/meet-session/components/HowToReadSection";
import { MentalModelSection } from "@/features/meet-session/components/MentalModelSection";
import { PolicyGuideSection } from "@/features/meet-session/components/PolicyGuideSection";
import { SourceFileGuideSection } from "@/features/meet-session/components/SourceFileGuideSection";
import { ValidationExercisesSection } from "@/features/meet-session/components/ValidationExercisesSection";

type MeetSessionFieldGuideProps = {
  bootstrap: BootstrapResponse | undefined;
  session: SessionSummary | null;
};

export function MeetSessionFieldGuide({ bootstrap, session }: MeetSessionFieldGuideProps) {
  const sessionID = session?.id ?? null;

  return (
    <article className="essay-article" data-part="meet-session-field-guide">
      <HowToReadSection />
      <MentalModelSection sessionID={sessionID} />
      <FieldsMatterSection />
      <HappyPathSection sessionID={sessionID} />
      <PolicyGuideSection />
      <ApiReferenceSection bootstrap={bootstrap} session={session} />
      <SourceFileGuideSection />
      <ValidationExercisesSection />
    </article>
  );
}
