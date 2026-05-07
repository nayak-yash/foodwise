import { CheckCheck, Clock3, Search, ShieldCheck } from "lucide-react";
import type { Overview } from "@/lib/types";

export function MetricsGrid({ overview }: { overview: Overview }) {
  return (
    <section className="stats-grid">
      <Stat
        label="Active investigations"
        value={String(overview.active_incidents)}
        icon={<Search size={18} />}
        note={overview.active_incidents ? "Evidence gathering in progress" : "No open disruptions"}
        accent={overview.active_incidents > 0}
      />
      <Stat
        label="Awaiting your approval"
        value={String(overview.pending_approvals)}
        icon={<ShieldCheck size={18} />}
        note="You’re in control of every action"
      />
      <Stat
        label="Kitchen preparation"
        value={String(overview.prep_minutes ?? "—")}
        suffix="min"
        icon={<Clock3 size={18} />}
        note={
          overview.prep_minutes > 18
            ? "Above the 18 min recovery target"
            : "Within the recovery target"
        }
        accent={overview.prep_minutes > 18}
      />
      <Stat
        label="Verified recoveries"
        value={String(overview.resolved_incidents)}
        icon={<CheckCheck size={18} />}
        note="Closed with measured evidence"
      />
    </section>
  );
}

function Stat({
  label,
  value,
  suffix,
  icon,
  note,
  accent,
}: {
  label: string;
  value: string;
  suffix?: string;
  icon: React.ReactNode;
  note: string;
  accent?: boolean;
}) {
  return (
    <article className="stat">
      <div className="stat-top">
        <span>{label}</span>
        <span className={accent ? "stat-icon accent" : "stat-icon"}>{icon}</span>
      </div>
      <div className="stat-value">
        {value}
        <span>{suffix}</span>
      </div>
      <div className={accent ? "stat-note orange-text" : "stat-note"}>
        <span className={accent ? "tiny-dot orange" : "tiny-dot"} />
        {note}
      </div>
    </article>
  );
}
