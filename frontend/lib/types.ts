export type View = "overview" | "incidents" | "inventory" | "forecasts";

export interface InventoryItem {
  id: string;
  name: string;
  unit: string;
  balance: number;
  reorder_point: number;
}

export interface PurchaseOrder {
  id: string;
  ingredient: string;
  supplier: string;
  quantity: number;
  received_quantity: number;
  outstanding: number;
  status: string;
  external_status: string;
}

export interface Hypothesis {
  name: string;
  assessment: string;
  evidence: string[];
  sources: string[];
}

export interface Diagnosis {
  root_cause?: string;
  summary?: string;
  mode?: string;
  missing_kg?: number;
  hypotheses?: Hypothesis[];
  limitations?: string[];
  model_analysis?: { explanation: string; additional_checks: string[] };
  retrieval?: {
    mode: string;
    documents: { source: string; content: string; score: number }[];
  };
}

export interface RecoveryAction {
  id: string;
  status: string;
  type: string;
  parameters: { quantity: number; reason: string };
}

export interface Recovery {
  checks?: Record<string, boolean>;
  observation_seconds?: number;
  required_window_seconds?: number;
}

export interface Incident {
  id: string;
  title: string;
  status: string;
  severity: string;
  created_at: string;
  diagnosis: Diagnosis;
  recovery: Recovery;
  actions?: RecoveryAction[];
  timeline?: { id: number; event: string; actor: string; created_at: string }[];
}

export interface StockRequest {
  id: string;
  incident_id: string;
  quantity: number;
  ingredient: string;
  status: string;
}

export interface DemandForecast {
  ingredient_id: string;
  ingredient: string;
  weekday: string;
  expected_kg: number;
  balance_kg: number;
  recommended_reorder_kg: number;
  planning_upper_kg: number;
  services: { service: string; expected_kg: number }[];
  backtest: {
    evaluated_days: number;
    weekday_mae_kg: number;
    seven_day_mean_mae_kg: number;
  };
  note: string;
}

export interface Overview {
  name: string;
  active_incidents: number;
  pending_approvals: number;
  resolved_incidents: number;
  prep_minutes: number;
  cancellation_rate: number;
  pending_events: number;
}

export interface Catalog {
  suppliers: { id: string; name: string }[];
  menu: { id: string; name: string; price: number }[];
}

export interface AgentStatus {
  ready: boolean;
  reasoning_mode: string;
  retrieval_mode: string;
  verification_window_seconds: number;
  last_error: string | null;
}

export interface DashboardData {
  overview: Overview;
  inventory: InventoryItem[];
  incidents: Incident[];
  purchaseOrders: PurchaseOrder[];
  requests: StockRequest[];
  catalog: Catalog;
}

export type RunAction = (
  name: string,
  task: () => Promise<unknown>,
  successMessage: string,
) => Promise<void>;
