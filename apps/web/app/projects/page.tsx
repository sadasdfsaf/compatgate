"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { createProject, getApiBaseUrl, getDevUser, listProjects } from "@/lib/api-client";
import type { Project, Protocol, Viewer } from "@/lib/types";

const protocolOptions: Protocol[] = ["openapi", "graphql", "grpc", "asyncapi"];

export default function ProjectsPage() {
  const [viewer, setViewer] = useState<Viewer | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [createdProject, setCreatedProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [repository, setRepository] = useState("");
  const [defaultProtocol, setDefaultProtocol] = useState<Protocol>("openapi");

  useEffect(() => {
    const currentViewer = getDevUser();
    setViewer(currentViewer);
    if (!currentViewer) {
      setLoading(false);
      return;
    }

    setLoading(true);
    listProjects(currentViewer)
      .then((items) => {
        setProjects(items);
        setError("");
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const canCreate = useMemo(() => name.trim().length > 0, [name]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!viewer || !canCreate) {
      return;
    }
    setCreating(true);
    try {
      const created = await createProject(viewer, {
        name: name.trim(),
        repository: repository.trim(),
        defaultProtocol
      });
      setProjects((current) => [created, ...current]);
      setCreatedProject(created);
      setName("");
      setRepository("");
      setDefaultProtocol("openapi");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
    } finally {
      setCreating(false);
    }
  }

  return (
    <main className="grid">
      <section className="panel">
        <div className="card-title">
          <div>
            <div className="pill">Projects</div>
            <h1 className="page-title">Browse uploaded reports by project.</h1>
          </div>
          {viewer ? <div className="muted">Viewer: {viewer.login}</div> : null}
        </div>
        {!viewer ? (
          <div className="empty">
            No dev user is set yet. Go to <Link href="/auth/signin">/auth/signin</Link>{" "}
            to create a local session first.
          </div>
        ) : null}
        {error ? <div className="error-box">{error}</div> : null}
      </section>

      {viewer ? (
        <section className="panel">
          <h2>Create a project</h2>
          <form className="grid" onSubmit={handleCreate}>
            <div className="form-grid">
              <div className="field">
                <label htmlFor="name">Name</label>
                <input
                  id="name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="CompatGate demo"
                />
              </div>
              <div className="field">
                <label htmlFor="repository">Repository</label>
                <input
                  id="repository"
                  value={repository}
                  onChange={(event) => setRepository(event.target.value)}
                  placeholder="owner/repo"
                />
              </div>
              <div className="field">
                <label htmlFor="protocol">Default protocol</label>
                <select
                  id="protocol"
                  value={defaultProtocol}
                  onChange={(event) => setDefaultProtocol(event.target.value as Protocol)}
                >
                  {protocolOptions.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="actions">
              <button className="button" type="submit" disabled={!canCreate || creating}>
                {creating ? "Creating..." : "Create project"}
              </button>
            </div>
          </form>
        </section>
      ) : null}

      {viewer && createdProject ? (
        <section className="panel">
          <div className="card-title">
            <div>
              <div className="pill">Next step</div>
              <h2>Upload your first report</h2>
            </div>
            <span className="muted">Created just now</span>
          </div>
          <div className="grid two">
            <article className="card">
              <h3>Project credentials</h3>
              <p className="muted">Use these values in the CLI upload step.</p>
              <div className="footer-note">
                <div><strong>projectId:</strong> <span className="code">{createdProject.id}</span></div>
                <div><strong>projectToken:</strong> <span className="code">{createdProject.projectToken || "returned by API"}</span></div>
              </div>
            </article>
            <article className="card">
              <h3>Upload command</h3>
              <pre className="code" style={{ whiteSpace: "pre-wrap" }}>{`compatgate upload \\
  --input ./compatgate-report.json \\
  --cloud-url ${getApiBaseUrl()} \\
  --project-id ${createdProject.id} \\
  --project-token ${createdProject.projectToken || "<token>"} \\
  --repository ${createdProject.repository || "owner/repo"}`}</pre>
            </article>
          </div>
        </section>
      ) : null}

      <section className="panel">
        <h2>Project list</h2>
        {loading ? <div className="empty">Loading projects...</div> : null}
        {!loading && viewer && projects.length === 0 ? (
          <div className="empty">
            No projects yet. Create one above or upload a report with the CLI.
          </div>
        ) : null}
        <div className="grid two">
          {projects.map((project) => (
            <Link key={project.id} href={`/projects/${project.id}`} className="card-link">
              <article className="card">
                <div className="card-title">
                  <h3>{project.name}</h3>
                  {project.defaultProtocol ? (
                    <span className="pill">{project.defaultProtocol}</span>
                  ) : null}
                </div>
                <div className="muted code">{project.repository || "No repository"}</div>
                <div className="footer-note">
                  {project.latestRun
                    ? `${project.latestRun.breakingCount} breaking / ${project.latestRun.findingCount} findings`
                    : "No uploaded runs yet"}
                </div>
              </article>
            </Link>
          ))}
        </div>
      </section>
    </main>
  );
}
