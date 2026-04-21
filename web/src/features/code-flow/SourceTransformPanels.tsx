import { Card, CodeBlock } from "@/components/primitives";

type SourceTransformPanelsProps = {
  source: string;
  transformedSource: string;
};

export function SourceTransformPanels({
  source,
  transformedSource
}: SourceTransformPanelsProps) {
  return (
    <div className="essay-source-pair">
      <Card title="Original Source">
        <CodeBlock>{source}</CodeBlock>
      </Card>
      <div className="essay-source-pair__arrow" aria-hidden="true">
        →
      </div>
      <Card title="Transformed Source">
        <CodeBlock>{transformedSource}</CodeBlock>
      </Card>
    </div>
  );
}
