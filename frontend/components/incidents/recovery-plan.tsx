import { Check, Clock3, FlaskConical, LoaderCircle, PackageCheck, Play } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { formatLabel } from "@/lib/format";
import { operationsClient } from "@/lib/operations-client";
import type { Incident, Recovery, RunAction, StockRequest } from "@/lib/types";

export function RecoveryPlan({
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
    <div>
      <h3 className="subheading">Recovery plan</h3>
      {incident.actions?.map((action) => (
        <div className="action-card" key={action.id}>
          <div className="action-top">
            <PackageCheck size={21} />
            <strong>Source {action.parameters.quantity} kg replacement stock</strong>
            <Badge value={action.status} />
          </div>
          <p>{action.parameters.reason}</p>
          <small>
            Execution creates an internal stock request. Actual receipt is recorded separately.
          </small>
          {action.status === "proposed" && (
            <div className="action-buttons">
              <button
                disabled={!!busy}
                className="button primary"
                onClick={() =>
                  void runAction(
                    "approve",
                    () => operationsClient.decideAction(action.id, "approve"),
                    "Approved. The agent will execute the stock request.",
                  )
                }
              >
                <Check size={15} />
                Approve action
              </button>
              <button
                disabled={!!busy}
                className="button secondary"
                onClick={() =>
                  void runAction(
                    "reject",
                    () => operationsClient.decideAction(action.id, "reject"),
                    "Action rejected. Incident escalated for manager review.",
                  )
                }
              >
                Reject
              </button>
            </div>
          )}
          {requests
            .filter(
              (request) => request.incident_id === incident.id && request.status === "requested",
            )
            .map((request) => (
              <div className="demo-receipt" key={request.id}>
                <FlaskConical size={17} />
                <div>
                  <strong>Complete the demo recovery</strong>
                  <p>
                    Simulates a {request.quantity} kg receipt and three healthy measurement windows
                    over 10 seconds.
                  </p>
                  <button
                    className="button secondary"
                    disabled={!!busy}
                    onClick={() =>
                      void runAction(
                        "recover",
                        () => operationsClient.simulateRecovery(request, refresh),
                        "Synthetic receipt and kitchen measurements recorded. Verification will update shortly.",
                      )
                    }
                  >
                    {busy === "recover" ? (
                      <LoaderCircle className="spin" size={14} />
                    ) : (
                      <Play size={14} />
                    )}
                    Simulate receipt & recovery
                  </button>
                </div>
              </div>
            ))}
        </div>
      ))}
      {!incident.actions?.length && (
        <p className="muted">The investigator is preparing a recovery recommendation.</p>
      )}
      {incident.recovery.checks && <RecoveryChecklist recovery={incident.recovery} />}
    </div>
  );
}

function RecoveryChecklist({ recovery }: { recovery: Recovery }) {
  return (
    <div className="recovery-checks">
      <h3 className="subheading">Recovery verification</h3>
      {Object.entries(recovery.checks || {}).map(([key, passed]) => (
        <div key={key}>
          <span className={passed ? "check-pass" : "check-pending"}>
            {passed ? <Check size={14} /> : <Clock3 size={14} />}
          </span>
          {formatLabel(key)}
          <span>{passed ? "Confirmed" : "Waiting"}</span>
        </div>
      ))}
      <p className="muted">
        Observation: {recovery.observation_seconds}s / {recovery.required_window_seconds}s minimum
      </p>
    </div>
  );
}
