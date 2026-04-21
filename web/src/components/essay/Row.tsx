import type { CSSProperties, ReactNode } from "react";

type RowProps = {
  children: ReactNode;
  style?: CSSProperties;
};

export function Row({ children, style }: RowProps) {
  return (
    <div className="essay-row" style={style}>
      {children}
    </div>
  );
}
