import { Search, Sparkles } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Loading } from "@/components/ui/feedback";
import { formatLabel, formatTime } from "@/lib/format";
import type { Diagnosis, Incident, RunAction, StockRequest } from "@/lib/types";
import { RecoveryPlan } from "./recovery-plan";

export function IncidentWorkspace({
  incident,
  requests,
  busy,
  runAction,
  refresh,
}: {
  incident: Incident;
  requests: StockRequest[];
  busy: string;
  runAction: RunAction;
  refresh: () => Promise<void>;
}) {
  return (
    <section className="panel detail-panel">
      <div className="panel-heading">
        <div>
          <div className="section-label">
            INCIDENT WORKSPACE <span className="mono">#{incident.id.slice(0, 8)}</span>
          </div>
          <h2>{incident.title}</h2>
        </div>
        <Badge value={incident.status} />
      </div>
      <div className="lifecycle">
        {["detected", "investigating", "awaiting_approval", "verifying", "resolved"].map(
          (stage, index) => (
            <div
              key={stage}
              className={
                incident.status === stage ||
                (stage === "awaiting_approval" && incident.status === "approved")
                  ? "stage current"
                  : "stage"
              }
            >
              <span>{index + 1}</span>
              {formatLabel(stage)}
            </div>
          ),
        )}
      </div>
      {incident.diagnosis.summary ? (
        <div className="detail-body">
          <DiagnosisReport diagnosis={incident.diagnosis} />
          <div className="detail-columns">
            <RecoveryPlan
              incident={incident}
              requests={requests}
              busy={busy}
              runAction={runAction}
              refresh={refresh}
            />
            <Timeline incident={incident} />
          </div>
          <KnowledgeSources diagnosis={incident.diagnosis} />
        </div>
      ) : (
        <Loading>Gathering evidence across your operations…</Loading>
      )}
    </section>
  );
}

function DiagnosisReport({ diagnosis }: { diagnosis: Diagnosis }) {
  return (
    <>
      <div className="diagnosis-heading">
        <span className="diagnosis-icon">
          <Search size={20} />
        </span>
        <div>
          <div className="section-label">EVIDENCE-BACKED FINDING</div>
          <h3>{diagnosis.root_cause}</h3>
          <p>{diagnosis.summary}</p>
        </div>
      </div>
      <div className="hypotheses">
        {diagnosis.hypotheses?.map((hypothesis) => (
          <article key={hypothesis.name} className="hypothesis">
            <div>
              <h4>{hypothesis.name}</h4>
              <Badge value={hypothesis.assessment} />
            </div>
            <ul>
              {hypothesis.evidence.map((evidence, index) => (
                <li key={index}>{evidence}</li>
              ))}
            </ul>
            <small>{hypothesis.sources.join(" · ")}</small>
          </article>
        ))}
      </div>
      {diagnosis.model_analysis && (
        <div className="ai-note">
          <Sparkles size={17} />
          <div>
            <strong>Model-assisted analysis</strong>
            <p>{diagnosis.model_analysis.explanation}</p>
          </div>
        </div>
      )}
    </>
  );
}

function Timeline({ incident }: { incident: Incident }) {
  return (
    <div>
      <h3 className="subheading">Investigation timeline</h3>
      <div className="timeline">
        {incident.timeline
          ?.slice(-10)
          .reverse()
          .map((event) => (
            <div className="timeline-entry" key={event.id}>
              <span className="timeline-dot" />
              <div>
                <strong>{formatLabel(event.event.replaceAll(".", " "))}</strong>
                <small>
                  {event.actor} · {formatTime(event.created_at)}
                </small>
              </div>
            </div>
          ))}
      </div>
    </div>
  );
}

function KnowledgeSources({ diagnosis }: { diagnosis: Diagnosis }) {
  return (
    <details className="sources">
      <summary>Sources, related knowledge & limitations</summary>
      {diagnosis.retrieval?.documents.map((document) => (
        <div key={document.source}>
          <strong>{document.source}</strong>
          <p>{document.content}</p>
        </div>
      ))}
      <ul>
        {diagnosis.limitations?.map((limitation) => (
          <li key={limitation}>{limitation}</li>
        ))}
      </ul>
    </details>
  );
}
