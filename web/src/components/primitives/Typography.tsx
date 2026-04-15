import clsx from "clsx";
import type { ElementType, ReactNode } from "react";

type TypographyVariant = "display" | "title" | "section" | "body" | "caption" | "mono";
type TypographyTone = "default" | "muted" | "accent" | "success" | "danger";

type TypographyProps<T extends ElementType> = {
  as?: T;
  children: ReactNode;
  className?: string;
  variant?: TypographyVariant;
  tone?: TypographyTone;
  unstyled?: boolean;
};

export function Typography<T extends ElementType = "p">({
  as,
  children,
  className,
  variant = "body",
  tone = "default",
  unstyled = false
}: TypographyProps<T>) {
  const Component = as ?? "p";

  if (unstyled) {
    return <Component className={className}>{children}</Component>;
  }

  return (
    <Component
      className={clsx("essay-typo", className)}
      data-part="typography"
      data-variant={variant}
      data-tone={tone}
    >
      {children}
    </Component>
  );
}
