import clsx from "clsx";
import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "ghost";
type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  unstyled?: boolean;
};

export function Button({
  children,
  className,
  variant = "primary",
  size = "md",
  unstyled = false,
  ...props
}: ButtonProps) {
  if (unstyled) {
    return (
      <button className={className} {...props}>
        {children}
      </button>
    );
  }

  return (
    <button
      className={clsx("essay-btn", className)}
      data-part="button"
      data-variant={variant}
      data-size={size}
      {...props}
    >
      {children}
    </button>
  );
}
