import clsx from "clsx";
import type { ReactNode } from "react";

type CodeBlockProps = {
  children: ReactNode;
  className?: string;
  unstyled?: boolean;
};

export function CodeBlock({ children, className, unstyled = false }: CodeBlockProps) {
  if (unstyled) {
    return (
      <pre className={className}>
        <code>{children}</code>
      </pre>
    );
  }

  return (
    <pre className={clsx("essay-code", className)} data-part="code-block">
      <code>{children}</code>
    </pre>
  );
}
