import { LoaderCircle, Play } from "lucide-react";
import type { View } from "@/lib/types";

const copy: Record<View, { title: string; description: string }> = {
  overview: {
    title: "Your kitchen, in focus.",
    description: "One connected view of what’s happening, why, and what comes next.",
  },
  incidents: {
    title: "Follow the evidence.",
    description: "Investigations that connect symptoms, test causes, and track recovery.",
  },
  inventory: {
    title: "Stay a step ahead of stock.",
    description: "Real receipts, recipe-based consumption, and a traceable stock ledger.",
  },
  forecasts: {
    title: "Plan for the next service.",
    description: "Weekday-aware ingredient forecasts, grounded in your operating history.",
  },
};

export function PageHeading({
  view,
  busy,
  loaded,
  onDemo,
}: {
  view: View;
  busy: string;
  loaded: boolean;
  onDemo: () => void;
}) {
  return (
    <div className="page-heading">
      <div className="heading-copy">
        <div className="eyebrow">A CLEARER PICTURE. A CALMER SERVICE.</div>
        <h1>{copy[view].title}</h1>
        <p>{copy[view].description}</p>
      </div>
      <button className="button primary" disabled={!!busy || !loaded} onClick={onDemo}>
        {busy === "demo" ? <LoaderCircle className="spin" size={16} /> : <Play size={15} />}Run
        incident demo
      </button>
    </div>
  );
}
