import { NextResponse } from "next/server";

type GithubTokenResponse = { access_token?: string };
type GithubUser = { login: string };

export async function GET(request: Request) {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const cookieState = request.headers.get("cookie")?.match(/compatgate_github_state=([^;]+)/)?.[1];
  const clientId = process.env.GITHUB_CLIENT_ID;
  const clientSecret = process.env.GITHUB_CLIENT_SECRET;
  const appUrl = process.env.COMPATGATE_WEB_URL;

  if (!code || !state || !cookieState || state !== cookieState || !clientId || !clientSecret || !appUrl) {
    return NextResponse.redirect(new URL("/auth/signin", request.url));
  }

  const tokenResponse = await fetch("https://github.com/login/oauth/access_token", {
    method: "POST",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify({
      client_id: clientId,
      client_secret: clientSecret,
      code,
      redirect_uri: `${appUrl.replace(/\/$/, "")}/api/auth/github/callback`,
      state
    })
  });

  const token = (await tokenResponse.json()) as GithubTokenResponse;
  if (!token.access_token) {
    return NextResponse.redirect(new URL("/auth/signin", request.url));
  }

  const userResponse = await fetch("https://api.github.com/user", {
    headers: {
      Accept: "application/vnd.github+json",
      Authorization: `Bearer ${token.access_token}`,
      "User-Agent": "CompatGate"
    }
  });

  const user = (await userResponse.json()) as GithubUser;
  if (!user.login) {
    return NextResponse.redirect(new URL("/auth/signin", request.url));
  }

  const response = NextResponse.redirect(new URL("/projects", request.url));
  response.cookies.set("compatgate_user", user.login, { httpOnly: false, sameSite: "lax", path: "/" });
  response.cookies.set("compatgate_auth_source", "github", { httpOnly: false, sameSite: "lax", path: "/" });
  response.cookies.set("compatgate_github_state", "", { httpOnly: true, sameSite: "lax", path: "/", maxAge: 0 });
  return response;
}
