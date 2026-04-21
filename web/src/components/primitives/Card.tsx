import clsx from "clsx";
import type { ReactNode } from "react";

export type CardProps = {
  title?: ReactNode;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  unstyled?: boolean;
};

export function Card({
  title,
  subtitle,
  actions,
  children,
  className,
  unstyled = false
}: CardProps) {
  if (unstyled) {
    return <section className={className}>{children}</section>;
  }

  return (
    <section className={clsx("essay-card", className)} data-part="card">
      {(title || actions) && (
        <header className="essay-card__header" data-part="card-header">
          <div className="essay-card__title-wrap">
            {title && (
              <h2 className="essay-card__title" data-part="card-title">
                {title}
              </h2>
            )}
          </div>
          {actions && (
            <div className="essay-card__actions" data-part="card-actions">
              {actions}
            </div>
          )}
        </header>
      )}
      <div className="essay-card__body" data-part="card-body">
        {subtitle && (
          <p className="essay-card__subtitle" data-part="card-subtitle">
            {subtitle}
          </p>
        )}
        {children}
      </div>
    </section>
  );
}
