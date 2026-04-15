import clsx from "clsx";
import type { ReactNode } from "react";

type ProseProps = {
  children: ReactNode;
  className?: string;
};

export function Prose({ children, className }: ProseProps) {
  return <p className={clsx("essay-prose", className)}>{children}</p>;
}
