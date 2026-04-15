import { CodeBlock } from "@/components/primitives/CodeBlock";
import { buildRequestFlowDiagram } from "@/features/meet-session/components/fieldGuideData";

type MentalModelSectionProps = {
  sessionID: string | null;
};

export function MentalModelSection({ sessionID }: MentalModelSectionProps) {
  return (
    <section className="essay-article__section">
      <h2>Mental Model And Request Flow</h2>
      <p className="essay-prose">
        In this section, the browser is acting as a careful observer of the REPL system. When you
        press <code>Create Session</code>, the browser issues an HTTP request. The article handler
        receives that request, asks the real REPL application to create a session using the default
        persistent profile, and returns a <code>SessionSummary</code>. The browser then keeps the
        returned session ID and can request a later snapshot of the same session.
      </p>
      <p className="essay-prose">
        That means this page is already teaching an important architectural idea: the UI never
        needs privileged knowledge of the runtime internals. It succeeds by understanding the
        public API contract, then rendering that contract faithfully.
      </p>
      <CodeBlock>{buildRequestFlowDiagram(sessionID)}</CodeBlock>
    </section>
  );
}
