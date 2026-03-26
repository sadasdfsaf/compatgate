"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useMemo, useState } from "react";

import { clearDevUser, getRememberedDevUser, setDevUser } from "@/lib/api-client";

export default function SignInPage() {
  const router = useRouter();
  const [username, setUsername] = useState(() => getRememberedDevUser());
  const [message, setMessage] = useState("");

  const canSubmit = useMemo(() => username.trim().length > 0, [username]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSubmit) return;
    setDevUser(username.trim());
    router.push("/projects");
  }

  function handleClear() {
    clearDevUser();
    setMessage("Cleared the dev session from local storage.");
  }

  return (
    <main className="grid two">
      <section className="panel">
        <div className="pill">Development sign in</div>
        <h1 className="page-title">Choose how you want to open the dashboard.</h1>
        <p className="muted">
          CompatGate supports a fast local development sign in and an optional GitHub OAuth flow. In both cases the dashboard reads data from the Go API.
        </p>
        <form className="grid" onSubmit={handleSubmit}>
          <div className="field">
            <label htmlFor="username">Dev username</label>
            <input id="username" name="username" value={username} onChange={(event) => setUsername(event.target.value)} placeholder="compatgate-dev" />
          </div>
          <div className="actions">
            <button className="button" type="submit" disabled={!canSubmit}>
              Continue in development mode
            </button>
            <button className="ghost-button" type="button" onClick={handleClear}>
              Clear local session
            </button>
          </div>
        </form>
        {message ? <div className="footer-note">{message}</div> : null}
        <div className="actions">
          <Link className="ghost-button" href="/api/auth/github/start">
            Try GitHub OAuth
          </Link>
        </div>
      </section>

      <section className="panel">
        <h2>How auth works</h2>
        <div className="grid">
          <div className="card">
            <h3>Development mode</h3>
            <p className="muted">Stores a local viewer in the browser and sends it as <span className="code">X-CompatGate-User</span> to the Go API.</p>
          </div>
          <div className="card">
            <h3>GitHub OAuth</h3>
            <p className="muted">Works when <span className="code">GITHUB_CLIENT_ID</span>, <span className="code">GITHUB_CLIENT_SECRET</span>, and <span className="code">COMPATGATE_WEB_URL</span> are configured.</p>
          </div>
          <div className="card">
            <h3>Recommended first step</h3>
            <p className="muted">
              Sign in here, create a project on <Link href="/projects">/projects</Link>, then use the generated <span className="code">projectId</span> and <span className="code">projectToken</span> in `compatgate upload`.
            </p>
          </div>
        </div>
      </section>
    </main>
  );
}
