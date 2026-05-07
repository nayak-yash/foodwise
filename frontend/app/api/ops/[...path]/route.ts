import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const allowed = new Set([
    "overview",
    "inventory",
    "catalog",
    "purchase-orders",
    "incidents",
    "stock-requests",
    "actions",
    "orders",
    "waste",
    "kitchen",
    "demo",
    "forecasts",
    "status",
  ]);
  if (!allowed.has(path[0]) || path.some((part) => part === "." || part === ".."))
    return NextResponse.json({ error: "Unknown API route" }, { status: 404 });
  if (request.method !== "GET") {
    const origin = request.headers.get("origin");
    if (!origin || new URL(origin).host !== request.headers.get("host"))
      return NextResponse.json({ error: "Same-origin request required" }, { status: 403 });
  }
  const intelligence = ["forecasts", "status"].includes(path[0]);
  if (intelligence && request.method !== "GET")
    return NextResponse.json({ error: "Method not allowed" }, { status: 405 });
  const base = intelligence ? process.env.INTELLIGENCE_URL : `${process.env.BACKEND_URL}/api/v1`;
  const token = intelligence ? process.env.AGENT_API_TOKEN : process.env.MANAGER_API_TOKEN;
  try {
    const response = await fetch(
      `${base}/${path.map(encodeURIComponent).join("/")}${request.nextUrl.search}`,
      {
        method: request.method,
        cache: "no-store",
        signal: AbortSignal.timeout(25000),
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
          "Idempotency-Key": request.headers.get("idempotency-key") || "",
        },
        body: request.method === "GET" ? undefined : await request.text(),
      },
    );
    return new NextResponse(await response.text(), {
      status: response.status,
      headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json(
      { error: "Service unavailable. Check container health and retry." },
      { status: 503 },
    );
  }
}
export const GET = proxy;
export const POST = proxy;
