import clsx from "clsx";
import type { ReactNode } from "react";

type TagVariant = "neutral" | "ok" | "warning" | "danger" | "accent";

type TagProps = {
  children: ReactNode;
  variant?: TagVariant;
  className?: string;
  unstyled?: boolean;
};

export function Tag({
  children,
  variant = "neutral",
  className,
  unstyled = false
}: TagProps) {
  if (unstyled) {
    return <span className={className}>{children}</span>;
  }

  return (
    <span className={clsx("essay-tag", className)} data-part="tag" data-variant={variant}>
      {children}
    </span>
  );
}
