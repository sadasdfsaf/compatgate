import Link from "next/link";

export default function HomePage() {
  return (
    <main className="grid">
      <section className="hero">
        <div className="pill">Minimal runnable Next.js app</div>
        <h1>Browse CompatGate findings from the Go API with a lightweight web console.</h1>
        <p className="muted">
          This build uses a local dev user and sends it to the backend through
          <span className="code"> X-CompatGate-User</span>. It is intentionally
          small so you can run it right now.
        </p>
        <div className="hero-actions">
          <Link className="button" href="/projects">
            Open projects
          </Link>
          <Link className="ghost-button" href="/auth/signin">
            Configure dev login
          </Link>
        </div>
      </section>
    </main>
  );
}
