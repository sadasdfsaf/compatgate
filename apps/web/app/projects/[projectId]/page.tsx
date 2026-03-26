"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

import { getDevUser, getProject, listRuns } from "@/lib/api-client";
import type { Project, RunSummary, Viewer } from "@/lib/types";

export default function ProjectDetailPage() {
  const params = useParams<{ projectId: string }>();
  const projectId = params.projectId;
  const [viewer, setViewer] = useState<Viewer | null>(null);
  const [project, setProject] = useState<Project | null>(null);
  const [runs, setRuns] = useState<RunSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const currentViewer = getDevUser();
    setViewer(currentViewer);
    if (!currentViewer || !projectId) {
      setLoading(false);
      return;
    }

    setLoading(true);
    Promise.all([getProject(currentViewer, projectId), listRuns(currentViewer, projectId)])
      .then(([projectData, runData]) => {
        setProject(projectData);
        setRuns(runData);
        setError("");
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [projectId]);

  return (
    <main className="grid">
      <section className="panel">
        <div className="actions">
          <Link className="ghost-button" href="/projects">
            Back to projects
          </Link>
        </div>
        {!viewer ? (
          <div className="empty">
            No dev user is set. Go to <Link href="/auth/signin">/auth/signin</Link>.
          </div>
        ) : null}
        {loading ? <div className="empty">Loading project...</div> : null}
        {error ? <div className="error-box">{error}</div> : null}
        {project ? (
          <div className="grid two">
            <article className="metric">
              Project
              <strong>{project.name}</strong>
              <span className="muted code">{project.repository || "No repository"}</span>
            </article>
            <article className="metric">
              Default protocol
              <strong>{project.defaultProtocol || "not set"}</strong>
              <span className="muted">Project ID: {project.id}</span>
            </article>
          </div>
        ) : null}
      </section>

      <section className="panel">
        <h2>Run history</h2>
        {project && runs.length === 0 ? (
          <div className="empty">No uploaded runs for this project yet.</div>
        ) : null}
        {runs.length > 0 ? (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Created</th>
                  <th>Protocol</th>
                  <th>Status</th>
                  <th>Findings</th>
                  <th>Breaking</th>
                  <th>Open</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr key={run.id}>
                    <td>{new Date(run.createdAt).toLocaleString()}</td>
                    <td>{run.protocol}</td>
                    <td>{run.status}</td>
                    <td>{run.findingCount}</td>
                    <td>{run.breakingCount}</td>
                    <td>
                      <Link
                        className="ghost-button"
                        href={`/projects/${projectId}/runs/${run.id}`}
                      >
                        View report
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    </main>
  );
}
