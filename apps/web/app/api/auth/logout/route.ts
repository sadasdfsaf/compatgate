import { NextResponse } from "next/server";

export async function POST(request: Request) {
  const response = NextResponse.redirect(new URL("/auth/signin", request.url));
  response.cookies.set("compatgate_user", "", { httpOnly: false, sameSite: "lax", path: "/", maxAge: 0 });
  response.cookies.set("compatgate_auth_source", "", { httpOnly: false, sameSite: "lax", path: "/", maxAge: 0 });
  return response;
}
