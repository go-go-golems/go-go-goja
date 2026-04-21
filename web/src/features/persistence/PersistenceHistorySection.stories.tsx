import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { withEssayProviders } from "@/storybook/withEssayProviders";
import {
  persistenceBootstrapFixture,
  sessionExportFixture
} from "@/features/meet-session/storyFixtures";
import { PersistenceHistorySection } from "@/features/persistence/PersistenceHistorySection";

const handlers = [
  http.get("/api/essay/sections/persistence-history-and-restore", () =>
    HttpResponse.json(persistenceBootstrapFixture)
  ),
  http.get("/api/sessions", () =>
    HttpResponse.json({ sessions: [sessionExportFixture.session] })
  ),
  http.post("/api/sessions", () =>
    HttpResponse.json({ session: { id: "session-durable-1", profile: "persistent", createdAt: "2026-04-15T04:20:00Z", cellCount: 0, bindingCount: 0, policy: { eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true }, observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true, jsdocExtraction: true }, persist: { enabled: true, sessions: true, evaluations: true, bindingVersions: true, bindingDocs: true } } } }, { status: 201 })
  ),
  http.post("/api/sessions/:sessionID/evaluate", () =>
    HttpResponse.json({
      session: { id: "session-durable-1", profile: "persistent", createdAt: "2026-04-15T04:20:00Z", cellCount: 1, bindingCount: 1, policy: { eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true }, observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true, jsdocExtraction: true }, persist: { enabled: true, sessions: true, evaluations: true, bindingVersions: true, bindingDocs: true } } },
      cell: { id: 1, execution: { status: "ok", result: "1", durationMs: 1, awaited: false, console: [], hadSideEffects: false, helperError: false }, rewrite: { transformedSource: "var x = 1; x", operations: [], mode: "instrumented", declaredNames: [], helperNames: [], lastHelperName: "", bindingHelperName: "", capturedLastExpr: true }, static: { diagnostics: [], topLevelBindings: [], unresolved: [], astNodeCount: 1, summary: [] }, runtime: { diffs: [], newBindings: ["x"], updatedBindings: [], removedBindings: [], leakedGlobals: [], persistedByWrap: ["x"], currentCellValue: "1" }, createdAt: "2026-04-15T04:20:01Z", source: "const x = 1; x" }
    })
  ),
  http.get("/api/sessions/:sessionID/history", () =>
    HttpResponse.json({ history: sessionExportFixture.evaluations })
  ),
  http.get("/api/sessions/:sessionID/export", () =>
    HttpResponse.json(sessionExportFixture)
  ),
  http.delete("/api/sessions/:sessionID", () =>
    HttpResponse.json({ deleted: true })
  ),
  http.post("/api/sessions/:sessionID/restore", () =>
    HttpResponse.json({ session: { id: "session-durable-1", profile: "persistent", createdAt: "2026-04-15T04:20:00Z", cellCount: 2, bindingCount: 2, policy: { eval: { mode: "instrumented", timeoutMs: 5000, captureLastExpression: true, supportTopLevelAwait: true }, observe: { staticAnalysis: true, runtimeSnapshot: true, bindingTracking: true, consoleCapture: true, jsdocExtraction: true }, persist: { enabled: true, sessions: true, evaluations: true, bindingVersions: true, bindingDocs: true } } } })
  )
];

const meta = {
  title: "Features/Persistence/PersistenceHistorySection",
  component: PersistenceHistorySection,
  decorators: [withEssayProviders()],
  parameters: {
    msw: {
      handlers
    }
  }
} satisfies Meta<typeof PersistenceHistorySection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
