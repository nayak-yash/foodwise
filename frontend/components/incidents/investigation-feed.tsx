import { Activity, ArrowRight, CheckCheck, CircleDot, ShieldCheck, Sparkles } from "lucide-react";
import { formatTime } from "@/lib/format";
import type { AgentStatus, Incident } from "@/lib/types";
import { Badge } from "@/components/ui/badge";

export function InvestigationFeed({
  incidents,
  selected,
  onSelect,
  pendingEvents,
  agent,
}: {
  incidents: Incident[];
  selected: string | null;
  onSelect: (id: string) => void;
  pendingEvents: number;
  agent: AgentStatus | null;
}) {
  return (
    <div className="investigation-layout">
      <section className="panel investigations">
        <div className="panel-heading">
          <div>
            <div className="section-label">OPERATIONS FEED</div>
            <h2>
              Investigations <span className="count">{incidents.length}</span>
            </h2>
          </div>
          <span className="live-label">
            <span className="live-dot" />
            Live
          </span>
        </div>
        {incidents.length === 0 ? (
          <EmptyFeed />
        ) : (
          <div className="incident-list">
            {incidents.map((incident) => (
              <button
                key={incident.id}
                onClick={() => onSelect(incident.id)}
                className={`incident-item ${selected === incident.id ? "chosen" : ""}`}
              >
                <div
                  className={`incident-icon ${incident.status === "resolved" ? "resolved" : ""}`}
                >
                  {incident.status === "resolved" ? (
                    <CheckCheck size={19} />
                  ) : (
                    <Activity size={19} />
                  )}
                </div>
                <div className="incident-copy">
                  <div className="incident-title">{incident.title}</div>
                  <p>
                    {incident.diagnosis.summary
                      ? `${incident.diagnosis.missing_kg} kg missing · Receiving discrepancy`
                      : "Correlating receiving and inventory records"}
                  </p>
                  <div className="incident-meta">
                    <Badge value={incident.status} />
                    <span>{formatTime(incident.created_at)}</span>
                  </div>
                </div>
                <ArrowRight size={16} />
              </button>
            ))}
          </div>
        )}
        <div className="panel-footer">
          <span>
            <CircleDot size={13} />
            Events → evidence → action
          </span>
          <span>{pendingEvents} queued events</span>
        </div>
      </section>
      <AgentCard agent={agent} />
    </div>
  );
}

function EmptyFeed() {
  return (
    <div className="empty-state">
      <span className="empty-icon">
        <ShieldCheck size={31} />
      </span>
      <h3>Ready when things get busy.</h3>
      <p>
        Your investigator is listening for operational events. Run the demo to follow a short
        delivery from detection to recovery.
      </p>
      <div className="empty-flow">
        <span>Detect</span>
        <ArrowRight size={13} />
        <span>Investigate</span>
        <ArrowRight size={13} />
        <span>Recover</span>
      </div>
    </div>
  );
}

function AgentCard({ agent }: { agent: AgentStatus | null }) {
  return (
    <section className="panel investigator-card">
      <div className="agent-graphic">
        <div className="orbit orbit-one" />
        <div className="orbit orbit-two" />
        <span className="spark-box">
          <Sparkles size={32} />
        </span>
        <span className="orbit-point p1" />
        <span className="orbit-point p2" />
      </div>
      <div className="section-label">YOUR OPERATIONS INVESTIGATOR</div>
      <h2>More than an alert.</h2>
      <p>Connected evidence. Competing hypotheses. A recovery plan you can actually follow.</p>
      <div className="agent-mode">
        <span>Reasoning</span>
        <strong>
          {agent?.reasoning_mode === "llm_assisted" ? "LLM-assisted" : "Deterministic evidence"}
        </strong>
      </div>
      <div className="agent-mode">
        <span>Knowledge retrieval</span>
        <strong>
          {agent?.retrieval_mode === "vector" ? "Vector similarity" : "Lexical / initializing"}
        </strong>
      </div>
      <div className="agent-mode">
        <span>Workflow memory</span>
        <strong>Persistent checkpoints</strong>
      </div>
    </section>
  );
}
