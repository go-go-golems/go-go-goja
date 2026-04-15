import clsx from "clsx";

type DividerProps = {
  className?: string;
  unstyled?: boolean;
};

export function Divider({ className, unstyled = false }: DividerProps) {
  if (unstyled) {
    return <hr className={className} />;
  }

  return <hr className={clsx("essay-divider", className)} data-part="divider" />;
}
