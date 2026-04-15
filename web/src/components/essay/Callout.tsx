import type { ReactNode } from "react";

type CalloutProps = {
  children: ReactNode;
};

export function Callout({ children }: CalloutProps) {
  return <section className="essay-callout">{children}</section>;
}
