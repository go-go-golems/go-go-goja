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
};
