import type { Metadata } from "next";
import { cookies } from "next/headers";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "CompatGate",
  description: "Open-source API compatibility governance across OpenAPI, GraphQL, gRPC, and AsyncAPI."
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const store = await cookies();
  const login = store.get("compatgate_user")?.value;
  const source = store.get("compatgate_auth_source")?.value;

  return (
    <html lang="en">
      <body>
        <div className="shell">
          <header className="topbar">
            <Link href="/" className="brand">
              <span className="brand-mark">C</span>
              <span>CompatGate</span>
            </Link>
            <nav className="nav">
              <Link className="ghost-button" href="/projects">
                Projects
              </Link>
              {login ? (
                <>
                  <span className="pill">{source || "session"}: {login}</span>
                  <form action="/api/auth/logout" method="post">
                    <button className="ghost-button" type="submit">
                      Sign out
                    </button>
                  </form>
                </>
              ) : (
                <Link className="ghost-button" href="/auth/signin">
                  Sign in
                </Link>
              )}
            </nav>
          </header>
          {children}
        </div>
      </body>
    </html>
  );
}
