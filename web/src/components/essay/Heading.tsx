import type { ReactNode } from "react";

type HeadingProps = {
  n: string | number;
  children: ReactNode;
};

export function Heading({ n, children }: HeadingProps) {
  return (
    <h2 className="essay-heading">
      <span className="essay-heading__number">{n}</span>
      <span>{children}</span>
    </h2>
  );
}
