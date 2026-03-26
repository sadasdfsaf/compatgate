export type Protocol = "openapi" | "graphql" | "grpc" | "asyncapi";
export type Severity = "info" | "warn" | "error";

export type Viewer = {
  login: string;
  source?: string;
};

export type ApiError = {
  code: string;
  message: string;
};

export type ApiEnvelope<T> = {
  data: T | null;
  error?: ApiError | null;
};

export type RunSummary = {
  id: string;
  projectId: string;
  status: string;
  protocol: Protocol;
  createdAt: string;
  findingCount: number;
  breakingCount: number;
  repository?: string;
  sha?: string;
  ref?: string;
};

export type Project = {
  id: string;
  name: string;
  repository?: string;
  defaultProtocol?: Protocol;
  owner: string;
  projectToken?: string;
  createdAt: string;
  latestRun?: RunSummary;
};

export type Finding = {
  protocol: Protocol;
  rule_id: string;
  severity: Severity;
  breaking: boolean;
  resource: string;
  message: string;
  before?: unknown;
  after?: unknown;
  source_location?: {
    file: string;
    line?: number;
    column?: number;
  };
  labels?: Record<string, string>;
};

export type Report = {
  protocols: Protocol[];
  summary: {
    finding_count: number;
    breaking_count: number;
    error_count: number;
    warn_count: number;
    info_count: number;
  };
  findings: Finding[];
  meta: {
    base: string;
    revision: string;
    generated_at: string;
    labels?: Record<string, string>;
  };
};

export type RunDetail = {
  run: RunSummary;
  report: Report;
};

export type CreateProjectPayload = {
  name: string;
  repository?: string;
  defaultProtocol?: Protocol;
};
