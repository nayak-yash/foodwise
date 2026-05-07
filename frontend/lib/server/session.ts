import { createHash, createHmac, timingSafeEqual } from "node:crypto";

/** Signs short-lived manager sessions; no API credential is exposed to the browser. */
export class SessionService {
  static readonly cookieName = "foodwise_session";
  static readonly lifetimeSeconds = 8 * 60 * 60;

  private static equal(left: string, right: string) {
    return timingSafeEqual(
      createHash("sha256").update(left).digest(),
      createHash("sha256").update(right).digest(),
    );
  }

  private signature(payload: string) {
    const secret = process.env.MANAGER_API_TOKEN;
    if (!secret) throw new Error("Manager session secret is not configured");
    return createHmac("sha256", secret).update(payload).digest("base64url");
  }

  authenticate(username: string, password: string) {
    const expectedUsername = process.env.DASHBOARD_USERNAME;
    const expectedPassword = process.env.DASHBOARD_PASSWORD;
    if (!expectedUsername || !expectedPassword) return false;
    return (
      SessionService.equal(username, expectedUsername) &&
      SessionService.equal(password, expectedPassword)
    );
  }

  create() {
    const payload = Buffer.from(
      JSON.stringify({ expires: Date.now() + SessionService.lifetimeSeconds * 1000 }),
    ).toString("base64url");
    return `${payload}.${this.signature(payload)}`;
  }

  verify(token?: string) {
    if (!token || token.length > 1024) return false;
    try {
      const [payload, signature, extra] = token.split(".");
      if (
        !payload ||
        !signature ||
        extra ||
        !SessionService.equal(signature, this.signature(payload))
      )
        return false;
      const session = JSON.parse(Buffer.from(payload, "base64url").toString());
      return typeof session.expires === "number" && session.expires > Date.now();
    } catch {
      return false;
    }
  }
}
