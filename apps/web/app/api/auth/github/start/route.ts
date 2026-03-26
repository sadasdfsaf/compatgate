import { randomUUID } from "node:crypto";
import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const clientId = process.env.GITHUB_CLIENT_ID;
  const appUrl = process.env.COMPATGATE_WEB_URL;

  if (!clientId || !appUrl) {
    return NextResponse.redirect(new URL("/auth/signin", request.url));
  }

  const state = randomUUID();
  const redirectUrl = new URL("https://github.com/login/oauth/authorize");
  redirectUrl.searchParams.set("client_id", clientId);
  redirectUrl.searchParams.set("redirect_uri", `${appUrl.replace(/\/$/, "")}/api/auth/github/callback`);
  redirectUrl.searchParams.set("scope", "read:user user:email");
  redirectUrl.searchParams.set("state", state);

  const response = NextResponse.redirect(redirectUrl);
  response.cookies.set("compatgate_github_state", state, { httpOnly: true, sameSite: "lax", path: "/" });
  return response;
}
