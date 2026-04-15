import { CodeBlock } from "@/components/primitives/CodeBlock";
import { buildCreateSessionPseudocode } from "@/features/meet-session/components/fieldGuideData";

type HappyPathSectionProps = {
  sessionID: string | null;
};

export function HappyPathSection({ sessionID }: HappyPathSectionProps) {
  return (
    <section className="essay-article__section">
      <h2>Pseudocode For The Happy Path</h2>
      <p className="essay-prose">
        The most important thing to notice is that the frontend stores only a small amount of
        state locally. The durable state lives in the backend session object. The browser keeps the
        session ID so it can ask for a fresh snapshot rather than pretending it is the source of
        truth.
      </p>
      <CodeBlock>{buildCreateSessionPseudocode(sessionID)}</CodeBlock>
    </section>
  );
}
