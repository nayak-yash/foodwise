import { formatLabel } from "@/lib/format";

const green = new Set(["resolved", "healthy", "received", "supported", "fulfilled"]);
const amber = new Set(["awaiting_approval", "low_stock", "high", "proposed", "partial"]);
const red = new Set(["escalated", "rejected"]);

export function Badge({ value }: { value: string }) {
  const color = green.has(value)
    ? "green"
    : amber.has(value)
      ? "amber"
      : red.has(value)
        ? "red"
        : "neutral";
  return <span className={`badge ${color}`}>{formatLabel(value)}</span>;
}
