import { NextRequest, NextResponse } from "next/server";
import { SessionService } from "@/lib/server/session";

export function proxy(request: NextRequest) {
  const path = request.nextUrl.pathname;
  if (path === "/login" || path === "/api/session") return NextResponse.next();
  const sessions = new SessionService();
  if (sessions.verify(request.cookies.get(SessionService.cookieName)?.value)) {
    return NextResponse.next();
  }
  if (path.startsWith("/api/")) {
    return NextResponse.json({ error: "Manager sign-in required" }, { status: 401 });
  }
  return NextResponse.redirect(new URL("/login", request.url));
}

export const config = { matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"] };
