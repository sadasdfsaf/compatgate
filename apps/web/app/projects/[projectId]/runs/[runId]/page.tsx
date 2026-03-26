"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { getDevUser, getRun } from "@/lib/api-client";
import type { Finding, RunDetail, Viewer } from "@/lib/types";

export default function RunDetailPage() {
  const params = useParams<{ projectId: string; runId: string }>();
  const [viewer, setViewer] = useState<Viewer | null>(null);
  const [detail, setDetail] = useState<RunDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [breakingOnly, setBreakingOnly] = useState(false);
  const [search, setSearch] = useState("");

  useEffect(() => {
    const currentViewer = getDevUser();
    setViewer(currentViewer);
    if (!currentViewer || !params.projectId || !params.runId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    getRun(currentViewer, params.projectId, params.runId)
      .then((run) => {
        setDetail(run);
        setError("");
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [params.projectId, params.runId]);

  const findings = useMemo(() => {
    const items = detail?.report.findings ?? [];
    return items.filter((finding) => {
      if (breakingOnly && !finding.breaking) {
        return false;
      }
      if (!search.trim()) {
        return true;
      }
      const haystack = `${finding.rule_id} ${finding.resource} ${finding.message}`.toLowerCase();
      return haystack.includes(search.trim().toLowerCase());
    });
  }, [detail, breakingOnly, search]);

  return (
    <main className="grid">
      <section className="panel">
        <div className="actions">
          <Link className="ghost-button" href={`/projects/${params.projectId}`}>
            Back to project
          </Link>
        </div>
        {!viewer ? (
          <div className="empty">
            No dev user is set. Go to <Link href="/auth/signin">/auth/signin</Link>.
          </div>
        ) : null}
        {loading ? <div className="empty">Loading run...</div> : null}
        {error ? <div className="error-box">{error}</div> : null}
        {detail ? (
          <div className="grid three">
            <article className="metric">
              Protocol
              <strong>{detail.run.protocol}</strong>
              <span className="muted">{detail.run.status}</span>
            </article>
            <article className="metric">
              Findings
              <strong>{detail.report.summary.finding_count}</strong>
              <span className="muted">Breaking: {detail.report.summary.breaking_count}</span>
            </article>
            <article className="metric">
              Git
              <strong className="code">{detail.run.sha || "local"}</strong>
              <span className="muted code">{detail.run.ref || detail.run.repository || "manual upload"}</span>
            </article>
          </div>
        ) : null}
      </section>

      <section className="panel">
        <div className="filter-row">
          <div className="field" style={{ minWidth: 240 }}>
            <label htmlFor="search">Search findings</label>
            <input
              id="search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="rule id, resource, message"
            />
          </div>
          <button
            className={breakingOnly ? "button" : "ghost-button"}
            type="button"
            onClick={() => setBreakingOnly((current) => !current)}
          >
            {breakingOnly ? "Showing breaking only" : "Show breaking only"}
          </button>
        </div>

        {detail && findings.length === 0 ? (
          <div className="empty">No findings match the current filters.</div>
        ) : null}

        {detail && findings.length > 0 ? (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Severity</th>
                  <th>Breaking</th>
                  <th>Rule</th>
                  <th>Resource</th>
                  <th>Message</th>
                  <th>Source</th>
                </tr>
              </thead>
              <tbody>
                {findings.map((finding) => (
                  <FindingRow
                    key={`${finding.rule_id}-${finding.resource}-${finding.message}`}
                    finding={finding}
                  />
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    </main>
  );
}

function FindingRow({ finding }: { finding: Finding }) {
  const source = finding.source_location
    ? `${finding.source_location.file}${finding.source_location.line ? `:${finding.source_location.line}` : ""}`
    : "n/a";

  return (
    <tr>
      <td>{finding.severity}</td>
      <td>{finding.breaking ? "yes" : "no"}</td>
      <td className="code">{finding.rule_id}</td>
      <td className="code">{finding.resource}</td>
      <td>
        <div>{finding.message}</div>
        {finding.before || finding.after ? (
          <div className="footer-note code">
            before: {stringifyCompact(finding.before)} | after: {stringifyCompact(finding.after)}
          </div>
        ) : null}
      </td>
      <td className="code">{source}</td>
    </tr>
  );
}

function stringifyCompact(value: unknown) {
  if (value === undefined || value === null) {
    return "n/a";
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}
