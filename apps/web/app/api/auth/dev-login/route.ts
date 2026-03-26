import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const user = url.searchParams.get("user") || process.env.NEXT_PUBLIC_COMPATGATE_DEV_USER || "compatgate-dev";
  const response = NextResponse.redirect(new URL("/projects", request.url));
  response.cookies.set("compatgate_user", user, { httpOnly: false, sameSite: "lax", path: "/" });
  response.cookies.set("compatgate_auth_source", "development", { httpOnly: false, sameSite: "lax", path: "/" });
  return response;
}
