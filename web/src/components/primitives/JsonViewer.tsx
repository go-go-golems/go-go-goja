import { useState } from "react";
import { Button } from "@/components/primitives/Button";
import { CodeBlock } from "@/components/primitives/CodeBlock";

type JsonViewerProps = {
  data: unknown;
  label?: string;
  collapsedByDefault?: boolean;
};

export function JsonViewer({
  data,
  label = "Raw JSON",
  collapsedByDefault = true
}: JsonViewerProps) {
  const [open, setOpen] = useState(!collapsedByDefault);

  return (
    <div data-part="json-viewer">
      <div className="mb-2">
        <Button
          variant="ghost"
          size="sm"
          type="button"
          onClick={() => setOpen((prev) => !prev)}
        >
          {open ? "Hide" : "Show"} {label}
        </Button>
      </div>
      {open && <CodeBlock>{JSON.stringify(data, null, 2)}</CodeBlock>}
    </div>
  );
}
