"use client";

import { useState } from "react";
import { AppShell } from "@/components/layout/app-shell";
import { PageHeading } from "@/components/dashboard/page-heading";
import { MetricsGrid } from "@/components/dashboard/metrics-grid";
import { InvestigationFeed } from "@/components/incidents/investigation-feed";
import { IncidentWorkspace } from "@/components/incidents/incident-workspace";
import { InventoryPanel } from "@/components/inventory/inventory-panel";
import { PurchaseOrders } from "@/components/inventory/purchase-orders";
import { OperationDialog } from "@/components/inventory/operation-dialog";
import { ForecastPanel } from "@/components/forecasting/forecast-panel";
import { Loading, Notice } from "@/components/ui/feedback";
import { useOperations } from "@/hooks/use-operations";
import { operationsClient } from "@/lib/operations-client";
import type { View } from "@/lib/types";

export default function Dashboard() {
  const [view, setView] = useState<View>("overview");
  const [operation, setOperation] = useState("");
  const operations = useOperations();
  const { data, agent, busy, detail, forecasts, runAction, refresh } = operations;
  const showInvestigations = view === "overview" || view === "incidents";

  return (
    <AppShell
      view={view}
      onNavigate={setView}
      activeIncidents={data?.overview.active_incidents || 0}
      agent={agent}
    >
      <PageHeading
        view={view}
        busy={busy}
        loaded={!!data}
        onDemo={() =>
          void runAction(
            "demo",
            () => operationsClient.runDemo(),
            "Demo event recorded. The investigator is checking the evidence.",
          )
        }
      />
      <Notice message={operations.error} error onDismiss={operations.dismissError} />
      <Notice message={operations.notice} onDismiss={operations.dismissNotice} />
      {!data ? (
        <Loading>Connecting your operations…</Loading>
      ) : (
        <>
          {showInvestigations && (
            <>
              <MetricsGrid overview={data.overview} />
              <InvestigationFeed
                incidents={data.incidents}
                selected={operations.selected}
                onSelect={operations.selectIncident}
                pendingEvents={data.overview.pending_events}
                agent={agent}
              />
              {detail && (
                <IncidentWorkspace
                  incident={detail}
                  requests={data.requests}
                  busy={busy}
                  runAction={runAction}
                  refresh={refresh}
                />
              )}
            </>
          )}
          {(view === "overview" || view === "inventory") && (
            <InventoryPanel inventory={data.inventory} onOperation={setOperation} />
          )}
          {view === "inventory" && (
            <PurchaseOrders orders={data.purchaseOrders} onOperation={setOperation} />
          )}
          {(view === "overview" || view === "forecasts") && (
            <ForecastPanel forecasts={forecasts} showEvaluation={view === "forecasts"} />
          )}
          {operation && (
            <OperationDialog
              operation={operation}
              inventory={data.inventory}
              catalog={data.catalog}
              busy={busy}
              onClose={() => setOperation("")}
              runAction={runAction}
            />
          )}
        </>
      )}
    </AppShell>
  );
}
