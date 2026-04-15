import type { BootstrapResponse, SessionSummary } from "@/features/meet-session/types";
import { resolveRoutes } from "@/features/meet-session/components/fieldGuideData";

type ApiReferenceSectionProps = {
  bootstrap: BootstrapResponse | undefined;
  session: SessionSummary | null;
};

export function ApiReferenceSection({ bootstrap, session }: ApiReferenceSectionProps) {
  const routes = resolveRoutes(bootstrap, session);

  return (
    <section className="essay-article__section">
      <h2>API Reference</h2>
      <p className="essay-prose">
        These are the main routes involved in Section 1. The article-specific routes are
        intentionally narrow wrappers over the underlying REPL API.
      </p>
      <table className="essay-reference-table">
        <thead>
          <tr>
            <th>Method</th>
            <th>Path</th>
            <th>Purpose</th>
          </tr>
        </thead>
        <tbody>
          {routes.map((route) => (
            <tr key={`${route.method}:${route.path}`}>
              <td>
                <code>{route.method}</code>
              </td>
              <td>
                <code>{route.resolvedPath}</code>
              </td>
              <td>{route.purpose}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
