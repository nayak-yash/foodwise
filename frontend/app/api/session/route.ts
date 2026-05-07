import { NextRequest, NextResponse } from "next/server";
import { SessionService } from "@/lib/server/session";

function sameOrigin(request: NextRequest) {
  const origin = request.headers.get("origin");
  try {
    return !!origin && new URL(origin).host === request.headers.get("host");
  } catch {
    return false;
  }
}

export async function POST(request: NextRequest) {
  if (!sameOrigin(request))
    return NextResponse.json({ error: "Same-origin request required" }, { status: 403 });
  const credentials = await request.json().catch(() => null);
  const sessions = new SessionService();
  if (
    typeof credentials?.username !== "string" ||
    typeof credentials?.password !== "string" ||
    !sessions.authenticate(credentials.username, credentials.password)
  ) {
    return NextResponse.json({ error: "Incorrect username or password" }, { status: 401 });
  }
  const response = NextResponse.json({ status: "signed_in" });
  response.cookies.set(SessionService.cookieName, sessions.create(), {
    httpOnly: true,
    sameSite: "strict",
    secure: request.nextUrl.protocol === "https:",
    path: "/",
    maxAge: SessionService.lifetimeSeconds,
  });
  return response;
}

export async function DELETE(request: NextRequest) {
  if (!sameOrigin(request))
    return NextResponse.json({ error: "Same-origin request required" }, { status: 403 });
  const response = NextResponse.json({ status: "signed_out" });
  response.cookies.delete(SessionService.cookieName);
  return response;
}
