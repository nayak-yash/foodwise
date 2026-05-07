"use client";

import { useState } from "react";
import { ArrowRight, LoaderCircle, UtensilsCrossed } from "lucide-react";

export default function Login() {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function signIn(form: FormData) {
    setBusy(true);
    setError("");
    try {
      const response = await fetch("/api/session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: form.get("username"), password: form.get("password") }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sign-in failed");
      window.location.assign("/");
    } catch (error) {
      setError((error as Error).message);
      setBusy(false);
    }
  }

  return (
    <div className="login-page">
      <div className="login-introduction">
        <span className="brand-mark">
          <UtensilsCrossed size={26} />
        </span>
        <div className="eyebrow">FOODWISE · OPERATIONS INTELLIGENCE</div>
        <h1>
          A clearer picture.
          <br />A calmer service.
        </h1>
        <p>Your intelligent operations partner, from the first signal to verified recovery.</p>
      </div>
      <section className="modal login-card">
        <div className="panel-heading">
          <div>
            <div className="section-label">YOUR RESTAURANT WORKSPACE</div>
            <h2>Welcome to Foodwise.</h2>
          </div>
        </div>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void signIn(new FormData(event.currentTarget));
          }}
        >
          <label>
            Username
            <input name="username" autoComplete="username" required autoFocus />
          </label>
          <label>
            Password
            <input name="password" type="password" autoComplete="current-password" required />
          </label>
          {error && (
            <p role="alert" className="message error">
              {error}
            </p>
          )}
          <button className="button primary" disabled={busy}>
            {busy ? <LoaderCircle size={16} className="spin" /> : <ArrowRight size={16} />}Open
            workspace
          </button>
          <p className="muted">Local demo: manager / restaurantops</p>
        </form>
      </section>
    </div>
  );
}
