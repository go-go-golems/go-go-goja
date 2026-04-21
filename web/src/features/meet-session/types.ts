export type RouteRef = {
  method: string;
  path: string;
  purpose: string;
};

export type FieldSpec = {
  label: string;
  jsonPath: string;
  description: string;
};

export type PanelSpec = {
  id: string;
  title: string;
  kind: string;
  description: string;
  fields?: FieldSpec[];
};

export type ActionSpec = {
  label: string;
  method: string;
  path: string;
};

export type SectionSpec = {
  id: string;
  title: string;
  summary: string;
  intro: string[];
  primaryAction: ActionSpec;
  panels: PanelSpec[];
};

export type SessionPolicy = {
  eval: {
    mode: string;
    timeoutMs: number;
    captureLastExpression: boolean;
    supportTopLevelAwait: boolean;
  };
  observe: {
    staticAnalysis: boolean;
    runtimeSnapshot: boolean;
    bindingTracking: boolean;
    consoleCapture: boolean;
    jsdocExtraction: boolean;
  };
  persist: {
    enabled: boolean;
    sessions: boolean;
    evaluations: boolean;
    bindingVersions: boolean;
    bindingDocs: boolean;
  };
};

export type DefaultViewSpec = {
  profile: string;
  policy: SessionPolicy;
};

export type BootstrapResponse = {
  section: SectionSpec;
  defaultView: DefaultViewSpec;
  rawRoutes: RouteRef[];
};

export type SessionSummary = {
  id: string;
  profile: string;
  createdAt: string;
  cellCount: number;
  bindingCount: number;
  policy: SessionPolicy;
  bindings?: BindingView[];
  history?: HistoryEntry[];
};

export type ProfileName = "raw" | "interactive" | "persistent";

export type ProfileSpec = {
  id: ProfileName;
  title: string;
  summary: string;
  policy: SessionPolicy;
  highlights: string[];
};

export type ProfilesBootstrapResponse = {
  section: SectionSpec;
  selectedProfile: ProfileName;
  profiles: ProfileSpec[];
  rawRoutes: RouteRef[];
};

export type ExampleSourceSpec = {
  id: string;
  label: string;
  source: string;
  rationale: string;
};

export type EvaluationBootstrapResponse = {
  section: SectionSpec;
  defaultProfile: ProfileName;
  starterSource: string;
  examples: ExampleSourceSpec[];
  rawRoutes: RouteRef[];
};

export type PersistenceBootstrapResponse = {
  section: SectionSpec;
  seedSources: ExampleSourceSpec[];
  rawRoutes: RouteRef[];
};

export type TimeoutBootstrapResponse = {
  section: SectionSpec;
  scenarios: ExampleSourceSpec[];
  rawRoutes: RouteRef[];
};

export type EvaluateResponse = {
  session: SessionSummary;
  cell: CellReport;
};

export type CellReport = {
  id: number;
  createdAt: string;
  source: string;
  static: StaticReport;
  rewrite: RewriteReport;
  execution: ExecutionReport;
  runtime: RuntimeReport;
};

export type ExecutionReport = {
  status: string;
  result: string;
  error?: string;
  durationMs: number;
  awaited: boolean;
  console: ConsoleEvent[];
  hadSideEffects: boolean;
  helperError: boolean;
};

export type ConsoleEvent = {
  kind: string;
  message: string;
};

export type StaticReport = {
  diagnostics: DiagnosticView[];
  topLevelBindings: TopLevelBindingView[];
  unresolved: IdentifierUseView[];
  astNodeCount: number;
  summary: StaticSummaryFact[];
};

export type StaticSummaryFact = {
  label: string;
  value: string;
};

export type RewriteReport = {
  mode: string;
  declaredNames: string[];
  helperNames: string[];
  lastHelperName: string;
  bindingHelperName: string;
  capturedLastExpr: boolean;
  transformedSource: string;
  operations: RewriteStep[];
  warnings?: string[];
  finalExpressionSource?: string;
};

export type RewriteStep = {
  kind: string;
  detail: string;
};

export type RuntimeReport = {
  diffs: GlobalDiffView[];
  newBindings: string[];
  updatedBindings: string[];
  removedBindings: string[];
  leakedGlobals: string[];
  persistedByWrap: string[];
  currentCellValue: string;
};

export type GlobalDiffView = {
  name: string;
  change: string;
  before?: string;
  after?: string;
  beforeKind?: string;
  afterKind?: string;
  sessionBound: boolean;
};

export type DiagnosticView = {
  severity: string;
  message: string;
};

export type TopLevelBindingView = {
  name: string;
  kind: string;
  line: number;
  snippet: string;
  extends?: string;
  referenceCount: number;
};

export type IdentifierUseView = {
  line: number;
  col: number;
  context?: string;
  nodeId: number;
  snippet?: string;
};

export type HistoryEntry = {
  cellId: number;
  createdAt: string;
  sourcePreview: string;
  resultPreview: string;
  status: string;
};

export type BindingView = {
  name: string;
  kind: string;
  origin: string;
  declaredInCell: number;
  lastUpdatedCell: number;
  runtime: BindingRuntimeView;
};

export type BindingRuntimeView = {
  valueKind: string;
  preview: string;
};

export type SessionRecord = {
  sessionId: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
  engineKind: string;
  metadataJson?: string;
};

export type HistoryResponse = {
  history: EvaluationRecord[];
};

export type EvaluationRecord = {
  evaluationId: number;
  sessionId: string;
  cellId: number;
  createdAt: string;
  rawSource: string;
  rewrittenSource: string;
  ok: boolean;
  resultJson?: unknown;
  errorText?: string;
};

export type SessionExport = {
  session: SessionRecord;
  evaluations: EvaluationRecord[];
};
