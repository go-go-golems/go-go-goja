import { Callout } from "@/components/essay/Callout";

export function AboutEssayCallout() {
  return (
    <Callout>
      <p className="essay-callout__body">
        <strong>About this essay.</strong> Every section teaches through real feedback. When you do
        something, the page reveals multiple synchronized views of the same event: the friendly
        explanation, the compact summary, and the exact backend payload. This is mock data — the
        real version will talk to <code>goja-repl serve</code>.
      </p>
    </Callout>
  );
}
