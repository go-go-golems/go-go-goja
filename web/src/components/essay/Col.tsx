import type { CSSProperties, ReactNode } from "react";

type ColProps = {
  children: ReactNode;
  style?: CSSProperties;
};

export function Col({ children, style }: ColProps) {
  return (
    <div className="essay-col" style={style}>
      {children}
    </div>
  );
}
