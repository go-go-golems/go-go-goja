import { Typography } from "@/components/primitives/Typography";

export function EssayMasthead() {
  return (
    <header className="essay-masthead" data-part="essay-masthead">
      <Typography as="h1" variant="display">
        ▪ The REPL Essay ▪
      </Typography>
      <Typography as="p" variant="caption" tone="muted" className="essay-masthead__subtitle">
        An interactive guide to the session model, evaluation pipeline, and persistence layer
      </Typography>
      <div className="essay-masthead__rules" aria-hidden="true">
        <div />
        <div />
      </div>
      <Typography as="p" variant="mono" tone="muted" className="essay-masthead__meta">
        GOJA-043 · go-go-goja · {new Date().toLocaleDateString("en-US")}
      </Typography>
    </header>
  );
}
