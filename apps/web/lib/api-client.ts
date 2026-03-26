import type {
  ApiEnvelope,
  CreateProjectPayload,
  Project,
  RunDetail,
  RunSummary,
  Viewer
} from "@/lib/types";

const DEV_USER_KEY = "compatgate.dev-user";
const DEV_USER_COOKIE = "compatgate_user";
const DEV_SOURCE_COOKIE = "compatgate_auth_source";
const API_BASE_URL =
  process.env.NEXT_PUBLIC_COMPATGATE_API_BASE_URL ?? "http://localhost:8080";
const DEFAULT_DEV_USER =
  process.env.NEXT_PUBLIC_COMPATGATE_DEV_USER ?? "compatgate-dev";

export function getApiBaseUrl() {
  return API_BASE_URL;
}

export function getDefaultDevUser() {
  return DEFAULT_DEV_USER;
}

export function getRememberedDevUser(): string {
  if (typeof window === "undefined") {
    return DEFAULT_DEV_USER;
  }
  return window.localStorage.getItem(DEV_USER_KEY)?.trim() || DEFAULT_DEV_USER;
}

function readCookie(name: string) {
  if (typeof document === "undefined") {
    return "";
  }
  return (
    document.cookie
      .split("; ")
      .find((entry) => entry.startsWith(`${name}=`))
      ?.split("=")[1] ?? ""
  );
}

export function getDevUser(): Viewer | null {
  if (typeof window === "undefined") {
    return null;
  }
  const login = decodeURIComponent(readCookie(DEV_USER_COOKIE));
  const source = decodeURIComponent(readCookie(DEV_SOURCE_COOKIE)) || "development";
  return login ? { login, source } : null;
}

export function setDevUser(login: string) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(DEV_USER_KEY, login);
    document.cookie = `${DEV_USER_COOKIE}=${encodeURIComponent(login)}; path=/; SameSite=Lax`;
    document.cookie = `${DEV_SOURCE_COOKIE}=development; path=/; SameSite=Lax`;
  }
}

export function clearDevUser() {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem(DEV_USER_KEY);
    document.cookie = `${DEV_USER_COOKIE}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
    document.cookie = `${DEV_SOURCE_COOKIE}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
  }
}

async function request<T>(
  path: string,
  viewer: Viewer,
  init?: RequestInit
): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  headers.set("X-CompatGate-User", viewer.login);
  if (init?.body) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
    cache: "no-store"
  });

  let payload: ApiEnvelope<T> | null = null;
  try {
    payload = (await response.json()) as ApiEnvelope<T>;
  } catch {
    payload = null;
  }

  if (!response.ok || payload?.error) {
    throw new Error(
      payload?.error?.message ??
        `CompatGate API request failed (${response.status})`
    );
  }

  if (!payload || payload.data === null) {
    throw new Error("CompatGate API returned an empty response");
  }

  return payload.data;
}

export function listProjects(viewer: Viewer) {
  return request<Project[]>("/api/v1/projects", viewer);
}

export function getProject(viewer: Viewer, projectId: string) {
  return request<Project>(`/api/v1/projects/${projectId}`, viewer);
}

export function createProject(viewer: Viewer, payload: CreateProjectPayload) {
  return request<Project>("/api/v1/projects", viewer, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listRuns(viewer: Viewer, projectId: string) {
  return request<RunSummary[]>(`/api/v1/projects/${projectId}/runs`, viewer);
}

export function getRun(viewer: Viewer, projectId: string, runId: string) {
  return request<RunDetail>(`/api/v1/projects/${projectId}/runs/${runId}`, viewer);
}
