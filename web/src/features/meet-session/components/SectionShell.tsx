import type { ReactNode } from "react";

type SectionShellProps = {
  header: ReactNode;
  actions: ReactNode;
  left: ReactNode;
  right: ReactNode;
  footer: ReactNode;
};

export function SectionShell({ header, actions, left, right, footer }: SectionShellProps) {
  return (
    <main className="essay-page" data-part="essay-page">
      {header}
      <section className="essay-page__actions">{actions}</section>
      <section className="essay-page__grid">
        <div>{left}</div>
        <div>{right}</div>
      </section>
      <section className="essay-page__footer">{footer}</section>
    </main>
  );
}
